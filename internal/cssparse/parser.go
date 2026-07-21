// Package cssparse provides the small, bounded CSS rule/declaration parser
// shared by Paper's HTML and SVG renderers. It intentionally models the
// stylesheet surface Paper consumes instead of attempting browser-complete CSS.
package cssparse

import (
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper/internal/htmllimits"
)

type RuleKind uint8

const (
	QualifiedRule RuleKind = iota
	AtRule
)

type Declaration struct {
	Property  string
	Value     string
	Important bool
}

type Rule struct {
	Kind         RuleKind
	Name         string
	Prelude      string
	Selectors    []string
	Declarations []Declaration
	Rules        []*Rule
}

func Parse(text string) ([]*Rule, error) {
	return ParseWithMaxRules(text, htmllimits.NoLimits().MaxStyleRules)
}

func ParseWithMaxRules(text string, maxRules int) ([]*Rule, error) {
	parser := stylesheetParser{text: text, maxRules: maxRules}
	return parser.parseRuleList(false)
}

type stylesheetParser struct {
	text     string
	position int
	count    int
	maxRules int
}

//nolint:gocognit // A stylesheet's delimiter-driven state machine is clearest in one loop.
func (parser *stylesheetParser) parseRuleList(stopAtBrace bool) ([]*Rule, error) {
	var rules []*Rule
	for {
		parser.skipSpaceAndComments()
		if parser.position >= len(parser.text) {
			return rules, nil
		}
		if stopAtBrace && parser.text[parser.position] == '}' {
			parser.position++
			return rules, nil
		}

		header, delimiter := parser.readHeader()
		header = strings.TrimSpace(stripComments(header))
		if delimiter == 0 {
			return rules, nil
		}
		if header == "" {
			if delimiter == '}' && stopAtBrace {
				return rules, nil
			}
			continue
		}
		if delimiter == ';' {
			if !strings.HasPrefix(header, "@") {
				continue
			}
			err := parser.increment()
			if err != nil {
				return rules, err
			}
			name, prelude := splitAtRuleHeader(header)
			rules = append(rules, &Rule{Kind: AtRule, Name: name, Prelude: prelude})
			continue
		}
		if delimiter == '}' {
			if stopAtBrace {
				return rules, nil
			}
			continue
		}

		err := parser.increment()
		if err != nil {
			return rules, err
		}
		if strings.HasPrefix(header, "@") {
			name, prelude := splitAtRuleHeader(header)
			rule := &Rule{Kind: AtRule, Name: name, Prelude: prelude}
			if isNestedRuleAtRule(name) {
				nested, err := parser.parseRuleList(true)
				rule.Rules = nested
				rules = append(rules, rule)
				if err != nil {
					return rules, err
				}
				continue
			}
			body, closed := parser.readBlock()
			rule.Declarations = parseDeclarations(body)
			rules = append(rules, rule)
			if !closed {
				return rules, nil
			}
			continue
		}

		body, closed := parser.readBlock()
		rules = append(rules, &Rule{
			Kind:         QualifiedRule,
			Selectors:    SplitSelectors(header),
			Declarations: parseDeclarations(body),
		})
		if !closed {
			return rules, nil
		}
	}
}

func (parser *stylesheetParser) increment() error {
	parser.count++
	if htmllimits.IntExceeded(parser.maxRules, parser.count) {
		return fmt.Errorf("%w: rule count %d exceeds limit %d", htmllimits.ErrStyleRulesTooLarge, parser.count, parser.maxRules)
	}
	return nil
}

// readHeader consumes through a top-level `{`, `;`, or `}`. Parentheses,
// attribute brackets, quoted strings, escapes, and comments cannot terminate a
// header. The returned delimiter has already been consumed.
func (parser *stylesheetParser) readHeader() (string, byte) {
	start := parser.position
	depthParen, depthBracket := 0, 0
	for parser.position < len(parser.text) {
		current := parser.text[parser.position]
		if current == '/' && parser.position+1 < len(parser.text) && parser.text[parser.position+1] == '*' {
			parser.skipComment()
			continue
		}
		if current == '\'' || current == '"' {
			parser.skipString(current)
			continue
		}
		switch current {
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case '{', ';', '}':
			if depthParen == 0 && depthBracket == 0 {
				header := parser.text[start:parser.position]
				parser.position++
				return header, current
			}
		}
		parser.position++
	}
	return parser.text[start:], 0
}

// readBlock is called immediately after an opening brace. It returns the raw
// block body and whether a matching brace was found. Nested braces occur in
// CSS custom values and are retained rather than treated as a parse failure.
func (parser *stylesheetParser) readBlock() (string, bool) {
	start := parser.position
	depth := 1
	for parser.position < len(parser.text) {
		current := parser.text[parser.position]
		if current == '/' && parser.position+1 < len(parser.text) && parser.text[parser.position+1] == '*' {
			parser.skipComment()
			continue
		}
		if current == '\'' || current == '"' {
			parser.skipString(current)
			continue
		}
		switch current {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				body := parser.text[start:parser.position]
				parser.position++
				return body, true
			}
		}
		parser.position++
	}
	return parser.text[start:], false
}

func (parser *stylesheetParser) skipSpaceAndComments() {
	for parser.position < len(parser.text) {
		if parser.text[parser.position] == '/' && parser.position+1 < len(parser.text) && parser.text[parser.position+1] == '*' {
			parser.skipComment()
			continue
		}
		if !isCSSSpace(parser.text[parser.position]) {
			return
		}
		parser.position++
	}
}

func (parser *stylesheetParser) skipComment() {
	parser.position += 2
	for parser.position+1 < len(parser.text) {
		if parser.text[parser.position] == '*' && parser.text[parser.position+1] == '/' {
			parser.position += 2
			return
		}
		parser.position++
	}
	parser.position = len(parser.text)
}

func (parser *stylesheetParser) skipString(quote byte) {
	parser.position++
	for parser.position < len(parser.text) {
		if parser.text[parser.position] == '\\' && parser.position+1 < len(parser.text) {
			parser.position += 2
			continue
		}
		if parser.text[parser.position] == quote {
			parser.position++
			return
		}
		parser.position++
	}
}

func splitAtRuleHeader(header string) (string, string) {
	end := 1
	for end < len(header) && isAtRuleNameCharacter(header[end]) {
		end++
	}
	return strings.ToLower(header[:end]), strings.TrimSpace(header[end:])
}

func isAtRuleNameCharacter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '-' || value == '_'
}

func isNestedRuleAtRule(name string) bool {
	switch name {
	case "@media", "@supports", "@layer", "@document":
		return true
	default:
		return false
	}
}

func parseDeclarations(body string) []Declaration {
	var declarations []Declaration
	for _, part := range splitTopLevel(stripComments(body), ';') {
		colon := topLevelIndex(part, ':')
		if colon < 0 {
			continue
		}
		property := strings.TrimSpace(part[:colon])
		if property == "" {
			continue
		}
		value, important := StripImportantSuffix(strings.TrimSpace(part[colon+1:]))
		declarations = append(declarations, Declaration{Property: property, Value: value, Important: important})
	}
	return declarations
}

// SplitSelectors splits a selector list on only top-level commas. It keeps
// selector text (including escapes and function/attribute contents) intact.
func SplitSelectors(value string) []string {
	var selectors []string
	for _, selector := range splitTopLevel(value, ',') {
		if selector = strings.TrimSpace(selector); selector != "" {
			selectors = append(selectors, selector)
		}
	}
	return selectors
}

func StripImportantSuffix(value string) (string, bool) {
	lower := strings.ToLower(value)
	if index := strings.LastIndex(lower, "!important"); index >= 0 && strings.TrimSpace(value[index+len("!important"):]) == "" {
		return strings.TrimSpace(value[:index]), true
	}
	return value, false
}

func splitTopLevel(value string, separator byte) []string {
	var parts []string
	start, parens, brackets, braces := 0, 0, 0, 0
	for index := 0; index < len(value); index++ {
		if value[index] == '\'' || value[index] == '"' {
			index = skipCSSString(value, index, value[index]) - 1
			continue
		}
		switch value[index] {
		case '(':
			parens++
		case ')':
			if parens > 0 {
				parens--
			}
		case '[':
			brackets++
		case ']':
			if brackets > 0 {
				brackets--
			}
		case '{':
			braces++
		case '}':
			if braces > 0 {
				braces--
			}
		default:
			if value[index] == separator && parens == 0 && brackets == 0 && braces == 0 {
				parts = append(parts, value[start:index])
				start = index + 1
			}
		}
	}
	parts = append(parts, value[start:])
	return parts
}

func topLevelIndex(value string, separator byte) int {
	parts := splitTopLevel(value, separator)
	if len(parts) == 1 {
		return -1
	}
	return len(parts[0])
}

func stripComments(value string) string {
	var result strings.Builder
	for index := 0; index < len(value); {
		if value[index] == '\'' || value[index] == '"' {
			next := skipCSSString(value, index, value[index])
			result.WriteString(value[index:next])
			index = next
			continue
		}
		if value[index] == '/' && index+1 < len(value) && value[index+1] == '*' {
			result.WriteByte(' ')
			index += 2
			for index+1 < len(value) && (value[index] != '*' || value[index+1] != '/') {
				index++
			}
			if index+1 < len(value) {
				index += 2
			}
			continue
		}
		result.WriteByte(value[index])
		index++
	}
	return result.String()
}

func skipCSSString(value string, index int, quote byte) int {
	index++
	for index < len(value) {
		if value[index] == '\\' && index+1 < len(value) {
			index += 2
			continue
		}
		if value[index] == quote {
			return index + 1
		}
		index++
	}
	return index
}

func isCSSSpace(value byte) bool {
	switch value {
	case ' ', '\n', '\r', '\t', '\f':
		return true
	default:
		return false
	}
}
