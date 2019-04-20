package yymmdd

import "time"

func Format(time time.Time, layout string) string {
	_, tokens := Lexer(layout)
	ds := Parse(tokens)
	return ds.Format(time)
}
