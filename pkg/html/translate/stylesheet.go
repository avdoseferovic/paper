package translate

import (
	"github.com/andybalholm/cascadia"
	"github.com/aymerick/douceur/parser"
	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"golang.org/x/net/html"
)

// stylesheet holds parsed CSS rules from <style> blocks with pre-compiled selectors.
type stylesheet struct {
	rules []compiledRule
}

type compiledRule struct {
	matcher      cascadia.Matcher
	declarations map[string]string
}

// parseStylesheet parses CSS text from <style> blocks into compiled rules.
// Invalid selectors are skipped silently. Empty text returns an empty stylesheet.
func parseStylesheet(text string) *stylesheet {
	ss := &stylesheet{}
	if text == "" {
		return ss
	}
	sheet, err := parser.Parse(text)
	if err != nil || sheet == nil {
		return ss
	}
	for _, rule := range sheet.Rules {
		if rule == nil || rule.Kind != 0 { // 0 = QualifiedRule
			continue
		}
		decls := make(map[string]string, len(rule.Declarations))
		for _, d := range rule.Declarations {
			if d == nil || d.Property == "" {
				continue
			}
			decls[d.Property] = d.Value
		}
		decls = css.ExpandShorthands(decls)
		for _, sel := range rule.Selectors {
			m, err := cascadia.Compile(sel)
			if err != nil {
				continue
			}
			ss.rules = append(ss.rules, compiledRule{matcher: m, declarations: decls})
		}
	}
	return ss
}

// applyToNode merges all matching stylesheet declarations into the ComputedStyle.
// Rules are applied in source order (later rules override earlier ones with equal specificity).
func (s *stylesheet) applyToNode(n *html.Node, style *css.ComputedStyle, parent *css.ComputedStyle) {
	if s == nil || len(s.rules) == 0 {
		return
	}
	for _, rule := range s.rules {
		if !rule.matcher.Match(n) {
			continue
		}
		for prop, val := range rule.declarations {
			style.Apply(prop, val, parent)
		}
	}
}
