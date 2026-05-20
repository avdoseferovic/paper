package translate

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/props"
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

// toPropsColor converts an RGBColor into a props.Color, multiplying any explicit
// per-color alpha by the parent's CSS opacity. Alpha is left nil (== opaque) when
// the resulting effective alpha is >= 1.0 so existing render paths short-circuit.
func toPropsColor(c *css.RGBColor, opacity float64) *props.Color {
	if c == nil {
		return nil
	}
	out := &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	a := c.A * opacity
	if a < 1 {
		out.Alpha = &a
	}
	return out
}

// effectiveOpacity returns the cascade opacity multiplier; 1.0 when unset so
// callers can unconditionally multiply. NewComputedStyle initialises Opacity
// to 1.0, so an Opacity of 0 means the CSS opacity:0 was explicitly applied.
func effectiveOpacity(style *css.ComputedStyle) float64 {
	if style == nil {
		return 1.0
	}
	if style.Opacity < 0 {
		return 0
	}
	if style.Opacity > 1 {
		return 1
	}
	return style.Opacity
}

// blockCellStyle converts a ComputedStyle's background and border fields into a
// Maroto props.Cell. Returns nil if no decorative styling is set.
func blockCellStyle(style *css.ComputedStyle) *props.Cell {
	if style == nil {
		return nil
	}
	hasBorder := style.BorderTopWidth > 0 || style.BorderRightWidth > 0 ||
		style.BorderBottomWidth > 0 || style.BorderLeftWidth > 0
	hasRadius := style.BorderRadius > 0 || style.BorderRadiusTopLeft > 0 ||
		style.BorderRadiusTopRight > 0 || style.BorderRadiusBottomLeft > 0 ||
		style.BorderRadiusBottomRight > 0
	if style.BackgroundColor == nil && !hasBorder && !hasRadius {
		return nil
	}
	op := effectiveOpacity(style)
	cell := &props.Cell{}
	cell.BackgroundColor = toPropsColor(style.BackgroundColor, op)
	cell.BorderTopColor = toPropsColor(style.BorderTopColor, op)
	cell.BorderRightColor = toPropsColor(style.BorderRightColor, op)
	cell.BorderBottomColor = toPropsColor(style.BorderBottomColor, op)
	cell.BorderLeftColor = toPropsColor(style.BorderLeftColor, op)
	cell.BorderTopThickness = style.BorderTopWidth
	cell.BorderRightThickness = style.BorderRightWidth
	cell.BorderBottomThickness = style.BorderBottomWidth
	cell.BorderLeftThickness = style.BorderLeftWidth
	cell.BorderRadius = style.BorderRadius
	cell.BorderRadiusTopLeft = style.BorderRadiusTopLeft
	cell.BorderRadiusTopRight = style.BorderRadiusTopRight
	cell.BorderRadiusBottomLeft = style.BorderRadiusBottomLeft
	cell.BorderRadiusBottomRight = style.BorderRadiusBottomRight
	return cell
}

// applyInlineStyleToRuns applies CSS-computed font size and color to every run
// whose own field is unset (run-level styling wins over block-level).
func applyInlineStyleToRuns(style *css.ComputedStyle, runs []props.RichRun) {
	if style == nil {
		return
	}
	for i := range runs {
		if style.FontSize > 0 && runs[i].Size == 0 {
			// FontSize is in mm; props.RichRun expects pt — convert.
			runs[i].Size = style.FontSize / 0.352778
		}
		if style.Color != nil && runs[i].Color == nil {
			runs[i].Color = toPropsColor(style.Color, effectiveOpacity(style))
		}
	}
}

// isDisplayNone checks for the display:none inline-style override.
func isDisplayNone(n *dom.Node) bool {
	return strings.Contains(n.InlineStyle(), "display:none") ||
		strings.Contains(n.InlineStyle(), "display: none")
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
