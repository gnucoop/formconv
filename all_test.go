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
	xls, err := decXlsFromFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	expected := &xlsForm{
		survey{
			types:  []string{"type1", "type2"},
			names:  []string{"name1", "name2"},
			labels: []string{"label1", "label2"},
		},
		choices{
			listNames: []string{"listname1", "listname2", "listname3"},
			names:     []string{"name1", "name2", "name3"},
			labels:    []string{"label1", "label2", "label3"},
		},
	}
	if !reflect.DeepEqual(xls, expected) {
		t.Fatalf("Error decoding %s: expected %v, got %v", fileName, expected, xls)
	}
}
