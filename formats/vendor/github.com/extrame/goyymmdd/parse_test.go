package yymmdd

import (
	"testing"
	"time"
)

var date = time.Date(2018, time.March, 15, 12, 0, 0, 0, time.UTC)

var tests = []struct {
	format   string
	expected string
}{
	{"yy-", "18-"},
	{"yyyymm", "201803"},
	{"yyyy-mm", "2018-03"},
	{"mmm yyyy", "Mar 2018"},
	{"yyyy-mm-dd", "2018-03-15"},
}

func TestParse(t *testing.T) {
	for i, testCase := range tests {
		_, tokens := Lexer(testCase.format)
		ds := Parse(tokens)
		actual := ds.Format(date)
		if actual != testCase.expected {
			t.Errorf("Case index %d failed. Expected: %s, Got: %s", i, testCase.expected, actual)
		}
	}
}
