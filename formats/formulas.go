package formats

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"
)

// preprocessFormula makes f scannable by text/scanner
func preprocessFormula(f string) string {
	// TODO: fix string quoting flaws.
	f = strings.ReplaceAll(f, "'", `"`)
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
	fieldName string // "." in formulas will be equivalent to ${fieldName}
	js        []byte
}

func (p *parser) Parse(formula, fieldName string) (js string, err error) {
	p.Scanner.Init(strings.NewReader(preprocessFormula(formula)))
	p.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings
	p.Error = scannerError
	p.Filename = ""
	p.fieldName = fieldName
	p.js = p.js[0:0]

	p.parseExpression(scanner.EOF)
	if p.ErrorCount > 0 {
		return "", errors.New(p.Filename)
	}
	return string(p.js), nil
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

func (p *parser) copy(tok byte) {
	p.consume(rune(tok))
	p.js = append(p.js, tok)
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
			p.js = append(p.js, p.TokenText()...)
		case '+', '-':
			if ch := p.peekNonspace(); ch == '+' || ch == '-' {
				p.unexpectedTokError(p.Next())
				return
			}
			p.js = append(p.js, byte(tok))
			continue
		case '$':
			p.consume('{')
			p.consume(scanner.Ident)
			p.js = append(p.js, p.TokenText()...)
			p.consume('}')
		case '.':
			if p.Peek() == '.' {
				p.error(`".." is not supported in formulas.`)
				return
			}
			p.js = append(p.js, p.fieldName...)
		case '(':
			p.js = append(p.js, '(')
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
		case '+', '-':
			p.js = append(p.js, ' ', byte(tok), ' ')
		case '*':
			p.js = append(p.js, '*')
		case '=':
			if p.Peek() == '=' {
				p.error(`Unexpected token "==". (did you mean "="?)`)
				return
			}
			p.js = append(p.js, " === "...)
		case '!':
			if p.Peek() != '=' {
				p.error(`Unary operator "!" not supported.`)
				return
			}
			p.consume('=')
			p.js = append(p.js, " !== "...)
		case '>':
			op := " > "
			if p.Peek() == '=' {
				p.consume('=')
				op = " >= "
			}
			p.js = append(p.js, op...)
		case '<':
			op := " < "
			if p.Peek() == '=' {
				p.consume('=')
				op = " <= "
			}
			p.js = append(p.js, op...)
		default:
			p.unexpectedTokError(tok)
			return
		}
	}
}

func (p *parser) parseOperatorIdent() {
	switch p.TokenText() {
	case "div":
		p.js = append(p.js, '/')
	case "mod":
		p.js = append(p.js, '%')
	case "and":
		p.js = append(p.js, " && "...)
	case "or":
		p.js = append(p.js, " || "...)
	default:
		p.unexpectedTokError(scanner.Ident)
	}
}

func (p *parser) parseEpressionIdent(expectedEnd rune) {
	switch ident := p.TokenText(); {
	case ident == "True" || ident == "False":
		p.js = append(p.js, strings.ToLower(ident)...)
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
		p.js = append(p.js, " ? "...)
		p.parseExpression(',') // then
		p.consume(',')
		p.js = append(p.js, " : "...)
		p.parseExpression(')') // else
		p.copy(')')
		return
	}
	if jsfunc, ok := func2jsfunc[name]; ok {
		// func(arg1, arg2...) becomes jsfunc(arg1, arg2...)
		p.js = append(p.js, jsfunc...)
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
		p.js = append(p.js, (")." + method + "(")...)
		p.parseFuncArgs()
		p.copy(')')
		return
	}
	if name == "string_length" {
		// string_length(s) becomes (s).length
		p.copy('(')
		p.parseExpression(')')
		p.copy(')')
		p.js = append(p.js, ".length"...)
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
		p.js = append(p.js, ", "...)
	}
}

var func2jsfunc = map[string]string{
	// Math:
	"math_max": "Math.max",
	"math_min": "Math.min",
	"int":      "Math.floor",
	"pow":      "Math.pow",
	"log":      "Math.log",
	"log10":    "Math.log10",
	"abs":      "Math.abs",
	"sin":      "Math.sin",
	"cos":      "Math.cos",
	"tan":      "Math.tan",
	"asin":     "Math.asin",
	"acos":     "Math.acos",
	"atan":     "Math.atan",
	"atan2":    "Math.atan2",
	"sqrt":     "Math.sqrt",
	"exp":      "Math.exp",
	"random":   "Math.random",
}
var func2jsmethod = map[string]string{
	// Strings:
	"contains":    "includes",
	"starts_with": "startsWith",
	"ends_with":   "endsWith",
	"substr":      "substring",
	"string":      "toString",
}
