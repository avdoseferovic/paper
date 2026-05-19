package translate

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
)

// computeNodeStyle parses the node's inline style="" attribute into a ComputedStyle.
// The parent ComputedStyle (if any) provides font-size context for em resolution.
//
// v1 limitation: <style> block selectors are NOT yet matched here. Only the inline
// style="" attribute is applied. Full cascade with <style> blocks is deferred.
func computeNodeStyle(n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	s := css.NewComputedStyle()
	if parent != nil {
		// Inherit font-size so em values resolve correctly.
		s.FontSize = parent.FontSize
	}
	inline := n.InlineStyle()
	if inline == "" {
		return s
	}
	for prop, val := range parseInlineStyle(inline) {
		s.Apply(prop, val, parent)
	}
	return s
}

// parseInlineStyle parses a CSS declaration block (e.g. "color:red; font-size:12pt")
// into a property→value map. Shorthands are expanded via css.ExpandShorthands.
func parseInlineStyle(decl string) map[string]string {
	raw := make(map[string]string)
	for part := range strings.SplitSeq(decl, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prop, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		prop = strings.TrimSpace(prop)
		val = strings.TrimSpace(val)
		if prop == "" || val == "" {
			continue
		}
		raw[prop] = val
	}
	return css.ExpandShorthands(raw)
}
