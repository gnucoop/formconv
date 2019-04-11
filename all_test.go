package main

import (
	"reflect"
	"testing"
)

func TestTranspose(t *testing.T) {
	rows := [][]string{{"1", "2", "3"}, {"a", "b", "c"}}
	cols := transpose(rows)
	expected := [][]string{{"1", "a"}, {"2", "b"}, {"3", "c"}}
	if !reflect.DeepEqual(cols, expected) {
		t.Fatalf("Couldn't transpose %v: expected %v, got %v", rows, expected, cols)
	}
}

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
		surveySheet{
			types:  []string{"type1", "type2"},
			names:  []string{"name1", "name2"},
			labels: []string{"label1", "label2"},
		},
		choicesSheet{
			listNames: []string{"listname1", "listname2", "listname3"},
			names:     []string{"name1", "name2", "name3"},
			labels:    []string{"label1", "label2", "label3"},
		},
	}
	if !reflect.DeepEqual(xls, expected) {
		t.Fatalf("Error decoding %s: expected %v, got %v", fileName, expected, xls)
	}
}

func TestBuildChoicesOrigins(t *testing.T) {
	choices := choicesSheet{
		listNames: []string{"list1", "list2", "list1"},
		names:     []string{"elem1a", "elem2a", "elem1b"},
		labels:    []string{"label1a", "label2a", "label1b"},
	}
	_, choicesMap := buildChoicesOrigins(&choices)
	expected := map[string][]Choice{
		"list1": {{"elem1a", "label1a"}, {"elem1b", "label1b"}},
		"list2": {{"elem2a", "label2a"}},
	}
	if !reflect.DeepEqual(choicesMap, expected) {
		t.Fatalf("Error building choices origins of %v: expected %v, got %v",
			&choices, expected, choicesMap)
	}
}
