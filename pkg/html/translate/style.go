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
	cell := &props.Cell{}
	if style.BackgroundColor != nil {
		cell.BackgroundColor = &props.Color{
			Red: style.BackgroundColor.R, Green: style.BackgroundColor.G, Blue: style.BackgroundColor.B,
		}
	}
	if c := style.BorderTopColor; c != nil {
		cell.BorderTopColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderRightColor; c != nil {
		cell.BorderRightColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderBottomColor; c != nil {
		cell.BorderBottomColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
	if c := style.BorderLeftColor; c != nil {
		cell.BorderLeftColor = &props.Color{Red: c.R, Green: c.G, Blue: c.B}
	}
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
			runs[i].Color = &props.Color{Red: style.Color.R, Green: style.Color.G, Blue: style.Color.B}
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
