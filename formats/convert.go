package formats

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func Convert(xls *XlsForm) (*AjfForm, error) {
	err := checkTypes(xls.Survey)
	if err != nil {
		return nil, err
	}
	err = checkNames(xls.Survey)
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

	err = processSettings(xls.Settings, &ajf)
	if err != nil {
		return nil, err
	}
	return &ajf, nil
}

func buildChoicesOrigins(rows []ChoicesRow) ([]ChoicesOrigin, map[string][]Choice) {
	choicesMap := make(map[string][]Choice)
	for _, row := range rows {
		choice := row.UserDefCells()
		choice["value"] = row.Name()
		choice["label"] = row.Label()
		choicesMap[row.ListName()] = append(choicesMap[row.ListName()], choice)
	}
	co := make(coSlice, 0, len(choicesMap))
	for name, list := range choicesMap {
		// Skip empty choice lists; useful in case the choice list
		// is to be omitted in the resulting json and defined later.
		if len(list) == 1 && list[0]["value"] == "" && list[0]["label"] == "" {
			continue
		}
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

func choiceType(rowType string) *FieldType {
	if isSelectOne(rowType) {
		return &FtSingleChoice
	}
	if isSelectMultiple(rowType) {
		return &FtMultipleChoice
	}
	panic("not a choice")
}
func choiceName(rowType string) string {
	if !isSelectOne(rowType) && !isSelectMultiple(rowType) {
		panic("not a choice")
	}
	return rowType[strings.Index(rowType, " ")+1:]
}

func fmtSrcErr(lineNum int, format string, a ...interface{}) error {
	return fmt.Errorf("line %d: "+format, append([]interface{}{lineNum}, a...)...)
}

func checkTypes(survey []SurveyRow) error {
	for _, row := range survey {
		switch {
		case isSupportedField(row.Type) || isIgnoredField(row.Type):
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

func checkNames(survey []SurveyRow) error {
	for _, row := range survey {
		name := row.Name()
		switch row.Type {
		case endGroup, endRepeat:
			if name != "" {
				return fmtSrcErr(row.LineNum, "End of group/repeat can't have a name.")
			}
		case "note":
			if name == "" {
				continue // name is optional for note
			}
			fallthrough
		default:
			if !isIdentifier(name) {
				return fmtSrcErr(row.LineNum, "Name %q is not a valid identifier.", name)
			}
		}
	}
	return nil
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) || (unicode.IsDigit(r) && i > 0) {
			continue // r is ok
		}
		return false
	}
	return true
}

func preprocessGroups(survey []SurveyRow) ([]SurveyRow, error) {
	// Make sure groups are balanced and repeats aren't nested.
	var stack []*SurveyRow
	for i := range survey {
		row := &survey[i]
		switch row.Type {
		case beginRepeat:
			if len(stack) > 0 {
				return nil, fmtSrcErr(row.LineNum, "Repeats can't be nested.")
			}
			fallthrough
		case beginGroup:
			stack = append(stack, row)
		case endRepeat, endGroup:
			if len(stack) == 0 ||
				stack[len(stack)-1].Type[len("begin"):] != row.Type[len("end"):] {
				return nil, fmtSrcErr(row.LineNum, "Unexpected end of group/repeat.")
			}
			stack = stack[0 : len(stack)-1]
		}
	}
	if len(stack) > 0 {
		return nil, fmtSrcErr(stack[len(stack)-1].LineNum, "Unclosed group/repeat.")
	}

	// Wrap everything into a temporary global group,
	// it allows building the form with a single call to buildGroup;
	// also create groups for adjacent ungrouped questions.
	newSurvey := []SurveyRow{MakeSurveyRow("type", beginGroup, "name", "global")}
	groupDepth := 0
	grouping := false
	slideNum := 0
	for _, row := range survey {
		switch row.Type {
		case beginGroup, beginRepeat:
			if grouping {
				newSurvey = append(newSurvey, MakeSurveyRow("type", endGroup))
				grouping = false
			}
			groupDepth++
		case endGroup, endRepeat:
			groupDepth--
		default:
			if groupDepth == 0 && !grouping {
				grouping = true
				slideName := "slide" + strconv.Itoa(slideNum)
				slideNum++
				newSurvey = append(newSurvey, MakeSurveyRow("type", beginGroup, "name", slideName))
			}
		}
		newSurvey = append(newSurvey, row)
	}
	if grouping {
		newSurvey = append(newSurvey, MakeSurveyRow("type", endGroup))
	}
	newSurvey = append(newSurvey, MakeSurveyRow("type", endGroup)) // global group
	return newSurvey, nil
}

type nodeBuilder struct {
	parser formulaParser
}

func (b *nodeBuilder) buildGroup(survey []SurveyRow) (Node, error) {
	row := survey[0]
	if row.Type != beginGroup && row.Type != beginRepeat {
		panic("not a group")
	}
	group := Node{
		Name:  row.Name(),
		Label: row.Label(),
		Type:  NtGroup,
		Nodes: make([]Node, 0, 8),
	}
	var err error
	group.Visibility, err = b.nodeVisibility(row)
	if err != nil {
		return Node{}, err
	}
	if row.Type == beginRepeat {
		group.Type = NtRepeatingSlide
		if row.RepeatCount() != "" {
			reps, ok := parseExcelUint(row.RepeatCount())
			if !ok {
				return Node{}, fmtSrcErr(row.LineNum, "repeat_count is not an unsigned integer.")
			}
			group.MaxReps = &reps
		}
	}
	for i := 1; i < len(survey); i++ {
		row := survey[i]
		switch {
		case isIgnoredField(row.Type):
			continue
		case isSupportedField(row.Type):
			field, err := b.buildField(row)
			if err != nil {
				return Node{}, err
			}
			group.Nodes = append(group.Nodes, field)
		case row.Type == beginGroup || row.Type == beginRepeat:
			end := groupEnd(survey, i)
			child, err := b.buildGroup(survey[i : end+1])
			if err != nil {
				return Node{}, err
			}
			group.Nodes = append(group.Nodes, child)
			i = end
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

func parseExcelUint(s string) (i int, ok bool) {
	// xlsx files from google sheets may contain ints like 1.23e2
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 || f > math.MaxInt32 || f != math.Floor(f) {
		return -1, false
	}
	return int(f), true
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
				return i
			}
		}
	}
	panic("group end not found")
}

func (b *nodeBuilder) buildField(row SurveyRow) (Node, error) {
	field := Node{
		Name:  row.Name(),
		Label: row.Label(),
		Hint:  row.Hint(),
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
	case row.Type == "textarea":
		field.FieldType = &FtText
	case row.Type == "boolean":
		field.FieldType = &FtBoolean
	case isSelectOne(row.Type) || isSelectMultiple(row.Type):
		field.FieldType = choiceType(row.Type)
		field.ChoicesOriginRef = choiceName(row.Type)
		if filter := row.ChoiceFilter(); filter != "" {
			js, err := b.parser.Parse(filter, "choice_filter", row.Name())
			if err != nil {
				return Node{}, fmtSrcErr(row.LineNum, "%s", err)
			}
			field.ChoicesFilter = &Formula{js}
		}
	case row.Type == "note":
		field.Label = ""
		field.FieldType = &FtNote
		field.HTML = row.Label()
	case row.Type == "date":
		field.FieldType = &FtDate
	case row.Type == "time":
		field.FieldType = &FtTime
	case row.Type == "calculate":
		field.FieldType = &FtFormula
		js, err := b.parser.Parse(row.Calculation(), "calculation", row.Name())
		if err != nil {
			return Node{}, fmtSrcErr(row.LineNum, "%s", err)
		}
		field.Formula = &Formula{js}
	case row.Type == "geopoint":
		field.FieldType = &FtGeolocation
		// may want to do field.TileLayer = row.Label()
	case row.Type == "barcode":
		field.FieldType = &FtBarcode
	case row.Type == "file":
		field.FieldType = &FtFile
	case row.Type == "image":
		field.FieldType = &FtImage
	default:
		panic("unexpected row type")
	}
	return field, nil
}

func (b *nodeBuilder) nodeVisibility(row SurveyRow) (*NodeVisibility, error) {
	rel := row.Relevant()
	if rel == "" {
		return nil, nil
	}
	js, err := b.parser.Parse(rel, "relevant", row.Name())
	if err != nil {
		return nil, fmtSrcErr(row.LineNum, "%s", err)
	}
	return &NodeVisibility{Condition: js}, nil
}

var requiredVals = map[string]bool{"": true, "yes": true, "no": true, "true": true, "false": true}

func (b *nodeBuilder) fieldValidation(row SurveyRow) (*FieldValidation, error) {
	req := row.Required()
	con := row.Constraint()
	if req == "" && con == "" && row.Type != "integer" {
		return nil, nil
	}
	v := new(FieldValidation)

	if !requiredVals[req] {
		return nil, fmtSrcErr(row.LineNum, `Invalid value %q in "required" column.`, req)
	}
	if req == "yes" || req == "true" {
		v.NotEmpty = true
	}

	if row.Type == "integer" {
		v.Conditions = []ValidationCondition{{
			Condition:        "isInt(" + row.Name() + ") || !notEmpty(" + row.Name() + ")",
			ClientValidation: true,
			ErrorMessage:     "The field value must be an integer.",
		}}
	}
	if con == "" {
		return v, nil
	}
	js, err := b.parser.Parse(con, "constraint", row.Name())
	if err != nil {
		return nil, fmtSrcErr(row.LineNum, "%s", err)
	}
	v.Conditions = append(v.Conditions, ValidationCondition{
		Condition:        js,
		ClientValidation: true,
		ErrorMessage:     row.ConstraintMsg(),
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

func processSettings(settings []SettingsRow, ajf *AjfForm) error {
	for _, row := range settings {
		lab := row.TagLabel()
		val := row.TagValue()
		if lab == "" && val == "" {
			continue
		}
		if lab == "" {
			return fmtSrcErr(row.LineNum, "Tag with no label.")
		}
		if !isIdentifier(val) {
			return fmtSrcErr(row.LineNum, "Tag value %q is not a valid identifier.", val)
		}
		var t Tag
		t.Label = lab
		t.Value[0] = val
		ajf.StringIdentifier = append(ajf.StringIdentifier, t)
	}
	return nil
}

const (
	beginGroup  = "begin group"
	endGroup    = "end group"
	beginRepeat = "begin repeat"
	endRepeat   = "end repeat"
)

var supportedFields = map[string]bool{
	"decimal": true, "integer": true, "text": true, "textarea": true, "boolean": true,
	"note": true, "date": true, "time": true, "calculate": true,
	"barcode": true, "geopoint": true, "file": true, "image": true,
}

func isSupportedField(typ string) bool {
	return supportedFields[typ] || isSelectOne(typ) || isSelectMultiple(typ)
}
func isSelectOne(typ string) bool      { return strings.HasPrefix(typ, "select_one ") }
func isSelectMultiple(typ string) bool { return strings.HasPrefix(typ, "select_multiple ") }

var ignoredFields = map[string]bool{ // metadata:
	"start": true, "end": true, "today": true, "deviceid": true, "subscriberid": true,
	"simserial": true, "phonenumber": true, "username": true, "email": true,
}

func isIgnoredField(typ string) bool { return ignoredFields[typ] }

var unsupportedFields = map[string]bool{
	"range": true, "geotrace": true, "geoshape": true,
	"datetime": true, "audio": true, "video": true,
	"acknowledge": true, "hidden": true, "xml-external": true,
}

func isUnsupportedField(typ string) bool { return unsupportedFields[typ] || isRank(typ) }
func isRank(typ string) bool             { return strings.HasPrefix(typ, "rank ") }
