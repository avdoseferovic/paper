package translate

import (
	"sort"

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
	matcher      cascadia.Sel
	declarations map[string]string
	order        int // source order (lower = earlier in stylesheet text)
}

// builtinCSS is a Maroto-shipped stylesheet prepended to every document.
// Its rules apply before user-supplied <style> blocks so users may override
// any built-in default. Inline style="" still has the highest precedence.
// Only tag-level defaults live here — opinionated presentational classes
// belong in the consumer's own stylesheet.
const builtinCSS = `
p { padding: 1mm 0 }
h1 { padding: 3mm 0 1mm 0 }
h2 { padding: 2mm 0 1mm 0 }
h3 { padding: 1mm 0 }
th { padding: 0.8mm 1mm }
ul, ol { margin-top: 2mm; margin-bottom: 1mm }
`

// parseStylesheet parses CSS text from <style> blocks into compiled rules.
// Invalid selectors are skipped silently. The built-in Maroto stylesheet is
// always prepended so its rules are applied first (and overridable by user CSS).
func parseStylesheet(text string) *stylesheet {
	ss := &stylesheet{}
	text = builtinCSS + "\n" + text
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
			m, err := cascadia.Parse(sel)
			if err != nil {
				continue
			}
			ss.rules = append(ss.rules, compiledRule{
				matcher:      m,
				declarations: decls,
				order:        len(ss.rules),
			})
		}
	}
	return ss
}

// applyToNode merges all matching stylesheet declarations into the ComputedStyle
// following CSS cascade rules: lower specificity first, equal specificity by source
// order (later wins). A class selector therefore wins over a tag selector even if
// the tag rule appears later in the stylesheet text.
func (s *stylesheet) applyToNode(n *html.Node, style *css.ComputedStyle, parent *css.ComputedStyle) {
	if s == nil || len(s.rules) == 0 {
		return
	}
	matching := make([]compiledRule, 0, len(s.rules))
	for _, rule := range s.rules {
		if rule.matcher.Match(n) {
			matching = append(matching, rule)
		}
	}
	sort.SliceStable(matching, func(i, j int) bool {
		si := matching[i].matcher.Specificity()
		sj := matching[j].matcher.Specificity()
		if si.Less(sj) {
			return true
		}
		if sj.Less(si) {
			return false
		}
		return matching[i].order < matching[j].order
	})
	for _, rule := range matching {
		for prop, val := range rule.declarations {
			style.Apply(prop, val, parent)
		}
	}
}
