package translate

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/tdewolff/parse/v2"
	tcss "github.com/tdewolff/parse/v2/css"
)

// This file provides a minimal CSS rule tree built on top of the streaming
// github.com/tdewolff/parse/v2/css parser. It replaces github.com/aymerick/douceur
// (which pulled in github.com/gorilla/css) while preserving exactly the surface
// the rest of the package consumed: a flat list of rules, each either a
// qualified rule (selector strings + declarations) or an at-rule (@media with
// nested rules, @font-face with declarations).

// cssRuleKind distinguishes qualified rules from at-rules.
type cssRuleKind int

const (
	qualifiedRule cssRuleKind = iota
	atRule
)

// cssDeclaration is a single `property: value` pair. value has any trailing
// `!important` stripped, matching douceur's Declaration.Value semantics.
type cssDeclaration struct {
	property string
	value    string
}

// cssRule mirrors the minimal subset of douceur's css.Rule that the translator
// relied on. For atRule, name is e.g. "@media"/"@font-face", prelude is the raw
// text between the name and the block, and rules holds nested qualified rules
// (used by @media). For qualifiedRule, selectors holds the comma-split selector
// group. declarations holds the block's declarations (qualified rules and
// @font-face).
type cssRule struct {
	kind         cssRuleKind
	name         string
	prelude      string
	selectors    []string
	declarations []cssDeclaration
	rules        []*cssRule
}

// parseCSS parses a stylesheet into a flat list of top-level rules. Malformed
// input is tolerated: the parser stops at the first error and returns whatever
// was parsed so far (douceur returned an error which callers ignored, yielding
// an empty sheet — stopping early is at least as forgiving).
func parseCSS(text string) []*cssRule {
	rules, _ := parseCSSWithLimit(text, htmllimits.NoLimits().MaxStyleRules)
	return rules
}

func parseCSSWithLimit(text string, maxRules int) ([]*cssRule, error) {
	p := tcss.NewParser(parse.NewInput(bytes.NewBufferString(text)), false)
	state := cssParseState{maxRules: maxRules}

	for {
		gt, _, data := p.Next()
		if gt == tcss.ErrorGrammar {
			return state.top, nil
		}
		err := state.handleGrammar(gt, string(data), p.Values())
		if err != nil {
			return state.top, err
		}
	}
}

type cssParseState struct {
	top      []*cssRule
	curAt    *cssRule // open at-rule (@media / @font-face), or nil
	curSet   *cssRule // open ruleset, or nil
	count    int
	maxRules int
}

func (s *cssParseState) handleGrammar(gt tcss.GrammarType, data string, values []tcss.Token) error {
	switch gt {
	case tcss.ErrorGrammar:
	case tcss.BeginAtRuleGrammar:
		return s.beginAtRule(data, values)
	case tcss.EndAtRuleGrammar:
		s.endAtRule()
	case tcss.AtRuleGrammar:
		return s.atRule(data, values)
	case tcss.BeginRulesetGrammar:
		return s.beginRuleset(values)
	case tcss.EndRulesetGrammar:
		s.endRuleset()
	case tcss.DeclarationGrammar, tcss.CustomPropertyGrammar:
		return s.declaration(data, values)
	case tcss.CommentGrammar, tcss.QualifiedRuleGrammar, tcss.TokenGrammar:
		// Not meaningful here: comments are dropped, QualifiedRuleGrammar is
		// never emitted by this parser, and bare tokens carry no rule.
	}
	return nil
}

func (s *cssParseState) increment(kind string) error {
	s.count++
	if htmllimits.IntExceeded(s.maxRules, s.count) {
		return fmt.Errorf("%w: %s count %d exceeds limit %d", htmllimits.ErrStyleRulesTooLarge, kind, s.count, s.maxRules)
	}
	return nil
}

func (s *cssParseState) beginAtRule(data string, values []tcss.Token) error {
	err := s.increment("rule")
	if err != nil {
		return err
	}
	s.curAt = &cssRule{
		kind:    atRule,
		name:    data,
		prelude: concatTokens(values),
	}
	return nil
}

func (s *cssParseState) endAtRule() {
	if s.curAt != nil {
		s.top = append(s.top, s.curAt)
		s.curAt = nil
	}
}

func (s *cssParseState) atRule(data string, values []tcss.Token) error {
	err := s.increment("rule")
	if err != nil {
		return err
	}
	// At-rule terminated by ';' with no block (e.g. @import). Callers only
	// act on @media/@font-face, but keep it for fidelity.
	s.top = append(s.top, &cssRule{
		kind:    atRule,
		name:    data,
		prelude: concatTokens(values),
	})
	return nil
}

func (s *cssParseState) beginRuleset(values []tcss.Token) error {
	err := s.increment("rule")
	if err != nil {
		return err
	}
	s.curSet = &cssRule{
		kind:      qualifiedRule,
		selectors: splitSelectors(values),
	}
	return nil
}

func (s *cssParseState) endRuleset() {
	if s.curSet == nil {
		return
	}
	if s.curAt != nil {
		s.curAt.rules = append(s.curAt.rules, s.curSet)
	} else {
		s.top = append(s.top, s.curSet)
	}
	s.curSet = nil
}

func (s *cssParseState) declaration(data string, values []tcss.Token) error {
	target := s.declTarget()
	if target == nil {
		return nil
	}
	err := s.increment("declaration")
	if err != nil {
		return err
	}
	target.declarations = append(target.declarations, cssDeclaration{
		property: data,
		value:    declValue(values),
	})
	return nil
}

// declTarget returns where a declaration should be appended given current
// nesting: an open ruleset wins (it lives inside @media or at top level),
// otherwise an open at-rule absorbs it (@font-face descriptors).
func (s *cssParseState) declTarget() *cssRule {
	if s.curSet != nil {
		return s.curSet
	}
	return s.curAt
}

// concatTokens joins token data verbatim and trims surrounding whitespace.
func concatTokens(tokens []tcss.Token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.Write(t.Data)
	}
	return strings.TrimSpace(b.String())
}

// declValue reconstructs a declaration value and strips a trailing !important,
// matching douceur's Declaration.Value (which excluded it).
func declValue(tokens []tcss.Token) string {
	v := concatTokens(tokens)
	if i := strings.LastIndex(strings.ToLower(v), "!important"); i >= 0 &&
		strings.TrimSpace(v[i+len("!important"):]) == "" {
		v = strings.TrimSpace(v[:i])
	}
	return v
}

// splitSelectors reconstructs the selector group and splits it on top-level
// commas only — commas inside :not(...) or [...] are part of a single selector.
// Depth is tracked across '(' (any token ending in '(', e.g. "nth-child(") /
// ')' and '[' / ']'.
func splitSelectors(tokens []tcss.Token) []string {
	var sels []string
	var b strings.Builder
	depth := 0
	for _, t := range tokens {
		s := string(t.Data)
		switch {
		case s == "," && depth == 0:
			if sel := strings.TrimSpace(b.String()); sel != "" {
				sels = append(sels, sel)
			}
			b.Reset()
			continue
		case s == "[", strings.HasSuffix(s, "("):
			depth++
		case s == "]", s == ")":
			if depth > 0 {
				depth--
			}
		}
		b.WriteString(s)
	}
	if sel := strings.TrimSpace(b.String()); sel != "" {
		sels = append(sels, sel)
	}
	return sels
}
