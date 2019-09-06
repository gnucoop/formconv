package formats

import (
	"io/ioutil"
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
		{{Type: beginGroup}, {Type: beginRepeat}, {Type: endRepeat}, {Type: endGroup}},
		{{Type: endRepeat}},
		{{Type: beginRepeat}, {Type: endGroup}, {Type: endRepeat}},
		{{Type: beginRepeat}, {Type: beginGroup}},
		{{Type: beginRepeat}, {Type: endRepeat}, {Type: "text"}},
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

func TestNonformulaFeatures(t *testing.T) {
	in := "testdata/noformulas.xlsx"
	out := "testdata/noformulas.json"
	oracle := "testdata/noformulas_oracle.json"

	xls, err := DecXlsFromFile(in)
	check(t, err)
	ajf, err := Convert(xls)
	check(t, err)
	err = EncJsonToFile(out, ajf)
	check(t, err)
	result, err := ioutil.ReadFile(out)
	check(t, err)
	expected, err := ioutil.ReadFile(oracle)
	check(t, err)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Unexpected result. Check the differences between %s and %s", out, oracle)
	}
}

func TestFormulaParser(t *testing.T) {
	var p parser

	formulas := map[string]string{
		`123 + 345.78 - "hello"`:                 `123 + 345.78 - "hello"`,
		`. = ${ident} and 1 != 2`:                `fieldName === ident && 1 !== 2`,
		`(  (1 - 2) * (3 + 4)  )`:                `((1 - 2)*(3 + 4))`,
		`1 + 2 - 3 * 4 div 5 mod 6`:              `1 + 2 - 3*4/5%6`,
		`1 < 2 and 3 <= 4 or 5 > 6 and 7 >= 8`:   `1 < 2 && 3 <= 4 || 5 > 6 && 7 >= 8`,
		`True = False`:                           `true === false`,
		`pow(sin(7) + (9))`:                      `Math.pow(Math.sin(7) + (9))`,
		`contains("abc", "b")`:                   `("abc").includes("b")`,
		`pi() and true()`:                        `Math.PI && true`,
		`if("banana", 1, 2)`:                     `("banana" ? 1 : 2)`,
		`regex("s", "re")`:                       `(("s").match("re") !== null)`,
		`string-length("hello")`:                 `("hello").length`,
		`exp10(${x})`:                            `Math.pow(10, x)`,
		`+(-(+(-5)))`:                            `+(-(+(-5)))`,
		`'hello \n \123 \xab \uabcd \Uabcd1234'`: `'hello \n \123 \xab \uabcd \Uabcd1234'`,
	}
	for formula, expected := range formulas {
		js, err := p.Parse(formula, "formula", "fieldName")
		if err != nil {
			t.Fatalf("Error converting formula:\n%s\n%s\n", formula, err)
		}
		if js != expected {
			t.Fatalf("Error converting formula:\n%s\nexpected:\n%s\ngot:\n%s\n", formula, expected, js)
		}
	}

	errFormulas := []string{
		"5++", "$dollar", "..", "((1)", ")(1)", "1 == 2", "!True", "1 << 2",
		"True andd False", "plainIdent > 3", "unknownFunc(7)",
		`'\g'`, `'\12'`, `'\xax'`,
	}
	for _, formula := range errFormulas {
		_, err := p.Parse(formula, "formula", "fieldName")
		if err == nil {
			t.Fatalf("Erroneus formula parsed successfully: %q", formula)
		}
	}
}

func TestFormulaFeatures(t *testing.T) {
	in := "testdata/formulas.xlsx"
	out := "testdata/formulas.json"
	oracle := "testdata/formulas_oracle.json"

	xls, err := DecXlsFromFile(in)
	check(t, err)
	ajf, err := Convert(xls)
	check(t, err)
	err = EncJsonToFile(out, ajf)
	check(t, err)
	result, err := ioutil.ReadFile(out)
	check(t, err)
	expected, err := ioutil.ReadFile(oracle)
	check(t, err)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Unexpected result. Check the differences between %s and %s", out, oracle)
	}
}

func TestListLanguages(t *testing.T) {
	if l := ListLanguages(nil); l != nil {
		t.Fatalf("ListLanguages(nil) expected to be nil, found %v", l)
	}
	rows := [][]string{
		{"type", "label", "label::English (en)", "label::French (fr)", "label::Italian (it)"},
	}
	list := ListLanguages(rows)
	expected := map[string]bool{"fr": true, "it": true}
	if !reflect.DeepEqual(list, expected) {
		t.Fatalf("Error listing languages of\n%v\nexpected: %v\nfound: %v", rows, list, expected)
	}
}

func TestTranslationIndex(t *testing.T) {
	if i := translationIndex(nil, "foo", "bar"); i != -1 {
		t.Fatalf("translationIndex(nil, \"foo\", \"bar\") expected to be -1, found %d", i)
	}
	row := []string{"type", "label", "label::English (en)", "label::French (fr)", "label::Italian (it)"}
	if i := translationIndex(row, "label", "fr"); i != 3 {
		t.Fatalf("translationIndex(%v, \"label\", \"fr\")\nexpected to be 3, found %d", row, i)
	}
	if i := translationIndex(row, "type", "en"); i != -1 {
		t.Fatalf("translationIndex(%v, \"type\", \"en\")\nexpected to be -1, found %d", row, i)
	}
}

func TestTranslation(t *testing.T) {
	if tr := Translation(nil, "en"); tr != nil {
		t.Fatalf("Translation(nil, \"en\") expected to be nil, found %v", tr)
	}
	rows := [][]string{
		{"", "", ""},
		{"type", "label", "label::Italian (it)"},
		{"text", "cheese", "formaggio"},
		{"number", "bread", "pane"},
	}
	tr := Translation(rows, "it")
	expected := map[string]string{"cheese": "formaggio", "bread": "pane"}
	if !reflect.DeepEqual(tr, expected) {
		t.Fatalf("Error translating %v\nexpected: %v\n got: %v", rows, expected, tr)
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
		_, err = Convert(xls)
		check(b, err)
	}
}
