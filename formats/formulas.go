package formats

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"
)

func preprocessFormula(f string) string {
	for old, new := range dumbFuncNames {
		f = strings.ReplaceAll(f, old, new)
	}
	return f
}

var dumbFuncNames = map[string]string{
	"starts-with":         "starts_with",
	"ends-with":           "ends_with",
	"substring-before":    "substring_before",
	"substring-after":     "substring_after",
	"string-length":       "string_length",
	"boolean-from-string": "boolean_from_string",
}

// parser parses xlsform formulas and produces the JavaScript equivalent.
// Can't be used concurrently.
type parser struct {
	scanner.Scanner
	strings.Builder
	fieldName string // in formulas, "." will be equivalent to "${fieldName}"
}

func (p *parser) Parse(formula, fieldName string) (string, error) {
	p.Scanner.Init(strings.NewReader(preprocessFormula(formula)))
	p.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings
	p.Error = scannerError
	p.Filename = ""

	p.Builder.Reset()
	p.Grow(len(formula) * 2)

	p.fieldName = fieldName

	p.parseExpression(scanner.EOF)
	if p.ErrorCount > 0 {
		return "", errors.New(p.Filename)
	}
	return p.Builder.String(), nil
}

// When the first error is encountered, it is stored inside Scanner.Position.Filename
func scannerError(s *scanner.Scanner, msg string) {
	if s.ErrorCount == 1 {
		s.Filename = msg
	}
}
func (p *parser) error(msg string) {
	p.ErrorCount++
	scannerError(&p.Scanner, msg)
}

func (p *parser) unexpectedTokError(tok rune) {
	p.error(fmt.Sprintf("Unexpected token %s.", scanner.TokenString(tok)))
}

func (p *parser) consume(tok rune) {
	t := p.Scan()
	if t != tok {
		p.error(fmt.Sprintf("Expected %s, found %s.",
			scanner.TokenString(tok), scanner.TokenString(t)))
	}
}

func (p *parser) copy(ch byte) {
	p.consume(rune(ch))
	p.WriteByte(ch)
}

func (p *parser) peekNonspace() rune {
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
func (p *parser) scanString(quote rune) {
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

func (p *parser) scanEscape(quote rune) {
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

func (p *parser) scanDigits(base, n int) {
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

func (p *parser) parseExpression(expectedEnd rune) {
	if expectedEnd != scanner.EOF && expectedEnd != ')' && expectedEnd != ',' {
		panic("invalid expectedEnd")
	}

	for {
		// Expression.
		switch tok := p.Scan(); tok {
		case scanner.Ident:
			p.parseEpressionIdent(expectedEnd)
		case scanner.Int, scanner.Float, scanner.String:
			p.WriteString(p.TokenText())
		case '\'':
			p.scanString('\'')
		case '+', '-':
			if ch := p.peekNonspace(); ch == '+' || ch == '-' {
				p.unexpectedTokError(p.Next())
				return
			}
			p.WriteByte(byte(tok))
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
				p.error(`Unary operator "!" not supported.`)
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

func (p *parser) parseOperatorIdent() {
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

func (p *parser) parseEpressionIdent(expectedEnd rune) {
	switch ident := p.TokenText(); {
	case ident == "True":
		p.WriteString("true")
	case ident == "False":
		p.WriteString("false")
	case p.Peek() == '(':
		p.parseFunctionCall()
	default:
		p.unexpectedTokError(scanner.Ident)
	}
}

func (p *parser) parseFunctionCall() {
	name := p.TokenText()
	if name == "if" {
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
		return
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
	if name == "string_length" {
		// string_length(s) becomes (s).length
		p.copy('(')
		p.parseExpression(')')
		p.copy(')')
		p.WriteString(".length")
		return
	}
	p.error(fmt.Sprintf("Unsupported function %q.", name))
}

func (p *parser) parseFuncArgs() {
	if p.peekNonspace() == ')' { // empty argument list
		return
	}
	for {
		p.parseExpression(',') // argument
		if p.peekNonspace() == ')' {
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
}
var func2jsmethod = map[string]string{
	// Strings:
	"contains":    "includes",
	"starts_with": "startsWith",
	"ends_with":   "endsWith",
	"substr":      "substring",
	"string":      "toString",
}
