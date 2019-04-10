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

func xls2ajf(xls *xlsForm) (*ajfForm, error) {
	xls, err := checkGroups(xls)
	if err != nil {
		return nil, err
	}
	var ajf ajfForm
	var choicesMap map[string][]choice
	ajf.ChoicesOrigins, choicesMap = buildChoicesOrigins(&xls.choices)

	groupDepth := 0
	var curSlide *slide
	for i, typ := range xls.survey.types {
		switch typ {
		case beginGroup:
			groupDepth++
			if groupDepth == 1 {
				ajf.Slides = append(ajf.Slides, slide{
					NodeType: ntSlide,
					Name:     xls.survey.names[i],
					Label:    xls.survey.labels[i],
					Fields:   make([]field, 0),
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
		curSlide.Fields = append(curSlide.Fields, field{
			NodeType:  ntField,
			FieldType: fieldTypeFrom(typ),
			Name:      xls.survey.names[i],
			Label:     xls.survey.labels[i],
		})
		curField := &curSlide.Fields[len(curSlide.Fields)-1]
		if strings.HasPrefix(typ, selectOne) {
			choiceName := typ[len(selectOne):]
			if _, present := choicesMap[choiceName]; !present {
				return nil, fmt.Errorf("Undefined single choice %s", choiceName)
			}
			curField.ChoicesOriginRef = choiceName
		}
		if xls.survey.required != nil && xls.survey.required[i] == "yes" {
			curField.Validation = &fieldValidation{NotEmpty: true}
		}
	}
	assignIds(&ajf)
	return &ajf, nil
}

var notBalancedErr = errors.New("Groups are not balanced")

func checkGroups(xls *xlsForm) (*xlsForm, error) {
	groupDepth := 0
	ungroupedItems := false
	for _, typ := range xls.survey.types {
		switch typ {
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
	if ungroupedItems || len(xls.survey.types) == 0 {
		// Wrap everything into a big group/slide.
		xls.survey.types = append([]string{beginGroup}, append(xls.survey.types, endGroup)...)
		xls.survey.names = append([]string{"form"}, append(xls.survey.names, "")...)
		xls.survey.labels = append([]string{"Form"}, append(xls.survey.labels, "")...)
	}
	return xls, nil
}

func buildChoicesOrigins(choices *choices) ([]choicesOrigin, map[string][]choice) {
	choicesMap := make(map[string][]choice)
	for i, name := range choices.listNames {
		choicesMap[name] = append(choicesMap[name], choice{
			Label: choices.labels[i],
			Value: choices.names[i],
		})
	}
	var co []choicesOrigin
	for name, list := range choicesMap {
		co = append(co, choicesOrigin{
			Type:        otFixed,
			Name:        name,
			ChoicesType: ctString,
			Choices:     list,
		})
	}
	return co, choicesMap
}

func fieldTypeFrom(typ string) fieldType {
	switch {
	case strings.HasPrefix(typ, selectOne):
		return ftSingleChoice
	case typ == "date":
		return ftDateInput
	default:
		return ftString
	}
}

func assignIds(ajf *ajfForm) {
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
