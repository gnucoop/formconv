package main

func xls2ajf(xls *xlsForm) (*ajfForm, error) {

	ajf := ajfForm{
		ChoicesOrigins: []choicesOrigin{{
			Type:        otFixed,
			Name:        "years",
			ChoicesType: ctString,
			Choices:     []choice{{"1 Year", "1_year"}, {"2 Years", "2_years"}},
		}},
		Slides: []slide{{
			Id:       1,
			Parent:   0,
			NodeType: ntSlide,
			Name:     "slide1",
			Label:    "First Slide",
			Fields: []field{{
				Id:               101,
				Parent:           1,
				NodeType:         ntField,
				FieldType:        ftSingleChoice,
				Name:             "yearfield",
				Label:            "Select Years",
				ChoicesOriginRef: "years",
				Validation:       fieldValidation{NotEmpty: true},
			}},
		}},
	}
	return &ajf, nil
}
