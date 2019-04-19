package main

import (
	"reflect"
	"testing"
)

func TestDeleteEmpty(t *testing.T) {
	rows := [][]string{{"1", "2", "3"}, {"", "", ""}, {"a", "b", "c"}}
	filtered := deleteEmpty(rows)
	expected := [][]string{{"1", "2", "3"}, {"a", "b", "c"}}
	if !reflect.DeepEqual(filtered, expected) {
		t.Fatalf("Error deleting empty rows from %v: expected %v, got %v", rows, expected, filtered)
	}
}

func TestDecodeXlsx(t *testing.T) {
	fileName := "testdata/skeleton.xlsx"
	xls, err := DecXlsFromFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	expected := &XlsForm{
		[]SurveyRow{
			{Type: "type1", Name: "name1", Label: "label1", Required: "yes"},
			{Type: "type2", Name: "name2", Label: "label2", Required: "yes"},
		},
		[]ChoicesRow{
			{ListName: "listname1", Name: "name1", Label: "label1"},
			{ListName: "listname2", Name: "name2", Label: "label2"},
			{ListName: "listname3", Name: "name3", Label: "label3"},
		},
	}
	if !reflect.DeepEqual(xls, expected) {
		t.Fatalf("Error decoding %s: expected %v, got %v", fileName, expected, xls)
	}
}

func TestBuildChoicesOrigins(t *testing.T) {
	choicesSheet := []ChoicesRow{
		{"list1", "elem1a", "label1a"},
		{"list2", "elem2a", "label2a"},
		{"list1", "elem1b", "label1b"},
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
