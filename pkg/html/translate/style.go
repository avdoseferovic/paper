package translate

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
)

// computeNodeStyle resolves the ComputedStyle for a node by:
//  1. Inheriting font-size from the parent (for em resolution)
//  2. Applying matching rules from the provided <style> block stylesheet
//  3. Applying the node's inline style="" attribute (highest precedence within source)
func computeNodeStyle(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	s := css.NewComputedStyle()
	if parent != nil {
		s.FontSize = parent.FontSize
	}
	if sheet != nil && n.RawNode() != nil {
		sheet.applyToNode(n.RawNode(), s, parent)
	}
	inline := n.InlineStyle()
	if inline != "" {
		for prop, val := range parseInlineStyle(inline) {
			s.Apply(prop, val, parent)
		}
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
