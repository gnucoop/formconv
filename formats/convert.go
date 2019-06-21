package formats

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func Convert(xls *XlsForm) (*AjfForm, error) {
	err := checkTypes(xls.Survey)
	if err != nil {
		return nil, err
	}

	var ajf AjfForm
	var choicesMap map[string][]Choice
	ajf.ChoicesOrigins, choicesMap = buildChoicesOrigins(xls.Choices)
	err = checkChoicesRef(xls.Survey, choicesMap)
	if err != nil {
		return nil, err
	}

	survey, err := preprocessGroups(xls.Survey)
	if err != nil {
		return nil, err
	}
	var b nodeBuilder
	global, err := b.buildGroup(survey)
	if err != nil {
		return nil, err
	}
	ajf.Slides = global.Nodes
	for i := range ajf.Slides {
		if ajf.Slides[i].Type == NtGroup {
			ajf.Slides[i].Type = NtSlide
		}
	}
	assignIds(ajf.Slides, 0)
	return &ajf, nil
}

func buildChoicesOrigins(rows []ChoicesRow) ([]ChoicesOrigin, map[string][]Choice) {
	choicesMap := make(map[string][]Choice)
	for _, row := range rows {
		choicesMap[row.ListName] = append(choicesMap[row.ListName], Choice{
			Value: row.Name,
			Label: row.Label,
		})
	}
	co := make(coSlice, 0, len(choicesMap))
	for name, list := range choicesMap {
		co = append(co, ChoicesOrigin{
			Type:        OtFixed,
			Name:        name,
			ChoicesType: CtString,
			Choices:     list,
		})
	}
	sort.Sort(co)
	return co, choicesMap
}

type coSlice []ChoicesOrigin

func (co coSlice) Len() int           { return len(co) }
func (co coSlice) Less(i, j int) bool { return co[i].Name < co[j].Name }
func (co coSlice) Swap(i, j int)      { co[i], co[j] = co[j], co[i] }

func checkChoicesRef(survey []SurveyRow, choicesMap map[string][]Choice) error {
	for _, row := range survey {
		if isSelectOne(row.Type) || isSelectMultiple(row.Type) {
			c := choiceName(row.Type)
			if _, ok := choicesMap[c]; !ok {
				return fmtSrcErr(row.LineNum, "Undefined single or multiple choice %q.", c)
			}
		}
	}
	return nil
}

func choiceName(rowType string) string { return rowType[strings.Index(rowType, " ")+1:] }

func fmtSrcErr(lineNum int, format string, a ...interface{}) error {
	return fmt.Errorf("line %d: "+format, append([]interface{}{lineNum}, a...)...)
}

func checkTypes(survey []SurveyRow) error {
	for _, row := range survey {
		switch {
		case isSupportedField(row.Type):
			continue
		case isUnsupportedField(row.Type):
			return fmtSrcErr(row.LineNum, "Questions of type %q are not supported.", row.Type)
		case row.Type == beginGroup || row.Type == endGroup:
			continue
		case row.Type == beginRepeat || row.Type == endRepeat:
			continue
		case row.Type == "":
			return fmtSrcErr(row.LineNum, "Empty type in non-empty survey row.", row.Type)
		default:
			return fmtSrcErr(row.LineNum, "Invalid type %q in survey.", row.Type)
		}
	}
	return nil
}

func preprocessGroups(survey []SurveyRow) ([]SurveyRow, error) {
	var stack []*SurveyRow
	ungroupedQLine := -1
	repeatLine := -1
	for i := range survey {
		row := &survey[i]
		switch row.Type {
		case beginRepeat:
			if len(stack) > 0 {
				return nil, fmtSrcErr(row.LineNum, "Repeats can't be nested.")
			}
			repeatLine = row.LineNum
			fallthrough
		case beginGroup:
			stack = append(stack, row)
		case endRepeat, endGroup:
			if len(stack) == 0 || stack[len(stack)-1].Type[len("begin"):] != row.Type[len("end"):] {
				return nil, fmtSrcErr(row.LineNum, "Unexpected end of group/repeat.")
			}
			stack = stack[0 : len(stack)-1]
		default:
			if len(stack) == 0 {
				ungroupedQLine = row.LineNum
			}
		}
	}
	if len(stack) > 0 {
		return nil, fmtSrcErr(stack[len(stack)-1].LineNum, "Unclosed group/repeat.")
	}
	if ungroupedQLine != -1 && repeatLine != -1 {
		return nil, fmt.Errorf(
			"Can't have ungrouped questions (line %d) and repeats (line %d) in the same file.",
			ungroupedQLine, repeatLine,
		)
	}
	if ungroupedQLine != -1 {
		// Wrap everything into a slide.
		survey = append([]SurveyRow{{Type: beginGroup, Name: "form", Label: "Form"}}, survey...)
		survey = append(survey, SurveyRow{Type: endGroup})
	}
	// Wrap everything into a global group,
	// it allows building the form with a single call to buildGroup.
	survey = append([]SurveyRow{{Type: beginGroup, Name: "global"}}, survey...)
	survey = append(survey, SurveyRow{Type: endGroup})
	return survey, nil
}

type nodeBuilder struct {
	parser parser // for formulas
}

func (b *nodeBuilder) buildGroup(survey []SurveyRow) (Node, error) {
	row := survey[0]
	if row.Type != beginGroup && row.Type != beginRepeat {
		panic("not a group")
	}
	group := Node{
		Name:  row.Name,
		Label: row.Label,
		Type:  NtGroup,
		Nodes: make([]Node, 0, 8),
	}
	var err error
	group.Visibility, err = b.nodeVisibility(&row)
	if err != nil {
		return Node{}, err
	}
	if row.Type == beginRepeat {
		group.Type = NtRepeatingSlide
		if row.RepeatCount != "" {
			reps, err := strconv.ParseUint(row.RepeatCount, 10, 16)
			if err != nil {
				return Node{}, fmtSrcErr(row.LineNum, "repeat_count is not an uint16.")
			}
			group.MaxReps = new(int)
			*group.MaxReps = int(reps)
		}
	}
	for i := 1; i < len(survey); i++ {
		row := survey[i]
		switch {
		case isSupportedField(row.Type):
			field, err := b.buildField(&row)
			if err != nil {
				return Node{}, err
			}
			group.Nodes = append(group.Nodes, field)
		case row.Type == beginGroup || row.Type == beginRepeat:
			end := groupEnd(survey, i)
			child, err := b.buildGroup(survey[i:end])
			if err != nil {
				return Node{}, err
			}
			group.Nodes = append(group.Nodes, child)
			i = end - 1
		case row.Type == endGroup || row.Type == endRepeat:
			if i != len(survey)-1 {
				panic("unexpected end of group")
			}
		default:
			panic("unexpected row type")
		}
	}
	return group, nil
}

func groupEnd(survey []SurveyRow, groupStart int) int {
	groupDepth := 1
	for i := groupStart + 1; i < len(survey); i++ {
		switch survey[i].Type {
		case beginGroup, beginRepeat:
			groupDepth++
		case endGroup, endRepeat:
			groupDepth--
			if groupDepth == 0 {
				return i + 1
			}
		}
	}
	panic("group end not found")
}

func (b *nodeBuilder) buildField(row *SurveyRow) (Node, error) {
	field := Node{
		Name:  row.Name,
		Label: row.Label,
		Type:  NtField,
	}
	var err error
	field.Visibility, err = b.nodeVisibility(row)
	if err != nil {
		return Node{}, err
	}
	field.Validation, err = b.fieldValidation(row)
	if err != nil {
		return Node{}, err
	}
	switch {
	case row.Type == "decimal" || row.Type == "integer":
		field.FieldType = &FtNumber
	case row.Type == "text":
		field.FieldType = &FtString
	case row.Type == "boolean":
		field.FieldType = &FtBoolean
	case isSelectOne(row.Type):
		field.FieldType = &FtSingleChoice
		field.ChoicesOriginRef = choiceName(row.Type)
	case isSelectMultiple(row.Type):
		field.FieldType = &FtMultipleChoice
		field.ChoicesOriginRef = choiceName(row.Type)
	case row.Type == "note":
		field.Label = ""
		field.FieldType = &FtNote
		field.HTML = row.Label
	case row.Type == "date":
		field.FieldType = &FtDate
	case row.Type == "time":
		field.FieldType = &FtTime
	case row.Type == "calculate":
		field.FieldType = &FtFormula
		js, err := b.parser.Parse(row.Calculation, "calculation", row.Name)
		if err != nil {
			return Node{}, fmtSrcErr(row.LineNum, "%s", err)
		}
		field.Formula = &Formula{js}
	default:
		panic("unexpected row type")
	}
	return field, nil
}

func (b *nodeBuilder) nodeVisibility(row *SurveyRow) (*NodeVisibility, error) {
	if row.Relevant == "" {
		return nil, nil
	}
	js, err := b.parser.Parse(row.Relevant, "relevant", row.Name)
	if err != nil {
		return nil, fmtSrcErr(row.LineNum, "%s", err)
	}
	return &NodeVisibility{Condition: js}, nil
}

func (b *nodeBuilder) fieldValidation(row *SurveyRow) (*FieldValidation, error) {
	if row.Required == "" && row.Constraint == "" && row.Type != "integer" {
		return nil, nil
	}
	v := new(FieldValidation)

	if row.Required != "" && row.Required != "yes" {
		return nil, fmtSrcErr(row.LineNum, `Invalid value %q in "required" column.`, row.Required)
	}
	if row.Required == "yes" {
		v.NotEmpty = true
	}

	if row.Type == "integer" {
		v.Conditions = []ValidationCondition{{
			Condition:        "isInt(" + row.Name + ")", // ajf function
			ClientValidation: true,
			ErrorMessage:     "The field value must be an integer.",
		}}
	}
	if row.Constraint == "" {
		return v, nil
	}
	js, err := b.parser.Parse(row.Constraint, "constraint", row.Name)
	if err != nil {
		return nil, fmtSrcErr(row.LineNum, "%s", err)
	}
	v.Conditions = append(v.Conditions, ValidationCondition{
		Condition:        js,
		ClientValidation: true,
		ErrorMessage:     row.ConstraintMessage,
	})
	return v, nil
}

const idMultiplier = 1000

func assignIds(nodes []Node, parent int) {
	if len(nodes) == 0 {
		return
	}
	nodes[0].Previous = parent
	nodes[0].Id = parent*idMultiplier + 1
	assignIds(nodes[0].Nodes, nodes[0].Id)
	for i := 1; i < len(nodes); i++ {
		nodes[i].Previous = nodes[i-1].Id
		nodes[i].Id = nodes[i-1].Id + 1
		assignIds(nodes[i].Nodes, nodes[i].Id)
	}
}

const (
	beginGroup  = "begin group"
	endGroup    = "end group"
	beginRepeat = "begin repeat"
	endRepeat   = "end repeat"
)

var supportedField = map[string]bool{
	"decimal": true, "integer": true, "text": true, "boolean": true,
	"note": true, "date": true, "time": true, "calculate": true,
}

func isSupportedField(typ string) bool {
	return supportedField[typ] || isSelectOne(typ) || isSelectMultiple(typ)
}
func isSelectOne(typ string) bool      { return strings.HasPrefix(typ, "select_one ") }
func isSelectMultiple(typ string) bool { return strings.HasPrefix(typ, "select_multiple ") }

var unsupportedField = map[string]bool{
	"range": true, "geopoint": true, "geotrace": true, "geoshape": true,
	"datetime": true, "image": true, "audio": true, "video": true, "file": true,
	"barcode": true, "acknowledge": true, "hidden": true, "xml-external": true,
	// metadata:
	"start": true, "end": true, "today": true, "deviceid": true, "subscriberid": true,
	"simserial": true, "phonenumber": true, "username": true, "email": true,
}

func isUnsupportedField(typ string) bool { return unsupportedField[typ] || isRank(typ) }
func isRank(typ string) bool             { return strings.HasPrefix(typ, "rank ") }
