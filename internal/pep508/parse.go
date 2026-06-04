package pep508

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

func ParseRequirement(input string) (Requirement, error) {
	p := &parser{input: input, pos: 0}
	return p.parseRequirement()
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseRequirement() (Requirement, error) {
	var req Requirement

	p.skipWhitespace()

	name := p.parseName()
	if name == "" {
		return req, fmt.Errorf("expected package name at position %d in %q", p.pos, p.input)
	}
	req.Name = types.NewPackageName(name)

	p.skipWhitespace()

	// Parse extras: [extra1, extra2]
	if p.peek() == '[' {
		extras, err := p.parseExtras()
		if err != nil {
			return req, err
		}
		req.Extras = extras
	}

	p.skipWhitespace()

	// Parse URL: @ https://...
	if p.peek() == '@' {
		p.pos++
		p.skipWhitespace()
		url := p.parseUntil(';')
		req.URL = strings.TrimSpace(url)
	} else {
		// Parse version specifiers
		specs, err := p.parseVersionSpecs()
		if err != nil {
			return req, err
		}
		req.Specifiers = specs
	}

	p.skipWhitespace()

	// Parse markers: ; python_version >= "3.8"
	if p.peek() == ';' {
		p.pos++
		p.skipWhitespace()
		marker, err := p.parseMarkerExpr()
		if err != nil {
			return req, err
		}
		req.Marker = marker
	}

	return req, nil
}

func (p *parser) parseName() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '_' || ch == '-' || ch == '.' || isAlphaNumeric(rune(ch)) {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}

func (p *parser) parseExtras() ([]types.ExtraName, error) {
	if p.peek() != '[' {
		return nil, nil
	}
	p.pos++ // consume '['

	var extras []types.ExtraName
	for {
		p.skipWhitespace()
		name := p.parseName()
		if name != "" {
			extras = append(extras, types.NewExtraName(name))
		}
		p.skipWhitespace()
		if p.peek() == ']' {
			p.pos++
			break
		}
		if p.peek() == ',' {
			p.pos++
			continue
		}
		return nil, fmt.Errorf("expected ',' or ']' in extras at position %d", p.pos)
	}
	return extras, nil
}

func (p *parser) parseVersionSpecs() (pep440.VersionSpecifiers, error) {
	p.skipWhitespace()

	// Check if there's a version specifier starting with an operator
	if p.pos >= len(p.input) || p.peek() == ';' {
		return nil, nil
	}

	ch := p.peek()
	if ch != '(' && ch != '>' && ch != '<' && ch != '=' && ch != '!' && ch != '~' {
		return nil, nil
	}

	// Handle parenthesized specifiers
	paren := false
	if ch == '(' {
		paren = true
		p.pos++
		p.skipWhitespace()
	}

	var endChar byte = ';'
	if paren {
		endChar = ')'
	}

	specStr := p.parseUntil(endChar)
	if paren {
		if p.peek() == ')' {
			p.pos++
		}
	}

	return pep440.ParseSpecifiers(specStr)
}

func (p *parser) parseMarkerExpr() (MarkerTree, error) {
	return p.parseMarkerOr()
}

func (p *parser) parseMarkerOr() (MarkerTree, error) {
	left, err := p.parseMarkerAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.matchKeyword("or") {
			p.skipWhitespace()
			right, err := p.parseMarkerAnd()
			if err != nil {
				return nil, err
			}
			left = &MarkerOr{Left: left, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *parser) parseMarkerAnd() (MarkerTree, error) {
	left, err := p.parseMarkerAtom()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.matchKeyword("and") {
			p.skipWhitespace()
			right, err := p.parseMarkerAtom()
			if err != nil {
				return nil, err
			}
			left = &MarkerAnd{Left: left, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *parser) parseMarkerAtom() (MarkerTree, error) {
	p.skipWhitespace()

	if p.peek() == '(' {
		p.pos++
		expr, err := p.parseMarkerOr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.peek() == ')' {
			p.pos++
		}
		return expr, nil
	}

	// Parse: variable op value OR value op variable
	left := p.parseMarkerValue()
	p.skipWhitespace()
	op, err := p.parseMarkerOp()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	right := p.parseMarkerValue()

	// Determine which side is the variable
	variable := left
	value := right
	if isQuoted(left) {
		variable = right
		value = left
	}
	value = unquote(value)

	return &MarkerExpr{Variable: variable, Op: op, Value: value}, nil
}

func (p *parser) parseMarkerValue() string {
	p.skipWhitespace()

	if p.peek() == '\'' || p.peek() == '"' {
		return p.parseQuotedString()
	}

	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '_' || ch == '.' || isAlphaNumeric(rune(ch)) {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}

func (p *parser) parseQuotedString() string {
	if p.pos >= len(p.input) {
		return ""
	}
	quote := p.input[p.pos]
	p.pos++
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != quote {
		p.pos++
	}
	result := p.input[start:p.pos]
	if p.pos < len(p.input) {
		p.pos++ // consume closing quote
	}
	return string(quote) + result + string(quote)
}

func (p *parser) parseMarkerOp() (MarkerOp, error) {
	p.skipWhitespace()

	if p.matchKeyword("not in") {
		return MarkerOpNotIn, nil
	}
	if p.matchKeyword("in") {
		return MarkerOpIn, nil
	}

	if p.pos+1 < len(p.input) {
		two := p.input[p.pos : p.pos+2]
		switch two {
		case "==":
			p.pos += 2
			return MarkerOpEqual, nil
		case "!=":
			p.pos += 2
			return MarkerOpNotEqual, nil
		case "<=":
			p.pos += 2
			return MarkerOpLessEqual, nil
		case ">=":
			p.pos += 2
			return MarkerOpGreaterEqual, nil
		}
	}

	if p.pos < len(p.input) {
		switch p.input[p.pos] {
		case '<':
			p.pos++
			return MarkerOpLess, nil
		case '>':
			p.pos++
			return MarkerOpGreater, nil
		}
	}

	return 0, fmt.Errorf("expected marker operator at position %d in %q", p.pos, p.input)
}

func (p *parser) matchKeyword(kw string) bool {
	if p.pos+len(kw) > len(p.input) {
		return false
	}
	if strings.ToLower(p.input[p.pos:p.pos+len(kw)]) == kw {
		// Must be followed by non-alphanumeric or end
		end := p.pos + len(kw)
		if end >= len(p.input) || !isAlphaNumeric(rune(p.input[end])) {
			p.pos = end
			return true
		}
	}
	return false
}

func (p *parser) parseUntil(ch byte) string {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != ch {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isQuoted(s string) bool {
	return len(s) >= 2 && (s[0] == '\'' || s[0] == '"')
}

func unquote(s string) string {
	if isQuoted(s) {
		return s[1 : len(s)-1]
	}
	return s
}
