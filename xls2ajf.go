package main

import (
	"errors"
	"fmt"
	"strings"
)

const (
	beginGroup = "begin group"
	endGroup   = "end group"
	selectOne  = "select_one "
)

func Xls2ajf(xls *XlsForm) (*AjfForm, error) {
	var err error
	xls.Survey, err = checkGroups(xls.Survey)
	if err != nil {
		return nil, err
	}
	var ajf AjfForm
	var choicesMap map[string][]Choice
	ajf.ChoicesOrigins, choicesMap = buildChoicesOrigins(xls.Choices)

	groupDepth := 0
	var curSlide *Slide
	for _, row := range xls.Survey {
		switch row.Type {
		case beginGroup:
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
			continue
		case endGroup:
			groupDepth--
			if groupDepth == 0 {
				curSlide = nil
			}
			continue
		}
		// default:
		curSlide.Fields = append(curSlide.Fields, Field{
			NodeType:  NtField,
			FieldType: fieldTypeFrom(row.Type),
			Name:      row.Name,
			Label:     row.Label,
		})
		curField := &curSlide.Fields[len(curSlide.Fields)-1]
		if strings.HasPrefix(row.Type, selectOne) {
			choiceName := row.Type[len(selectOne):]
			if _, present := choicesMap[choiceName]; !present {
				return nil, fmt.Errorf("Undefined single choice %s", choiceName)
			}
			curField.ChoicesOriginRef = choiceName
		}
		if row.Required == "yes" {
			curField.Validation = &FieldValidation{NotEmpty: true}
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

func fieldTypeFrom(typ string) FieldType {
	switch {
	case strings.HasPrefix(typ, selectOne):
		return FtSingleChoice
	case typ == "date":
		return FtDateInput
	default:
		return FtString
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
