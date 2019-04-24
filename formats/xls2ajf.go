package formats

import (
	"errors"
	"fmt"
	"strings"
)

func Xls2ajf(xls *XlsForm) (*AjfForm, error) {
	survey, err := checkGroups(xls.Survey)
	if err != nil {
		return nil, err
	}
	var ajf AjfForm
	var choicesMap map[string][]Choice
	ajf.ChoicesOrigins, choicesMap = buildChoicesOrigins(xls.Choices)

	groupDepth := 0
	var curGroup *Node
	for _, row := range survey {
		switch {
		case row.Type == beginGroup:
			groupDepth++
			if groupDepth == 1 {
				ajf.Slides = append(ajf.Slides, Node{
					Name:  row.Name,
					Label: row.Label,
					Type:  NtSlide,
					Nodes: make([]Node, 0),
				})
				curGroup = &ajf.Slides[len(ajf.Slides)-1]
			}
		case row.Type == endGroup:
			groupDepth--
			if groupDepth == 0 {
				curGroup = nil
			}
		case isSupportedField(row.Type):
			curGroup.Nodes = append(curGroup.Nodes, Node{
				Name:      row.Name,
				Label:     row.Label,
				Type:      NtField,
				FieldType: fieldTypeFrom(row.Type),
			})
			curField := &curGroup.Nodes[len(curGroup.Nodes)-1]
			if *curField.FieldType == FtSingleChoice || *curField.FieldType == FtMultipleChoice {
				choiceName := row.Type[strings.Index(row.Type, " ")+1:]
				if _, present := choicesMap[choiceName]; !present {
					return nil, fmt.Errorf("Undefined choice %s", choiceName)
				}
				curField.ChoicesOriginRef = choiceName
			}
			if *curField.FieldType == FtNote {
				curField.HTML = row.Label
			}
			if row.Required == "yes" {
				curField.Validation = &FieldValidation{NotEmpty: true}
			}
		case isUnsupportedField(row.Type):
			return nil, fmt.Errorf("Field type %q is not supported", row.Type)
		case row.Type == beginRepeat || row.Type == endRepeat:
			return nil, fmt.Errorf("Repeats are not supported")
		default:
			return nil, fmt.Errorf("Invalid type %q in survey", row.Type)
		}
	}
	assignIds(&ajf)
	return &ajf, nil
}

var notBalancedErr = errors.New("Groups are not balanced")

func checkGroups(survey []SurveyRow) ([]SurveyRow, error) {
	groupDepth := 0
	ungroupedItems := false
	for _, row := range survey {
		switch row.Type {
		case beginGroup:
			groupDepth++
		case endGroup:
			groupDepth--
			if groupDepth < 0 {
				return nil, notBalancedErr
			}
		default:
			if groupDepth == 0 {
				ungroupedItems = true
			}
		}
	}
	if groupDepth != 0 {
		return nil, notBalancedErr
	}
	if ungroupedItems || len(survey) == 0 {
		// Wrap everything into a big group/slide.
		survey = append([]SurveyRow{{Type: beginGroup, Name: "form", Label: "Form"}}, survey...)
		survey = append(survey, SurveyRow{Type: endGroup})
	}
	return survey, nil
}

func buildChoicesOrigins(rows []ChoicesRow) ([]ChoicesOrigin, map[string][]Choice) {
	choicesMap := make(map[string][]Choice)
	for _, row := range rows {
		choicesMap[row.ListName] = append(choicesMap[row.ListName], Choice{
			Value: row.Name,
			Label: row.Label,
		})
	}
	var co []ChoicesOrigin
	for name, list := range choicesMap {
		co = append(co, ChoicesOrigin{
			Type:        OtFixed,
			Name:        name,
			ChoicesType: CtString,
			Choices:     list,
		})
	}
	return co, choicesMap
}

const (
	beginGroup  = "begin group"
	endGroup    = "end group"
	beginRepeat = "begin repeat"
	endRepeat   = "end repeat"
)

var supportedField = map[string]bool{
	"decimal": true, "text": true, "select_one yes_no": true, "note": true,
	"date": true, "time": true, "calculate": true,
}

func isSupportedField(typ string) bool {
	return supportedField[typ] || isSelectOne(typ) || isSelectMultiple(typ)
}
func isSelectOne(typ string) bool {
	return strings.HasPrefix(typ, "select_one ") && typ != "select_one yes_no"
}
func isSelectMultiple(typ string) bool { return strings.HasPrefix(typ, "select_multiple ") }

var unsupportedField = map[string]bool{
	"integer": true, "range": true, "geopoint": true, "geotrace": true, "geoshape": true,
	"datetime": true, "image": true, "audio": true, "video": true, "file": true,
	"barcode": true, "acknowledge": true, "hidden": true, "xml-external": true,
	// metadata:
	"start": true, "end": true, "today": true, "deviceid": true, "subscriberid": true,
	"simserial": true, "phonenumber": true, "username": true, "email": true,
}

func isUnsupportedField(typ string) bool { return unsupportedField[typ] || isRank(typ) }
func isRank(typ string) bool             { return strings.HasPrefix(typ, "rank ") }

func fieldTypeFrom(typ string) *FieldType {
	switch {
	case typ == "decimal":
		return &FtNumber
	case typ == "text":
		return &FtString
	case typ == "select_one yes_no":
		return &FtBoolean
	case isSelectOne(typ):
		return &FtSingleChoice
	case isSelectMultiple(typ):
		return &FtMultipleChoice
	case typ == "note":
		return &FtNote
	case typ == "date":
		return &FtDate
	case typ == "time":
		return &FtTime
	case typ == "calculate":
		return &FtFormula
	case isUnsupportedField(typ):
		panic("unsupported type: " + typ)
	default:
		panic("unrecognized type: " + typ)
	}
}

func assignIds(ajf *AjfForm) {
	for i := range ajf.Slides {
		slide := &ajf.Slides[i]
		slide.Id = i + 1
		slide.Previous = i
		for j := range slide.Nodes {
			field := &slide.Nodes[j]
			field.Id = slide.Id*1000 + j
			if j == 0 {
				field.Previous = slide.Id
			} else {
				field.Previous = slide.Nodes[j-1].Id
			}
		}
	}
}
