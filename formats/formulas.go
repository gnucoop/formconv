package formats

import (
	"fmt"
	"strings"
	"text/scanner"
)

// formulaParser parses xlsform formulas and produces the JavaScript equivalent.
// Can't be used concurrently.
type formulaParser struct {
	scanner.Scanner
	strings.Builder
	fieldName string // in formulas, "." will be equivalent to "${fieldName}"
	err       error
}

func (p *formulaParser) Parse(formula, formulaName, fieldName string) (js string, err error) {
	if strings.HasPrefix(formula, "js:") {
		return strings.TrimSpace(formula[3:]), nil
	}

	p.Scanner.Init(strings.NewReader(formula))
	p.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings
	p.Error = func(_ *scanner.Scanner, msg string) { p.error(msg) }
	p.Filename = formulaName

	p.Builder.Reset()
	p.Grow(len(formula) * 2)

	p.fieldName = fieldName
	p.err = nil

	p.parseExpression(scanner.EOF)
	if p.err != nil {
		return "", p.err
	}
	return p.Builder.String(), nil
}

func (p *formulaParser) error(msg string) {
	if p.err != nil {
		return
	}
	p.err = fmt.Errorf("formula %s:%d:%d: %s", p.Filename, p.Line, p.Column, msg)
}

func (p *formulaParser) unexpectedTokError(tok rune) {
	tokString := scanner.TokenString(tok)
	if tok == scanner.Ident || tok == scanner.Int || tok == scanner.Float {
		tokString = p.TokenText()
	}
	p.error(fmt.Sprintf("Unexpected token %s", tokString))
}

func (p *formulaParser) consume(ch rune) {
	tok := p.Scan()
	if tok != ch {
		p.error(fmt.Sprintf("Expected %s, found %s.",
			scanner.TokenString(ch), scanner.TokenString(tok)))
	}
}

func (p *formulaParser) copy(ch rune) {
	p.consume(ch)
	p.WriteRune(ch)
}

func (p *formulaParser) peekNonspace() rune {
	for {
		ch := p.Peek()
		if p.Whitespace&(1<<uint(ch)) == 0 || ch == scanner.EOF { // (not a whitespace) or EOF
			return ch
		}
		p.Next()
	}
}

// scanString is used to scan single-quoted strings.
// The code is adapted from Scanner.scanString.
func (p *formulaParser) scanString(quote rune) {
	// Initial quote has already been scanned.
	p.WriteRune(quote)
	for {
		ch := p.Next()
		if ch == '\n' || ch < 0 {
			p.error("String literal not terminated.")
			return
		}
		if ch == '\\' {
			p.scanEscape(quote)
		} else {
			p.WriteRune(ch)
		}
		if ch == quote {
			return
		}
	}
}

func (p *formulaParser) scanEscape(quote rune) {
	// Initial \ has already been scanned.
	p.WriteByte('\\')
	switch p.Peek() {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		p.WriteRune(p.Next())
	case '0', '1', '2', '3', '4', '5', '6', '7':
		p.scanDigits(8, 3)
	case 'x':
		p.WriteRune(p.Next())
		p.scanDigits(16, 2)
	case 'u':
		p.WriteRune(p.Next())
		p.scanDigits(16, 4)
	case 'U':
		p.WriteRune(p.Next())
		p.scanDigits(16, 8)
	default:
		p.error("Illegal char escape.")
	}
}

func (p *formulaParser) scanDigits(base, n int) {
	for i := 0; i < n; i++ {
		ch := p.Next()
		if digitVal(ch) >= base {
			p.error("Illegal char escape.")
			return
		}
		p.WriteRune(ch)
	}
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (p *formulaParser) parseExpression(expectedEnd rune) {
	if expectedEnd != scanner.EOF && expectedEnd != ')' && expectedEnd != ',' {
		panic("invalid expectedEnd")
	}

	for {
		// Expression.
		switch tok := p.Scan(); tok {
		case scanner.Ident:
			p.parseExpressionIdent(expectedEnd)
		case scanner.Int, scanner.Float, scanner.String:
			p.WriteString(p.TokenText())
		case '\'':
			p.scanString('\'')
		case '+', '-':
			if ch := p.peekNonspace(); ch == '+' || ch == '-' {
				p.unexpectedTokError(p.Next())
				return
			}
			p.WriteRune(tok)
			continue
		case '$':
			p.consume('{')
			p.consume(scanner.Ident)
			p.WriteString(p.TokenText())
			p.consume('}')
		case '.':
			if p.Peek() == '.' {
				p.error(`".." is not supported in formulas.`)
				return
			}
			p.WriteString(p.fieldName)
		case '(':
			p.WriteByte('(')
			p.parseExpression(')')
			p.copy(')')
		default:
			p.unexpectedTokError(tok)
			return
		}

		// Possible end of expression. expectedEnd can be:
		// EOF,
		// ')' for expressions between parentheses,
		// ',' for function arguments, in which case we also accept ')' instead of ','.
		// Note that we don't consume the end token.
		if tok := p.peekNonspace(); tok == expectedEnd || (tok == ')' && expectedEnd == ',') {
			return
		}

		// Operator.
		switch tok := p.Scan(); tok {
		case scanner.Ident:
			p.parseOperatorIdent()
		case '+':
			p.WriteString(" + ")
		case '-':
			p.WriteString(" - ")
		case '*':
			p.WriteByte('*')
		case '=':
			if p.Peek() == '=' {
				p.error(`Unexpected token "==". (did you mean "="?)`)
				return
			}
			p.WriteString(" === ")
		case '!':
			if p.Peek() != '=' {
				p.error(`Unary operator "!" not supported, use "not" function.`)
				return
			}
			p.consume('=')
			p.WriteString(" !== ")
		case '>':
			op := " > "
			if p.Peek() == '=' {
				p.consume('=')
				op = " >= "
			}
			p.WriteString(op)
		case '<':
			op := " < "
			if p.Peek() == '=' {
				p.consume('=')
				op = " <= "
			}
			p.WriteString(op)
		default:
			p.unexpectedTokError(tok)
			return
		}
	}
}

func (p *formulaParser) parseOperatorIdent() {
	switch p.TokenText() {
	case "div":
		p.WriteByte('/')
	case "mod":
		p.WriteByte('%')
	case "and":
		p.WriteString(" && ")
	case "or":
		p.WriteString(" || ")
	default:
		p.unexpectedTokError(scanner.Ident)
	}
}

// parseExpressionIdent parses an expression that starts with an identifier (already scanned).
// It has to deal with the following function names that contain a minus:
// count-selected, starts-with, ends-with, substring-before,
// substring-after, string-length, boolean-from-string.
func (p *formulaParser) parseExpressionIdent(expectedEnd rune) {
	if p.Peek() == '(' {
		p.parseFuncCall()
		return
	}
	switch p.TokenText() {
	case "True":
		p.WriteString("true")
	case "False":
		p.WriteString("false")
	case "count", "starts", "ends", "substring", "string", "boolean":
		p.parseFuncCall()
	default:
		if p.Filename == "choice_filter" {
			// Formulas in choice filters can have unbound identifiers,
			// which must be interpreted as fields of the choice.
			p.WriteString("$choice.")
			p.WriteString(p.TokenText())
		} else {
			p.error(fmt.Sprintf("Unknown identifier %q.", p.TokenText()))
		}
	}
}

func (p *formulaParser) parseFuncCall() {
	name := p.TokenText()
	for p.Peek() == '-' {
		p.consume('-')
		name += "-"
		p.consume(scanner.Ident)
		name += p.TokenText()
	}

	if jsfunc, ok := func2jsfunc[name]; ok {
		// func(arg1, arg2...) becomes jsfunc(arg1, arg2...)
		p.WriteString(jsfunc)
		p.copy('(')
		p.parseFuncArgs()
		p.copy(')')
		return
	}
	if method, ok := func2jsmethod[name]; ok {
		// func(arg1, arg2...) becomes (arg1).method(arg2...)
		p.copy('(')
		p.parseExpression(',') // arg1
		p.consume(',')
		p.WriteString(").")
		p.WriteString(method)
		p.WriteByte('(')
		p.parseFuncArgs()
		p.copy(')')
		return
	}
	if constant, ok := func2jsconstant[name]; ok {
		// func() becomes constant
		p.consume('(')
		p.consume(')')
		p.WriteString(constant)
		return
	}
	switch name {
	case "if":
		// if(cond, then, else) becomes (cond ? then : else)
		p.copy('(')
		p.parseExpression(',') // cond
		p.consume(',')
		p.WriteString(" ? ")
		p.parseExpression(',') // then
		p.consume(',')
		p.WriteString(" : ")
		p.parseExpression(')') // else
		p.copy(')')
	case "regex":
		// regex(s, re) becomes ((s).match(re) !== null)
		p.consume('(')
		p.WriteString("((")
		p.parseExpression(',') // s
		p.consume(',')
		p.WriteString(").match(")
		p.parseExpression(')') // re
		p.consume(')')
		p.WriteString(") !== null)")
	case "string-length", "count-selected":
		// string-length(s) and count-selected(s) become (s).length
		p.copy('(')
		p.parseExpression(')')
		p.copy(')')
		p.WriteString(".length")
	case "exp10":
		// exp10(x) becomes Math.pow(10, x)
		p.consume('(')
		p.WriteString("Math.pow(10, ")
		p.parseExpression(')')
		p.copy(')')
	default:
		p.error(fmt.Sprintf("Unsupported function %q.", name))
	}
}

func (p *formulaParser) parseFuncArgs() {
	if p.peekNonspace() == ')' { // empty argument list
		return
	}
	for {
		// parseExpression always consumes input or sets err,
		// so this loop should be finite.
		p.parseExpression(',') // argument
		if p.err != nil || p.peekNonspace() == ')' {
			return
		}
		p.consume(',')
		p.WriteString(", ")
	}
}

var func2jsfunc = map[string]string{
	// Math:
	"max":    "Math.max",
	"min":    "Math.min",
	"int":    "Math.floor",
	"pow":    "Math.pow",
	"log":    "Math.log",
	"log10":  "Math.log10",
	"abs":    "Math.abs",
	"sin":    "Math.sin",
	"cos":    "Math.cos",
	"tan":    "Math.tan",
	"asin":   "Math.asin",
	"acos":   "Math.acos",
	"atan":   "Math.atan",
	"atan2":  "Math.atan2",
	"sqrt":   "Math.sqrt",
	"exp":    "Math.exp",
	"random": "Math.random",
	"round":  "round",

	// Conversion functions:
	"string":  "String",
	"number":  "Number",
	"boolean": "Boolean",

	// Others:
	"not":      "!",
	"selected": "valueInChoice",
}
var func2jsmethod = map[string]string{
	// Strings:
	"contains":    "includes",
	"starts-with": "startsWith",
	"ends-with":   "endsWith",
	"substr":      "substring",
	"concat":      "concat",
}
var func2jsconstant = map[string]string{
	"pi":    "Math.PI",
	"true":  "true",
	"false": "false",
}
