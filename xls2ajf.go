package main

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
	var curSlide *Slide
	for _, row := range survey {
		switch {
		case row.Type == beginGroup:
			groupDepth++
			if groupDepth == 1 {
				ajf.Slides = append(ajf.Slides, Slide{
					NodeType: NtSlide,
					Name:     row.Name,
					Label:    row.Label,
					Fields:   make([]Field, 0),
				})
				curSlide = &ajf.Slides[len(ajf.Slides)-1]
			}
		case row.Type == endGroup:
			groupDepth--
			if groupDepth == 0 {
				curSlide = nil
			}
		case stringField[row.Type] || supportedField[row.Type] ||
			strings.HasPrefix(row.Type, selectOne) || strings.HasPrefix(row.Type, selectMultiple):

			curSlide.Fields = append(curSlide.Fields, Field{
				NodeType:  NtField,
				FieldType: fieldTypeFrom(row.Type),
				Name:      row.Name,
				Label:     row.Label,
			})
			curField := &curSlide.Fields[len(curSlide.Fields)-1]
			if strings.HasPrefix(row.Type, selectOne) || strings.HasPrefix(row.Type, selectMultiple) {
				choiceName := row.Type[strings.Index(row.Type, " ")+1:]
				if _, present := choicesMap[choiceName]; !present {
					return nil, fmt.Errorf("Undefined choice %s", choiceName)
				}
				curField.ChoicesOriginRef = choiceName
			}
			if row.Required == "yes" {
				curField.Validation = &FieldValidation{NotEmpty: true}
			}
		case unsupportedField[row.Type] || strings.HasPrefix(row.Type, rank):
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

	selectOne      = "select_one "
	selectMultiple = "select_multiple "
	rank           = "rank "
)

var (
	stringField = map[string]bool{
		"text": true, "geopoint": true, "geotrace": true, "geoshape": true, "time": true, "datetime": true,
	}
	supportedField = map[string]bool{
		"integer": true, "decimal": true, "note": true, "date": true, "calculate": true, "acknowledge": true, // selectOne, selectMiltiple
	}
	unsupportedField = map[string]bool{
		"range": true, "image": true, "audio": true, "video": true, "file": true, "barcode": true, "hidden": true, "xml-external": true, // rank
	}
	metadata = map[string]bool{
		"start": true, "end": true, "today": true, "deviceid": true, "subscriberid": true, "simserial": true, "phonenumber": true, "username": true, "email": true,
	}
)

func fieldTypeFrom(typ string) FieldType {
	switch {
	case typ == "integer" || typ == "decimal":
		return FtNumber
	case stringField[typ]:
		return FtString
	case strings.HasPrefix(typ, selectOne):
		return FtSingleChoice
	case strings.HasPrefix(typ, selectMultiple):
		return FtMultipleChoice
	case typ == "note":
		return FtEmpty
	case typ == "date":
		return FtDateInput
	case typ == "calculate":
		return FtFormula
	case typ == "acknowledge":
		return FtBoolean
	case strings.HasPrefix(typ, rank):
		fallthrough
	case unsupportedField[typ]:
		panic("unsupported")
	default:
		panic("unrecognized")
	}
}

func assignIds(ajf *AjfForm) {
	for i := range ajf.Slides {
		slide := &ajf.Slides[i]
		slide.Id = i + 1
		slide.Parent = i
		for j := range slide.Fields {
			field := &slide.Fields[j]
			field.Id = slide.Id*1000 + j
			if j == 0 {
				field.Parent = slide.Id
			} else {
				field.Parent = slide.Fields[j-1].Id
			}
		}
	}
}
