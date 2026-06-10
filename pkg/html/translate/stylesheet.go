package translate

import (
	"sort"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"golang.org/x/net/html"
)

// stylesheet holds parsed CSS rules from <style> blocks with pre-compiled selectors.
// @font-face AtRules are collected separately on fontFaces. @media print/all
// rules are flattened into the normal rule stream; other AtRules are skipped.
type stylesheet struct {
	rules     []compiledRule
	pseudos   []compiledPseudoRule
	fontFaces []fontFaceRule

	// pageDecls accumulates declarations from plain `@page { ... }` rules in
	// source order (later rules override earlier ones at consumption time).
	// skippedPages records preludes of unsupported page rules (`:first`,
	// named pages) for unsupported-handler reporting.
	pageDecls    []cssDeclaration
	skippedPages []string
}

type compiledRule struct {
	matcher      cascadia.Sel
	declarations map[string]string
	order        int // source order (lower = earlier in stylesheet text)
}

type compiledPseudoRule struct {
	compiledRule
	pseudo string // "before" | "after"
}

// builtinCSS is a Paper-shipped stylesheet prepended to every document.
// Its rules apply before user-supplied <style> blocks so users may override
// any built-in default. Inline style="" still has the highest precedence.
// Only tag-level defaults live here — opinionated presentational classes
// belong in the consumer's own stylesheet.
const builtinCSS = `
[hidden] { display: none }
p { padding: 1mm 0 }
pre { padding: 1mm 0; white-space: pre; font-family: courier }
h1 { padding: 3mm 0 1mm 0 }
h2 { padding: 2mm 0 1mm 0 }
h3 { padding: 1mm 0 }
th { padding: 0.8mm 1mm }
ul, ol { margin-top: 2mm; margin-bottom: 1mm }
dt { padding-top: 1mm }
dd { padding-bottom: 1mm }
summary { padding: 1mm 0 }
caption { padding: 1mm 0; text-align: center }
`

// parseStylesheet parses CSS text from <style> blocks into compiled rules.
// Invalid selectors are skipped silently. The built-in Paper stylesheet is
// always prepended so its rules are applied first (and overridable by user CSS).
func parseStylesheet(text string) *stylesheet {
	return parseStylesheetWithContentWidth(text, defaultContentWidthMM)
}

func parseStylesheetWithContentWidth(text string, contentWidthMM float64) *stylesheet {
	ss, _ := parseStylesheetWithLimits(text, contentWidthMM, htmllimits.NoLimits())
	return ss
}

func parseStylesheetWithLimits(text string, contentWidthMM float64, limits htmllimits.Limits) (*stylesheet, error) {
	ss := &stylesheet{}
	order := 0
	for _, rule := range parseCSS(builtinCSS) {
		ss.addParsedRule(rule, &order, contentWidthMM)
	}
	rules, err := parseCSSWithLimit(text, limits.MaxStyleRules)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		ss.addParsedRule(rule, &order, contentWidthMM)
	}
	return ss, nil
}

func (s *stylesheet) addParsedRule(rule *cssRule, order *int, contentWidthMM float64) {
	if rule == nil {
		return
	}
	if rule.kind == atRule {
		switch rule.name {
		case "@font-face":
			if face, ok := extractFontFace(rule); ok {
				s.fontFaces = append(s.fontFaces, face)
			}
		case "@media":
			if mediaAppliesToPrintAtWidth(rule.prelude, contentWidthMM) {
				for _, nested := range rule.rules {
					s.addParsedRule(nested, order, contentWidthMM)
				}
			}
		case "@page":
			if prelude := strings.TrimSpace(rule.prelude); prelude != "" {
				// `:first`, `:left/:right`, and named pages are out of scope.
				s.skippedPages = append(s.skippedPages, "@page "+prelude)
				return
			}
			s.pageDecls = append(s.pageDecls, rule.declarations...)
		}
		return
	}
	if rule.kind != qualifiedRule {
		return
	}
	decls := make(map[string]string, len(rule.declarations))
	for _, d := range rule.declarations {
		if d.property == "" {
			continue
		}
		decls[d.property] = d.value
	}
	decls = css.ExpandShorthands(decls)
	for _, sel := range rule.selectors {
		baseSelector, pseudo := splitPseudoElementSelector(sel)
		m, err := cascadia.Parse(baseSelector)
		if err != nil {
			continue
		}
		compiled := compiledRule{
			matcher:      m,
			declarations: decls,
			order:        *order,
		}
		*order++
		if pseudo != "" {
			s.pseudos = append(s.pseudos, compiledPseudoRule{
				compiledRule: compiled,
				pseudo:       pseudo,
			})
			continue
		}
		s.rules = append(s.rules, compiled)
	}
}

func mediaAppliesToPrintAtWidth(prelude string, contentWidthMM float64) bool {
	for query := range strings.SplitSeq(prelude, ",") {
		if mediaQueryAppliesToPrintAtWidth(query, contentWidthMM) {
			return true
		}
	}
	return false
}

func mediaQueryAppliesToPrintAtWidth(query string, contentWidthMM float64) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || strings.HasPrefix(query, "not ") {
		return false
	}
	query = strings.TrimSpace(strings.TrimPrefix(query, "only "))
	conditions := query
	if !strings.HasPrefix(query, "(") {
		mediaType, mediaConditions := splitMediaType(query)
		if mediaType != "print" && mediaType != "all" {
			return false
		}
		conditions = mediaConditions
	}
	for _, condition := range mediaConditions(conditions) {
		ok, supported := mediaConditionMatches(condition, contentWidthMM)
		if !supported || !ok {
			return false
		}
	}
	return true
}

func splitMediaType(query string) (string, string) {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return "", ""
	}
	mediaType := fields[0]
	rest := strings.TrimSpace(query[len(mediaType):])
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "and"))
	return mediaType, rest
}

func mediaConditions(query string) []string {
	var conditions []string
	depth := 0
	start := -1
	for i, r := range query {
		switch r {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					conditions = append(conditions, strings.TrimSpace(query[start:i]))
					start = -1
				}
			}
		}
	}
	return conditions
}

func mediaConditionMatches(condition string, contentWidthMM float64) (bool, bool) {
	prop, value, ok := strings.Cut(condition, ":")
	if !ok {
		return false, false
	}
	widthMM := parseMediaQueryLength(value, contentWidthMM)
	if widthMM <= 0 {
		return false, false
	}
	contentWidthPx := contentWidthMM / 0.264583
	targetPx := widthMM / 0.264583
	switch strings.TrimSpace(prop) {
	case "min-width":
		return contentWidthPx >= targetPx, true
	case "max-width":
		return contentWidthPx <= targetPx, true
	case "width":
		return contentWidthPx == targetPx, true
	default:
		return false, false
	}
}

func parseMediaQueryLength(value string, contentWidthMM float64) float64 {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "vw") {
		v, err := strconv.ParseFloat(strings.TrimSpace(value[:len(value)-2]), 64)
		if err != nil {
			return 0
		}
		return contentWidthMM * v / 100
	}
	if strings.Contains(value, "%") || strings.Contains(value, "calc(") {
		return css.ParseLengthCtx(value, 0, contentWidthMM)
	}
	return css.ParseLength(value, 0)
}

func splitPseudoElementSelector(selector string) (string, string) {
	trimmed := strings.TrimSpace(selector)
	lower := strings.ToLower(trimmed)
	for _, suffix := range []struct {
		value  string
		pseudo string
	}{
		{value: "::before", pseudo: "before"},
		{value: ":before", pseudo: "before"},
		{value: "::after", pseudo: "after"},
		{value: ":after", pseudo: "after"},
	} {
		if strings.HasSuffix(lower, suffix.value) {
			base := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix.value)])
			if base == "" {
				base = "*"
			}
			return base, suffix.pseudo
		}
	}
	return trimmed, ""
}

// applyToNodeCtx merges all matching stylesheet declarations into the ComputedStyle
// following CSS cascade rules: lower specificity first, equal specificity by source
// order (later wins). A class selector therefore wins over a tag selector even if
// the tag rule appears later in the stylesheet text.
func (s *stylesheet) applyToNodeCtx(n *html.Node, style *css.ComputedStyle, parent *css.ComputedStyle, ctxWidth float64) {
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
			style.ApplyCtx(prop, val, parent, ctxWidth)
		}
	}
}

func (s *stylesheet) applyPseudoToNodeCtx(
	n *html.Node,
	style *css.ComputedStyle,
	parent *css.ComputedStyle,
	ctxWidth float64,
	pseudo string,
) {
	if s == nil || len(s.pseudos) == 0 {
		return
	}
	matching := make([]compiledPseudoRule, 0, len(s.pseudos))
	for _, rule := range s.pseudos {
		if rule.pseudo == pseudo && rule.matcher.Match(n) {
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
			style.ApplyCtx(prop, val, parent, ctxWidth)
		}
	}
}
