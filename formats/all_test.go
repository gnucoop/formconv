package formats

import (
	"reflect"
	"testing"
)

func TestDecodeXls(t *testing.T) {
	fileName := "testdata/skeleton"
	expected := &XlsForm{
		Survey: []SurveyRow{
			{LineNumber: 2, Type: "type1", Name: "name1", Label: "label1", Required: "yes"},
			{LineNumber: 3, Type: "type2", Name: "name2", Label: "label2", Required: "yes"},
		},
		Choices: []ChoicesRow{
			{LineNumber: 4, ListName: "listname1", Name: "name1", Label: "label1"},
			{LineNumber: 6, ListName: "listname2", Name: "name2", Label: "label2"},
			{LineNumber: 8, ListName: "listname3", Name: "name3", Label: "label3"},
		},
	}
	for _, ext := range []string{".xls", ".xlsx"} {
		xls, err := DecXlsFromFile(fileName + ext)
		if err != nil {
			t.Fatal(err)
		}
		expected.FileName = "skeleton" + ext
		if !reflect.DeepEqual(xls, expected) {
			t.Fatalf("Error decoding %s: expected %v, got %v", fileName+ext, expected, xls)
		}
	}
}

func TestBuildChoicesOrigins(t *testing.T) {
	choicesSheet := []ChoicesRow{
		{"list1", "elem1a", "label1a", 0},
		{"list2", "elem2a", "label2a", 0},
		{"list1", "elem1b", "label1b", 0},
	}
	choices, _ := buildChoicesOrigins(choicesSheet)
	expected1 := []ChoicesOrigin{{
		Type:        OtFixed,
		Name:        "list1",
		ChoicesType: CtString,
		Choices:     []Choice{{"elem1a", "label1a"}, {"elem1b", "label1b"}},
	}, {
		Type:        OtFixed,
		Name:        "list2",
		ChoicesType: CtString,
		Choices:     []Choice{{"elem2a", "label2a"}},
	}}
	expected2 := []ChoicesOrigin{expected1[1], expected1[0]}
	if !reflect.DeepEqual(choices, expected1) && !reflect.DeepEqual(choices, expected2) {
		t.Fatalf("Error building choices origins of %v: expected %v, got %v",
			&choicesSheet, expected1, choices)
	}
}
