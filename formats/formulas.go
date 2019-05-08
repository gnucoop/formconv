package formats

import (
	"errors"
	"fmt"
	"strings"
	"text/scanner"
)

// parser parses xlsform formulas and produces the JavaScript equivalent.
// Can't be used concurrently.
type parser struct {
	scanner.Scanner
	fieldName string
	js        []byte
}

func (p *parser) Parse(formula, fieldName string) (js string, err error) {
	p.Scanner.Init(strings.NewReader(formula))
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
func (p *parser) consume(expected rune) {
	if tok := p.Scan(); tok != expected {
		p.error(fmt.Sprintf("Expected %s, found %s.",
			scanner.TokenString(expected), scanner.TokenString(tok)))
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
			p.consume(')')
			p.js = append(p.js, ')')
		default:
			p.unexpectedTokError(tok)
			return
		}

		// Possible end of expression. expectedEnd can be:
		// EOF,
		// ')' for expressions between parentheses,
		// ',' for function arguments, in which case we also accept ')' instead of ','.
		// Note that we don't consume the end token.
		if tok := p.Peek(); tok == expectedEnd || tok == ')' && expectedEnd == ',' {
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
	case ident == "if":
		// if(cond, then, else) becomes (cond ? then : else)
		p.consume('(')
		p.js = append(p.js, '(')
		p.parseExpression(',') // cond
		p.consume(',')
		p.js = append(p.js, " ? "...)
		p.parseExpression(',') // then
		p.consume(',')
		p.js = append(p.js, " : "...)
		p.parseExpression(',') // else
		p.consume(')')
		p.js = append(p.js, ')')
	case p.Peek() == '(': // function call
		// TODO: check supported functions
		p.consume('(')
		p.js = append(p.js, '(')
		p.parseFuncArgs()
		p.consume(')')
		p.js = append(p.js, ')')
	default: // plain identifier
		p.js = append(p.js, ident...)
	}
}

func (p *parser) parseFuncArgs() {
	if p.Peek() == ')' { // empty argument list
		return
	}
	for {
		p.parseExpression(',') // argument

		if p.Peek() == ')' { // possible end of argument list
			return
		}

		p.consume(',')
		p.js = append(p.js, ", "...)
	}
}
