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
	for i, typ := range xls.survey.types {
		switch {
		case typ == beginGroup:
			groupDepth++
			if groupDepth == 1 {
				ajf.Slides = append(ajf.Slides, slide{
					Id:       len(ajf.Slides) + 1,
					Parent:   0,
					NodeType: ntSlide,
					Name:     xls.survey.names[i],
					Label:    xls.survey.labels[i],
					Fields:   make([]field, 0),
				})
			}
		case typ == endGroup:
			groupDepth--
		case strings.HasPrefix(typ, selectOne):
			choiceName := typ[len(selectOne):]
			if _, present := choicesMap[choiceName]; !present {
				return nil, fmt.Errorf("Undefined single choice %s", choiceName)
			}
			currentSlide := &ajf.Slides[len(ajf.Slides)-1]
			currentSlide.Fields = append(currentSlide.Fields, field{
				Id:               currentSlide.Id*1000 + len(currentSlide.Fields) + 1,
				Parent:           currentSlide.Id,
				NodeType:         ntField,
				FieldType:        ftSingleChoice,
				Name:             xls.survey.names[i],
				Label:            xls.survey.labels[i],
				ChoicesOriginRef: choiceName,
				// Validation
			})
		}
	}
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
	// We want empty slices to be json-encoded as [], not null:
	co := make([]choicesOrigin, 0)
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
