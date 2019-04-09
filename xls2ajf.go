package main

import "errors"

const (
	beginGroup = "begin group"
	endGroup   = "end group"
)

func xls2ajf(xls *xlsForm) (*ajfForm, error) {
	xls, err := checkGroups(xls)
	if err != nil {
		return nil, err
	}
	var ajf ajfForm
	ajf.ChoicesOrigins = buildChoicesOrigins(&xls.choices)

	groupDepth := 0
	slideId := 1
	//var currentSlide *slide
	for i, typ := range xls.survey.types {
		switch typ {
		case beginGroup:
			groupDepth++
			if groupDepth == 1 {
				ajf.Slides = append(ajf.Slides, slide{
					Id:       slideId,
					Parent:   0,
					NodeType: ntSlide,
					Name:     xls.survey.names[i],
					Label:    xls.survey.labels[i],
					Fields:   make([]field, 0),
				})
				slideId++
				//currentSlide = &ajf.Slides[len(ajf.Slides)-1]
			}
		case endGroup:
			groupDepth--
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

func buildChoicesOrigins(choices *choices) []choicesOrigin {
	lists := make(map[string][]choice)
	for i, listName := range choices.listNames {
		lists[listName] = append(lists[listName], choice{
			Label: choices.labels[i],
			Value: choices.names[i],
		})
	}
	// We want empty slices to be json-encoded as [], not null:
	co := make([]choicesOrigin, 0)
	for name, list := range lists {
		co = append(co, choicesOrigin{
			Type:        otFixed,
			Name:        name,
			ChoicesType: ctString,
			Choices:     list,
		})
	}
	return co
}
