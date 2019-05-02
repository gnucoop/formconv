package formats

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func check(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func logFatalDiff(t testing.TB, a, b interface{}) {
	t.Helper()
	// Don't use pretty.Ldiff, it doesn't call t.Helper().
	for _, diff := range pretty.Diff(a, b) {
		t.Log(diff)
	}
	t.FailNow()
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
		check(t, err)
		if !reflect.DeepEqual(xls, expected) {
			t.Errorf("Error decoding %s, unexpected result:", fileName+ext)
			logFatalDiff(t, xls, expected)
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
	expected := []ChoicesOrigin{{
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
	if !reflect.DeepEqual(choices, expected) {
		t.Errorf("Error building choices origins of\n%# v\nunexpected result:",
			pretty.Formatter(choicesSheet))
		logFatalDiff(t, choices, expected)
	}
}

func TestPreprocessGroups(t *testing.T) {
	errSurveys := [][]SurveyRow{
		{
			{Type: beginGroup}, {Type: beginRepeat}, {Type: endRepeat}, {Type: endGroup},
		}, {
			{Type: endRepeat},
		}, {
			{Type: beginRepeat}, {Type: endGroup}, {Type: endRepeat},
		}, {
			{Type: beginRepeat}, {Type: beginGroup},
		}, {
			{Type: beginRepeat}, {Type: endRepeat}, {Type: "text"},
		},
	}
	for _, errSurvey := range errSurveys {
		_, err := preprocessGroups(errSurvey)
		if err == nil {
			t.Fatalf("Couldn't find error in erroneus survey:\n%# v", pretty.Formatter(errSurvey))
		}
	}

	survey := []SurveyRow{{Type: "text"}}
	processed, err := preprocessGroups(survey)
	check(t, err)
	expected := []SurveyRow{
		{Type: beginGroup, Name: "global"},
		{Type: beginGroup, Name: "form", Label: "Form"},
		{Type: "text"},
		{Type: endGroup},
		{Type: endGroup},
	}
	if !reflect.DeepEqual(processed, expected) {
		t.Error(`Error wrapping []SurveyRow{{Type: "text"}}, unexpected result:`)
		logFatalDiff(t, processed, expected)
	}
}

func BenchmarkDecXls(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := DecXlsFromFile("testdata/Picaps_baseline_form.xls")
		check(b, err)
	}
}

func BenchmarkDecXlsx(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := DecXlsFromFile("testdata/Picaps_baseline_form.xlsx")
		check(b, err)
	}
}

func BenchmarkXls2ajf(b *testing.B) {
	xls, err := DecXlsFromFile("testdata/Picaps_baseline_form.xlsx")
	check(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = Xls2ajf(xls)
		check(b, err)
	}
}
