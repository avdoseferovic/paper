package translate

import (
	"slices"

	"github.com/avdoseferovic/paper/internal/cssparse"
	"github.com/avdoseferovic/paper/internal/htmllimits"
)

// cssRuleKind distinguishes qualified rules from at-rules. These local types
// keep the translator's existing surface stable while parsing itself lives in
// the cycle-free internal/cssparse package shared with SVG.
type cssRuleKind int

const (
	qualifiedRule cssRuleKind = iota
	atRule
)

type cssDeclaration struct {
	property  string
	value     string
	important bool
}

type cssRule struct {
	kind         cssRuleKind
	name         string
	prelude      string
	selectors    []string
	declarations []cssDeclaration
	rules        []*cssRule
}

// parseCSS tolerates malformed CSS by returning its completed prefix. The
// caller's historical behavior ignored parser errors, so this preserves the
// partial stylesheet rather than discarding usable preceding rules.
func parseCSS(text string) []*cssRule {
	rules, _ := parseCSSWithLimit(text, htmllimits.NoLimits().MaxStyleRules)
	return rules
}

func parseCSSWithLimit(text string, maxRules int) ([]*cssRule, error) {
	rules, err := cssparse.ParseWithMaxRules(text, maxRules)
	return adaptCSSRules(rules), err
}

func adaptCSSRules(rules []*cssparse.Rule) []*cssRule {
	adapted := make([]*cssRule, 0, len(rules))
	for _, rule := range rules {
		adapted = append(adapted, adaptCSSRule(rule))
	}
	return adapted
}

func adaptCSSRule(rule *cssparse.Rule) *cssRule {
	adapted := &cssRule{
		kind:      qualifiedRule,
		name:      rule.Name,
		prelude:   rule.Prelude,
		selectors: slices.Clone(rule.Selectors),
	}
	if rule.Kind == cssparse.AtRule {
		adapted.kind = atRule
	}
	adapted.declarations = make([]cssDeclaration, 0, len(rule.Declarations))
	for _, declaration := range rule.Declarations {
		adapted.declarations = append(adapted.declarations, cssDeclaration{
			property:  declaration.Property,
			value:     declaration.Value,
			important: declaration.Important,
		})
	}
	adapted.rules = adaptCSSRules(rule.Rules)
	return adapted
}

func stripImportantSuffix(value string) (string, bool) {
	return cssparse.StripImportantSuffix(value)
}
