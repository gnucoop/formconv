package formats

import (
	"reflect"
	"testing"
)

func check(err error, t testing.TB) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeXls(t *testing.T) {
	fileName := "testdata/skeleton"
	expected := &XlsForm{
		Survey: []SurveyRow{
			{LineNum: 2, Type: "type1", Name: "name1", Label: "label1", Required: "yes"},
			{LineNum: 3, Type: "type2", Name: "name2", Label: "label2", Required: "yes"},
		},
		Choices: []ChoicesRow{
			{LineNum: 4, ListName: "listname1", Name: "name1", Label: "label1"},
			{LineNum: 6, ListName: "listname2", Name: "name2", Label: "label2"},
			{LineNum: 8, ListName: "listname3", Name: "name3", Label: "label3"},
		},
	}
	for _, ext := range []string{".xls", ".xlsx"} {
		xls, err := DecXlsFromFile(fileName + ext)
		check(err, t)
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

func BenchmarkPicaps(b *testing.B) {
	xls, err := DecXlsFromFile("testdata/Picaps_baseline_form.xlsx")
	check(err, b)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = Xls2ajf(xls)
		check(err, b)
	}
}
