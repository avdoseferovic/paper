package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// computeNodeStyleRooted is computeNodeStyle with an explicit root style seed.
// When parent is nil and root is non-nil, root is used as the inheritance
// source so :root / html-level CSS variables propagate into body descendants.
func computeNodeStyleRooted(sheet *stylesheet, n *dom.Node, parent, root *css.ComputedStyle) *css.ComputedStyle {
	effectiveParent := parent
	if effectiveParent == nil {
		effectiveParent = root
	}
	return computeNodeStyle(sheet, n, effectiveParent)
}

// computeNodeStyle resolves the ComputedStyle for a node by:
//  1. Inheriting font-size and CSS custom properties from the parent
//  2. Applying matching rules from the provided <style> block stylesheet
//  3. Applying the node's inline style="" attribute (highest precedence within source)
//
// ctxWidth is the parent's content width in mm, used to resolve % and calc(%).
// When 0 (or when there is no parent width), percentages in length properties
// resolve to 0 (matching previous behaviour).
func computeNodeStyle(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	ctxWidth := 0.0
	if parent != nil && parent.Width > 0 {
		ctxWidth = parent.Width
	}
	return computeNodeStyleCtx(sheet, n, parent, ctxWidth)
}

// computeNodeStyleCtx is computeNodeStyle with an explicit context width.
// Callers that know the available content width (e.g. the top-level translator
// passing contentWidthMM for direct body children) should use this variant.
func computeNodeStyleCtx(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle, ctxWidth float64) *css.ComputedStyle {
	s := css.NewComputedStyle()
	if parent != nil {
		s.FontSize = parent.FontSize
		// Inherit CSS custom properties via shallow copy so children don't
		// pollute parent's map.
		if len(parent.Vars) > 0 {
			s.Vars = make(map[string]string, len(parent.Vars))
			for k, v := range parent.Vars {
				s.Vars[k] = v
			}
		}
	}
	if sheet != nil && n.RawNode() != nil {
		sheet.applyToNodeCtx(n.RawNode(), s, parent, ctxWidth)
	}
	inline := n.InlineStyle()
	if inline != "" {
		for prop, val := range parseInlineStyle(inline) {
			s.ApplyCtx(prop, val, parent, ctxWidth)
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
// Paper props.Cell. Returns nil if no decorative styling is set.
func (tr *translator) blockCellStyle(style *css.ComputedStyle) *props.Cell {
	cell := baseBlockCellStyle(style)
	if style == nil {
		return cell
	}
	bgImage := tr.backgroundImage(style)
	if bgImage == nil {
		return cell
	}
	if cell == nil {
		cell = &props.Cell{}
	}
	cell.BackgroundImage = bgImage
	return cell
}

func baseBlockCellStyle(style *css.ComputedStyle) *props.Cell {
	if style == nil {
		return nil
	}
	hasBorder := style.BorderTopWidth > 0 || style.BorderRightWidth > 0 ||
		style.BorderBottomWidth > 0 || style.BorderLeftWidth > 0
	hasRadius := style.BorderRadius > 0 || style.BorderRadiusTopLeft > 0 ||
		style.BorderRadiusTopRight > 0 || style.BorderRadiusBottomLeft > 0 ||
		style.BorderRadiusBottomRight > 0
	if style.BackgroundColor == nil && style.BackgroundGradient == nil &&
		len(style.BoxShadow) == 0 && style.OutlineWidth == 0 && !hasBorder && !hasRadius {
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
	cell.BorderTopStyle = cssBorderStyleToLineStyle(style.BorderTopStyle)
	cell.BorderRightStyle = cssBorderStyleToLineStyle(style.BorderRightStyle)
	cell.BorderBottomStyle = cssBorderStyleToLineStyle(style.BorderBottomStyle)
	cell.BorderLeftStyle = cssBorderStyleToLineStyle(style.BorderLeftStyle)
	cell.BorderRadius = style.BorderRadius
	cell.BorderRadiusTopLeft = style.BorderRadiusTopLeft
	cell.BorderRadiusTopRight = style.BorderRadiusTopRight
	cell.BorderRadiusBottomLeft = style.BorderRadiusBottomLeft
	cell.BorderRadiusBottomRight = style.BorderRadiusBottomRight
	if g := style.BackgroundGradient; g != nil {
		cell.BackgroundGradient = cssGradientToProps(g)
	}
	if len(style.BoxShadow) > 0 {
		cell.BoxShadow = cssShadowsToProps(style.BoxShadow)
	}
	if style.OutlineWidth > 0 {
		cell.OutlineWidth = style.OutlineWidth
		cell.OutlineStyle = cssBorderStyleToLineStyle(style.OutlineStyle)
		if style.OutlineColor != nil {
			cell.OutlineColor = &props.Color{Red: style.OutlineColor.R, Green: style.OutlineColor.G, Blue: style.OutlineColor.B}
		}
		cell.OutlineOffset = style.OutlineOffset
	}
	return cell
}

// cssShadowsToProps converts css.Shadow slice to props.Shadow slice.
func cssShadowsToProps(shadows []css.Shadow) []props.Shadow {
	out := make([]props.Shadow, len(shadows))
	for i, s := range shadows {
		out[i] = props.Shadow{
			OffsetX:    s.OffsetX,
			OffsetY:    s.OffsetY,
			BlurRadius: s.BlurRadius,
			Spread:     s.Spread,
			Inset:      s.Inset,
		}
		if s.Color != nil {
			out[i].Color = &props.Color{Red: s.Color.R, Green: s.Color.G, Blue: s.Color.B}
		}
	}
	return out
}

// cssGradientToProps converts a parsed css.Gradient to a props.Gradient.
func cssGradientToProps(g *css.Gradient) *props.Gradient {
	if g == nil {
		return nil
	}
	pg := &props.Gradient{}
	switch g.Kind {
	case css.GradientLinear:
		pg.Kind = props.GradientLinear
		if g.Linear != nil {
			pg.AngleDeg = g.Linear.AngleDeg
			pg.Stops = cssStopsToProps(g.Linear.Stops)
		}
	case css.GradientRadial:
		pg.Kind = props.GradientRadial
		if g.Radial != nil {
			pg.Circle = g.Radial.Circle
			pg.CX = g.Radial.CX
			pg.CY = g.Radial.CY
			pg.Stops = cssStopsToProps(g.Radial.Stops)
		}
	}
	return pg
}

func cssStopsToProps(stops []css.GradientStop) []props.GradientStop {
	out := make([]props.GradientStop, len(stops))
	for i, s := range stops {
		out[i] = props.GradientStop{
			Color:    props.Color{Red: s.Color.R, Green: s.Color.G, Blue: s.Color.B},
			Position: s.Position,
		}
	}
	return out
}

// applyInlineStyleToRuns applies CSS-computed font size and color to every run
// whose own field is unset (run-level styling wins over block-level).
// Also applies typography properties (letter-spacing, text-transform) to all runs.
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
		if style.LetterSpacing > 0 && runs[i].LetterSpacing == 0 {
			runs[i].LetterSpacing = style.LetterSpacing
		}
		if style.TextTransform != "" && style.TextTransform != "none" {
			runs[i].Text = css.ApplyTextTransform(runs[i].Text, style.TextTransform)
		}
		if style.TextShadow != nil && runs[i].TextShadow == nil {
			runs[i].TextShadow = cssShadowToProps(style.TextShadow)
		}
	}
}

// cssShadowToProps converts a single css.Shadow to a *props.Shadow.
func cssShadowToProps(s *css.Shadow) *props.Shadow {
	if s == nil {
		return nil
	}
	ps := &props.Shadow{
		OffsetX:    s.OffsetX,
		OffsetY:    s.OffsetY,
		BlurRadius: s.BlurRadius,
		Spread:     s.Spread,
		Inset:      s.Inset,
	}
	if s.Color != nil {
		ps.Color = &props.Color{Red: s.Color.R, Green: s.Color.G, Blue: s.Color.B}
	}
	return ps
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
	for _, part := range splitStyleDeclarations(decl) {
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

func splitStyleDeclarations(decl string) []string {
	var parts []string
	start := 0
	depth := 0
	var quote rune
	for i, r := range decl {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case r == ';' && depth == 0:
			parts = append(parts, decl[start:i])
			start = i + 1
		}
	}
	parts = append(parts, decl[start:])
	return parts
}

// cssBorderStyleToLineStyle maps a CSS border-style string to a linestyle.Type.
// Unmapped or empty values default to linestyle.Solid.
func cssBorderStyleToLineStyle(s string) linestyle.Type {
	switch s {
	case "dashed":
		return linestyle.Dashed
	case "dotted":
		return linestyle.Dotted
	default:
		return linestyle.Solid
	}
}
