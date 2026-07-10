package translate

import (
	"maps"
	"strconv"
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// computeNodeStyleRooted is computeNodeStyle with an explicit root style seed.
// When parent is nil and root is non-nil, root is used as the inheritance
// source so :root / html-level CSS variables propagate into body descendants.
func computeNodeStyleRooted(sheet *stylesheet, n *dom.Node, root *css.ComputedStyle) *css.ComputedStyle {
	return computeNodeStyle(sheet, n, root)
}

func (tr *translator) computeBlockStyle(n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	effectiveParent := parent
	if effectiveParent == nil {
		effectiveParent = tr.rootStyle
	}
	ctxWidth := tr.availableContentWidth()
	if effectiveParent != nil && effectiveParent.Width > 0 {
		ctxWidth = effectiveParent.Width
	}
	return computeNodeStyleCtx(tr.sheet, n, effectiveParent, ctxWidth)
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
		s.TextAlign = parent.TextAlign
		s.LetterSpacing = parent.LetterSpacing
		s.Visibility = parent.Visibility
		// Inherit CSS custom properties via shallow copy so children don't
		// pollute parent's map.
		if len(parent.Vars) > 0 {
			s.Vars = make(map[string]string, len(parent.Vars))
			maps.Copy(s.Vars, parent.Vars)
		}
	}
	applyCascade(sheet, n, s, parent, ctxWidth)
	return s
}

func computeInlineNodeStyle(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	s := inheritInlineStyle(parent)
	applyCascade(sheet, n, s, parent, 0)
	return s
}

// applyCascade layers stylesheet and inline declarations in CSS cascade order:
// stylesheet normal → inline normal → stylesheet !important → inline !important.
func applyCascade(sheet *stylesheet, n *dom.Node, s, parent *css.ComputedStyle, ctxWidth float64) {
	if sheet != nil && n.RawNode() != nil {
		sheet.applyToNodeCtx(n.RawNode(), s, parent, ctxWidth, false)
	}
	normal, important := parseInlineStyleImportance(n.InlineStyle())
	for prop, val := range normal {
		s.ApplyCtx(prop, val, parent, ctxWidth)
	}
	if sheet != nil && n.RawNode() != nil {
		sheet.applyToNodeCtx(n.RawNode(), s, parent, ctxWidth, true)
	}
	for prop, val := range important {
		s.ApplyCtx(prop, val, parent, ctxWidth)
	}
}

func computePseudoNodeStyle(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle, pseudo string) *css.ComputedStyle {
	s := inheritInlineStyle(parent)
	if sheet != nil && n.RawNode() != nil {
		sheet.applyPseudoToNodeCtx(n.RawNode(), s, parent, 0, pseudo)
	}
	return s
}

func inheritInlineStyle(parent *css.ComputedStyle) *css.ComputedStyle {
	s := css.NewComputedStyle()
	if parent == nil {
		return s
	}
	s.FontFamily = parent.FontFamily
	s.FontSize = parent.FontSize
	s.FontWeight = parent.FontWeight
	s.FontStyle = parent.FontStyle
	s.Color = cloneCSSColor(parent.Color)
	s.TextDecoration = parent.TextDecoration
	s.LineHeight = parent.LineHeight
	s.BackgroundColor = cloneCSSColor(parent.BackgroundColor)
	s.TextShadow = cloneCSSShadow(parent.TextShadow)
	s.TextShadows = cloneCSSShadows(parent.TextShadows)
	s.Visibility = parent.Visibility
	s.Opacity = parent.Opacity
	s.LetterSpacing = parent.LetterSpacing
	s.TextTransform = parent.TextTransform
	// white-space is an inherited CSS property: a child run (e.g. <strong>) or a
	// generated-content marker (::before) inside a `white-space: nowrap` pill
	// must stay nowrap too, or the pill wraps mid-content and its background is
	// painted as several fragments with a stranded marker on the prior line.
	s.WhiteSpace = parent.WhiteSpace
	s.VerticalAlign = parent.VerticalAlign
	s.VerticalOffset = parent.VerticalOffset
	s.Quotes = parent.Quotes
	if len(parent.Vars) > 0 {
		s.Vars = make(map[string]string, len(parent.Vars))
		maps.Copy(s.Vars, parent.Vars)
	}
	return s
}

func blockInlineStyle(style *css.ComputedStyle) *css.ComputedStyle {
	s := inheritInlineStyle(style)
	s.BackgroundColor = nil
	s.VerticalAlign = ""
	s.VerticalOffset = 0
	return s
}

func cloneCSSColor(c *css.RGBColor) *css.RGBColor {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

func cloneCSSShadow(s *css.Shadow) *css.Shadow {
	if s == nil {
		return nil
	}
	clone := *s
	clone.Color = cloneCSSColor(s.Color)
	return &clone
}

func cloneCSSShadows(shadows []css.Shadow) []css.Shadow {
	if len(shadows) == 0 {
		return nil
	}
	out := make([]css.Shadow, len(shadows))
	for i, shadow := range shadows {
		out[i] = shadow
		out[i].Color = cloneCSSColor(shadow.Color)
	}
	return out
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
	if isVisibilityHidden(style) {
		return nil
	}
	hasBorder := visibleBorderSide(style.BorderTopWidth, style.BorderTopStyle) ||
		visibleBorderSide(style.BorderRightWidth, style.BorderRightStyle) ||
		visibleBorderSide(style.BorderBottomWidth, style.BorderBottomStyle) ||
		visibleBorderSide(style.BorderLeftWidth, style.BorderLeftStyle)
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
	if visibleBorderSide(style.BorderTopWidth, style.BorderTopStyle) {
		cell.BorderTopColor = toPropsColor(style.BorderTopColor, op)
		cell.BorderTopThickness = style.BorderTopWidth
	}
	if visibleBorderSide(style.BorderRightWidth, style.BorderRightStyle) {
		cell.BorderRightColor = toPropsColor(style.BorderRightColor, op)
		cell.BorderRightThickness = style.BorderRightWidth
	}
	if visibleBorderSide(style.BorderBottomWidth, style.BorderBottomStyle) {
		cell.BorderBottomColor = toPropsColor(style.BorderBottomColor, op)
		cell.BorderBottomThickness = style.BorderBottomWidth
	}
	if visibleBorderSide(style.BorderLeftWidth, style.BorderLeftStyle) {
		cell.BorderLeftColor = toPropsColor(style.BorderLeftColor, op)
		cell.BorderLeftThickness = style.BorderLeftWidth
	}
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
			out[i].Color = toPropsColor(s.Color, 1)
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
	case css.GradientConic:
		pg.Kind = props.GradientConic
		if g.Conic != nil {
			pg.AngleDeg = g.Conic.FromDeg
			pg.CX = g.Conic.CX
			pg.CY = g.Conic.CY
			pg.Stops = cssStopsToProps(g.Conic.Stops)
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

// effectiveUniformRadius returns the run-background corner radius for an inline
// element. CSS `border-radius: 9999px` is expanded into four per-corner
// longhands at parse time, leaving the uniform BorderRadius at 0, so fall back
// to the largest per-corner value (pills set all four equal). The render layer
// caps the radius to half the line height, so an over-large value is harmless.
func effectiveUniformRadius(style *css.ComputedStyle) float64 {
	if style.BorderRadius > 0 {
		return style.BorderRadius
	}
	r := style.BorderRadiusTopLeft
	for _, c := range []float64{style.BorderRadiusTopRight, style.BorderRadiusBottomRight, style.BorderRadiusBottomLeft} {
		if c > r {
			r = c
		}
	}
	return r
}

func inlineBorderWidth(style *css.ComputedStyle) float64 {
	if style == nil {
		return 0
	}
	width := 0.0
	if isInlineBorderSide(style.BorderTopWidth, style.BorderTopStyle) {
		width = max(width, style.BorderTopWidth)
	}
	if isInlineBorderSide(style.BorderRightWidth, style.BorderRightStyle) {
		width = max(width, style.BorderRightWidth)
	}
	if isInlineBorderSide(style.BorderBottomWidth, style.BorderBottomStyle) {
		width = max(width, style.BorderBottomWidth)
	}
	if isInlineBorderSide(style.BorderLeftWidth, style.BorderLeftStyle) {
		width = max(width, style.BorderLeftWidth)
	}
	return width
}

func visibleBorderSide(width float64, style string) bool {
	return isInlineBorderSide(width, style)
}

func visibleBorderWidth(width float64, style string) float64 {
	if !visibleBorderSide(width, style) {
		return 0
	}
	return width
}

func boxSizingBorderBox(style *css.ComputedStyle) bool {
	return style != nil && strings.EqualFold(strings.TrimSpace(style.BoxSizing), "border-box")
}

func cssContentBoxWidth(style *css.ComputedStyle) float64 {
	if style == nil || style.Width <= 0 {
		return 0
	}
	if !boxSizingBorderBox(style) {
		return style.Width
	}
	return max(0,
		style.Width-
			style.PaddingLeft-
			style.PaddingRight-
			visibleBorderWidth(style.BorderLeftWidth, style.BorderLeftStyle)-
			visibleBorderWidth(style.BorderRightWidth, style.BorderRightStyle),
	)
}

func cssContentBoxHeight(style *css.ComputedStyle) float64 {
	if style == nil || style.Height <= 0 {
		return 0
	}
	if !boxSizingBorderBox(style) {
		return style.Height
	}
	return max(0,
		style.Height-
			style.PaddingTop-
			style.PaddingBottom-
			visibleBorderWidth(style.BorderTopWidth, style.BorderTopStyle)-
			visibleBorderWidth(style.BorderBottomWidth, style.BorderBottomStyle),
	)
}

func isInlineBorderSide(width float64, style string) bool {
	if width <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(style)) {
	case cssValueNone, cssValueHidden:
		return false
	default:
		return true
	}
}

func inlineBorderColor(style *css.ComputedStyle) *css.RGBColor {
	if style == nil {
		return nil
	}
	for _, color := range []*css.RGBColor{
		style.BorderTopColor,
		style.BorderRightColor,
		style.BorderBottomColor,
		style.BorderLeftColor,
	} {
		if color != nil {
			return color
		}
	}
	return nil
}

type inlineBoxStyle struct {
	ID          int
	Background  *props.Color
	BorderColor *props.Color
	BorderWidth float64
	Radius      float64
	PadLeft     float64
	PadRight    float64
	PadY        float64
	MarginLeft  float64
	MarginRight float64
	BoxShadows  []props.Shadow
}

func inlineBoxFromStyle(style, parent *css.ComputedStyle) (inlineBoxStyle, bool) {
	if style == nil {
		return inlineBoxStyle{}, false
	}
	op := effectiveOpacity(style)
	background := style.BackgroundColor
	if sameCSSColor(background, inheritedBackground(parent)) {
		background = nil
	}
	box := inlineBoxStyle{
		Background:  toPropsColor(background, op),
		BorderWidth: inlineBorderWidth(style),
		Radius:      effectiveUniformRadius(style),
		PadLeft:     style.PaddingLeft,
		PadRight:    style.PaddingRight,
		PadY:        max(style.PaddingTop, style.PaddingBottom),
		MarginLeft:  style.MarginLeft,
		MarginRight: style.MarginRight,
		BoxShadows:  cssShadowsToProps(style.BoxShadow),
	}
	if c := inlineBorderColor(style); c != nil {
		box.BorderColor = toPropsColor(c, op)
	}
	if box.Background == nil && (box.BorderColor == nil || box.BorderWidth <= 0) && len(box.BoxShadows) == 0 {
		return inlineBoxStyle{}, false
	}
	return box, true
}

func inheritedBackground(parent *css.ComputedStyle) *css.RGBColor {
	if parent == nil {
		return nil
	}
	return parent.BackgroundColor
}

func sameCSSColor(a, b *css.RGBColor) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

func applyInlineStyleToRun(style *css.ComputedStyle, run *props.RichRun) {
	if style == nil || run == nil {
		return
	}
	applyInlineVisibilityStyle(style, run)
	applyInlineFontStyle(style, run)
	applyInlineColorStyle(style, run)
	applyInlineBackgroundStyle(style, run)
	applyInlineBorderStyle(style, run)
	applyInlineEmptyBoxStyle(style, run)
	applyInlineSpacingStyle(style, run)
	applyInlineTextStyle(style, run)
}

func applyInlineVisibilityStyle(style *css.ComputedStyle, run *props.RichRun) {
	if isVisibilityHidden(style) {
		run.Hidden = true
		run.Hyperlink = nil
		run.LocalAnchor = ""
	}
}

func applyInlineFontStyle(style *css.ComputedStyle, run *props.RichRun) {
	if family := firstFontFamily(style.FontFamily); family != "" && run.Family == "" {
		run.Family = family
	}
	if shouldApplyCSSFontStyle(style) {
		run.Style = mergeCSSFontStyle(run.Style, style.FontWeight, style.FontStyle)
	}
	if style.FontSize > 0 && run.Size == 0 {
		// FontSize is in mm; props.RichRun expects pt — convert.
		run.Size = style.FontSize / 0.352778
	}
	if style.LineHeight > 0 && style.LineHeight != 1 && run.LineHeight == 0 {
		run.LineHeight = style.LineHeight
	}
	if ws := strings.ToLower(strings.TrimSpace(style.WhiteSpace)); ws != "" && ws != cssValueNormal && run.WhiteSpace == "" {
		run.WhiteSpace = ws
	}
}

func shouldApplyCSSFontStyle(style *css.ComputedStyle) bool {
	return isBoldCSSFontWeight(style.FontWeight) ||
		isSemiboldCSSFontWeight(style.FontWeight) ||
		isItalicCSSFontStyle(style.FontStyle)
}

func applyInlineColorStyle(style *css.ComputedStyle, run *props.RichRun) {
	if style.Color != nil && run.Color == nil {
		run.Color = toPropsColor(style.Color, effectiveOpacity(style))
	}
}

func applyInlineBackgroundStyle(style *css.ComputedStyle, run *props.RichRun) {
	if style.BackgroundColor != nil && run.Background == nil {
		run.Background = toPropsColor(style.BackgroundColor, effectiveOpacity(style))
		applyInlineBackgroundBoxStyle(style, run)
	}
}

func applyInlineBackgroundBoxStyle(style *css.ComputedStyle, run *props.RichRun) {
	// A backgrounded inline element with a corner radius and/or horizontal
	// padding renders as a rounded "pill"/badge (e.g. category chips). Plain
	// backgrounds (<mark>, <code>) leave these at 0 and render as before.
	if run.BgRadius == 0 {
		run.BgRadius = effectiveUniformRadius(style)
	}
	if run.BgPadX == 0 && style.PaddingLeft == style.PaddingRight {
		run.BgPadX = style.PaddingLeft
	}
	if run.BgPadLeft == 0 {
		run.BgPadLeft = style.PaddingLeft
	}
	if run.BgPadRight == 0 {
		run.BgPadRight = style.PaddingRight
	}
	if run.BgPadY == 0 {
		run.BgPadY = max(style.PaddingTop, style.PaddingBottom)
	}
}

func applyInlineBorderStyle(style *css.ComputedStyle, run *props.RichRun) {
	if run.BorderWidth == 0 {
		run.BorderWidth = inlineBorderWidth(style)
	}
	if run.BorderColor == nil {
		applyInlineBorderColor(style, run)
	}
	if len(style.BoxShadow) > 0 && len(run.BoxShadows) == 0 {
		run.BoxShadows = cssShadowsToProps(style.BoxShadow)
	}
}

func applyInlineBorderColor(style *css.ComputedStyle, run *props.RichRun) {
	if c := inlineBorderColor(style); c != nil {
		run.BorderColor = toPropsColor(c, effectiveOpacity(style))
	}
}

func applyInlineEmptyBoxStyle(style *css.ComputedStyle, run *props.RichRun) {
	if run.Text != "" || run.Image != nil {
		return
	}
	if style.Width > 0 && run.InlineBoxWidth == 0 {
		run.InlineBoxWidth = cssContentBoxWidth(style)
	}
	if style.Height > 0 && run.InlineBoxHeight == 0 {
		run.InlineBoxHeight = cssContentBoxHeight(style)
	}
}

func applyInlineSpacingStyle(style *css.ComputedStyle, run *props.RichRun) {
	if style.MarginRight > 0 && run.InlineMarginRight == 0 {
		run.InlineMarginRight = style.MarginRight
	}
	if style.MarginLeft > 0 && run.InlineMarginLeft == 0 {
		run.InlineMarginLeft = style.MarginLeft
	}
	if style.LetterSpacing > 0 && run.LetterSpacing == 0 {
		run.LetterSpacing = style.LetterSpacing
	}
}

func applyInlineTextStyle(style *css.ComputedStyle, run *props.RichRun) {
	applyTextDecoration(style.TextDecoration, run)
	if align := richRunVerticalAlignFromCSS(style.VerticalAlign); align != "" {
		run.VerticalAlign = align
	}
	if style.VerticalOffset != 0 {
		run.VerticalOffset = style.VerticalOffset
	}
	if style.TextTransform != "" && style.TextTransform != cssValueNone {
		run.Text = css.ApplyTextTransform(run.Text, style.TextTransform)
	}
	applyInlineTextShadowStyle(style, run)
}

func applyInlineTextShadowStyle(style *css.ComputedStyle, run *props.RichRun) {
	if len(style.TextShadows) > 0 && len(run.TextShadows) == 0 && run.TextShadow == nil {
		run.TextShadows = cssShadowsToProps(style.TextShadows)
		if len(run.TextShadows) > 0 {
			run.TextShadow = &run.TextShadows[0]
		}
	} else if style.TextShadow != nil && run.TextShadow == nil {
		run.TextShadow = cssShadowToProps(style.TextShadow)
	}
}

func isVisibilityHidden(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(style.Visibility)) {
	case cssValueHidden, "collapse":
		return true
	default:
		return false
	}
}

func firstFontFamily(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	first, _, _ := strings.Cut(value, ",")
	return strings.Trim(strings.TrimSpace(first), `'"`)
}

func isItalicCSSFontStyle(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "italic", "oblique":
		return true
	default:
		return false
	}
}

func isBoldCSSFontWeight(value string) bool {
	return strings.ToLower(strings.TrimSpace(value)) == "bold"
}

func isSemiboldCSSFontWeight(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "600", "semibold":
		return true
	default:
		return false
	}
}

func mergeCSSFontStyle(existing fontstyle.Type, weight, style string) fontstyle.Type {
	bold := strings.Contains(string(existing), string(fontstyle.Bold)) || isBoldCSSFontWeight(weight)
	semibold := strings.Contains(string(existing), string(fontstyle.Semibold)) || isSemiboldCSSFontWeight(weight)
	italic := strings.Contains(string(existing), string(fontstyle.Italic)) || isItalicCSSFontStyle(style)
	switch {
	case bold && italic:
		return fontstyle.BoldItalic
	case bold:
		return fontstyle.Bold
	case semibold && italic:
		return fontstyle.SemiboldItalic
	case semibold:
		return fontstyle.Semibold
	case italic:
		return fontstyle.Italic
	default:
		return existing
	}
}

func applyTextDecoration(value string, run *props.RichRun) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == cssValueNone {
		return
	}
	if strings.Contains(value, "underline") {
		run.Underline = true
	}
	if strings.Contains(value, "line-through") {
		run.Strikethrough = true
	}
}

func richRunVerticalAlignFromCSS(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case verticalAlignBaseline:
		return verticalAlignBaseline
	case verticalAlignSub:
		return verticalAlignSub
	case verticalAlignSuper, "sup":
		return verticalAlignSuper
	default:
		return ""
	}
}

func generatedContentRuns(value string, n *dom.Node, ctx runContext) ([]props.RichRun, bool) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", cssValueNormal, cssValueNone:
		return nil, false
	}

	parser := &generatedContentParser{node: n, ctx: ctx}
	if !parser.consume(value) {
		return nil, false
	}
	parser.flushText()
	return parser.runs, parser.consumed
}

type generatedContentParser struct {
	node     *dom.Node
	ctx      runContext
	text     strings.Builder
	runs     []props.RichRun
	consumed bool
}

func (p *generatedContentParser) consume(value string) bool {
	for value != "" {
		value = strings.TrimLeft(value, " \t\r\n\f")
		if value == "" {
			break
		}
		rest, ok := p.consumeNext(value)
		if !ok {
			return false
		}
		value = rest
		p.consumed = true
	}
	return true
}

func (p *generatedContentParser) flushText() {
	if p.text.Len() == 0 {
		return
	}
	p.runs = append(p.runs, richRunFromContext(p.text.String(), p.ctx))
	p.text.Reset()
}

func (p *generatedContentParser) consumeNext(value string) (string, bool) {
	if rest, matched, ok := p.consumeFunction(value); matched {
		return rest, ok
	}
	if rest, matched := p.consumeQuoteKeyword(value); matched {
		return rest, true
	}
	if value[0] == '"' || value[0] == '\'' {
		literal, rest, ok := readCSSContentString(value)
		if !ok {
			return "", false
		}
		p.text.WriteString(literal)
		return rest, true
	}
	return "", false
}

func (p *generatedContentParser) consumeFunction(value string) (string, bool, bool) {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "attr(") {
		return p.consumeAttrFunction(value)
	}
	if strings.HasPrefix(lower, "counter(") {
		return p.consumeCounterFunction(value)
	}
	if strings.HasPrefix(lower, "counters(") {
		return p.consumeCountersFunction(value)
	}
	if strings.HasPrefix(lower, "url(") {
		rest, ok := appendGeneratedImageRun(value, p.ctx, p.flushText, &p.runs)
		return rest, true, ok
	}
	return "", false, true
}

func (p *generatedContentParser) consumeAttrFunction(value string) (string, bool, bool) {
	args, rest, ok := readCSSFunction(value, "attr")
	if !ok {
		return "", true, false
	}
	name := strings.Trim(strings.TrimSpace(args), `'"`)
	if name != "" && p.node != nil {
		p.text.WriteString(p.node.Attr(name))
	}
	return rest, true, true
}

func (p *generatedContentParser) consumeCounterFunction(value string) (string, bool, bool) {
	args, rest, ok := readCSSFunction(value, "counter")
	if !ok {
		return "", true, false
	}
	content, ok := generatedCounterText(args, p.ctx.counters)
	if !ok {
		return "", true, false
	}
	p.text.WriteString(content)
	return rest, true, true
}

func (p *generatedContentParser) consumeCountersFunction(value string) (string, bool, bool) {
	args, rest, ok := readCSSFunction(value, "counters")
	if !ok {
		return "", true, false
	}
	content, ok := generatedCountersText(args, p.ctx.counters)
	if !ok {
		return "", true, false
	}
	p.text.WriteString(content)
	return rest, true, true
}

func (p *generatedContentParser) consumeQuoteKeyword(value string) (string, bool) {
	if rest, ok := consumeContentKeyword(value, "open-quote"); ok {
		p.text.WriteString(p.ctx.quotes.open(p.ctx.style))
		return rest, true
	}
	if rest, ok := consumeContentKeyword(value, "close-quote"); ok {
		p.text.WriteString(p.ctx.quotes.close(p.ctx.style))
		return rest, true
	}
	if rest, ok := consumeContentKeyword(value, "no-open-quote"); ok {
		p.ctx.quotes.noOpen()
		return rest, true
	}
	if rest, ok := consumeContentKeyword(value, "no-close-quote"); ok {
		p.ctx.quotes.noClose()
		return rest, true
	}
	return "", false
}

func appendGeneratedImageRun(
	value string,
	ctx runContext,
	flushText func(),
	runs *[]props.RichRun,
) (string, bool) {
	args, rest, ok := readCSSFunction(value, "url")
	if !ok {
		return "", false
	}
	src, ok := cssStringArg(args)
	if !ok || src == "" {
		return "", false
	}
	if ctx.contentImage == nil {
		return rest, true
	}
	img, ok := ctx.contentImage(src, ctx.style)
	if !ok {
		return rest, true
	}
	flushText()
	run := richRunFromContext("", ctx)
	run.Image = img
	run.InlineBoxWidth = 0
	run.InlineBoxHeight = 0
	*runs = append(*runs, run)
	return rest, true
}

func consumeContentKeyword(value, keyword string) (string, bool) {
	if len(value) < len(keyword) || !strings.EqualFold(value[:len(keyword)], keyword) {
		return "", false
	}
	if len(value) == len(keyword) {
		return "", true
	}
	next := value[len(keyword)]
	if next == ' ' || next == '\t' || next == '\r' || next == '\n' || next == '\f' {
		return value[len(keyword):], true
	}
	return "", false
}

func readCSSFunction(value, name string) (string, string, bool) {
	prefix := strings.ToLower(name) + "("
	if !strings.HasPrefix(strings.ToLower(value), prefix) {
		return "", "", false
	}
	depth := 1
	var quote byte
	escaped := false
	start := len(prefix)
	for i := start; i < len(value); i++ {
		ch := value[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return value[start:i], value[i+1:], true
			}
		}
	}
	return "", "", false
}

func generatedCounterText(args string, counters *counterState) (string, bool) {
	parts := splitCSSFunctionArgs(args)
	if len(parts) == 0 {
		return "", false
	}
	name := cssIdentifierArg(parts[0])
	if name == "" {
		return "", false
	}
	style := listStyleDecimal
	if len(parts) > 1 {
		style = cssIdentifierArg(parts[1])
	}
	return formatCounterValue(counters.value(name), style), true
}

func generatedCountersText(args string, counters *counterState) (string, bool) {
	parts := splitCSSFunctionArgs(args)
	if len(parts) < 2 {
		return "", false
	}
	name := cssIdentifierArg(parts[0])
	separator, ok := cssStringArg(parts[1])
	if name == "" || !ok {
		return "", false
	}
	style := listStyleDecimal
	if len(parts) > 2 {
		style = cssIdentifierArg(parts[2])
	}
	values := counters.allValues(name)
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = formatCounterValue(value, style)
	}
	return strings.Join(out, separator), true
}

func splitCSSFunctionArgs(args string) []string {
	var parts []string
	start := 0
	depth := 0
	var quote rune
	escaped := false
	for i, r := range args {
		switch {
		case quote != 0:
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
		default:
			switch r {
			case '"', '\'':
				quote = r
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			case ',':
				if depth == 0 {
					parts = append(parts, strings.TrimSpace(args[start:i]))
					start = i + 1
				}
			}
		}
	}
	parts = append(parts, strings.TrimSpace(args[start:]))
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func cssIdentifierArg(value string) string {
	return strings.Trim(strings.TrimSpace(value), `'"`)
}

func cssStringArg(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if value[0] == '"' || value[0] == '\'' {
		text, rest, ok := readCSSContentString(value)
		return text, ok && strings.TrimSpace(rest) == ""
	}
	return value, true
}

func readCSSContentString(value string) (string, string, bool) {
	if value == "" {
		return "", "", false
	}
	quote := value[0]
	var out strings.Builder
	for i := 1; i < len(value); i++ {
		ch := value[i]
		switch ch {
		case '\\':
			text, next := decodeCSSEscape(value, i+1)
			out.WriteString(text)
			i = next - 1
		case quote:
			return out.String(), value[i+1:], true
		default:
			out.WriteByte(ch)
		}
	}
	return "", "", false
}

// decodeCSSEscape decodes the escape sequence starting after a backslash at
// value[start] per CSS Syntax §4.3.7: one to six hex digits followed by an
// optional whitespace terminator, or a single literal character. It returns
// the decoded text and the index of the first unconsumed byte.
func decodeCSSEscape(value string, start int) (string, int) {
	if start >= len(value) {
		return "", start
	}
	end := start
	for end < len(value) && end-start < 6 && isHexDigit(value[end]) {
		end++
	}
	if end == start {
		return string(value[start]), start + 1
	}
	code, err := strconv.ParseUint(value[start:end], 16, 32)
	if err != nil || code == 0 || code > 0x10FFFF {
		return string(value[start]), start + 1
	}
	// A single whitespace after the hex digits terminates the escape.
	if end < len(value) && (value[end] == ' ' || value[end] == '\t' || value[end] == '\n') {
		end++
	}
	return string(rune(code)), end
}

func isHexDigit(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		return true
	default:
		return false
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
		ps.Color = toPropsColor(s.Color, 1)
	}
	return ps
}

// isDisplayNone checks for the display:none inline-style override.
func isDisplayNone(n *dom.Node) bool {
	if n != nil && n.Attr("hidden") != "" {
		return true
	}
	return n != nil &&
		(strings.Contains(n.InlineStyle(), "display:none") ||
			strings.Contains(n.InlineStyle(), "display: none"))
}

// parseInlineStyle parses a CSS declaration block (e.g. "color:red; font-size:12pt")
// into a property→value map. Shorthands are expanded via css.ExpandShorthands.
// !important suffixes are stripped; importance is ignored (see
// parseInlineStyleImportance for cascade-aware parsing).
func parseInlineStyle(decl string) map[string]string {
	normal, important := parseInlineStyleImportance(decl)
	maps.Copy(normal, important)
	return normal
}

// parseInlineStyleImportance parses a CSS declaration block into two
// property→value maps: normal declarations and !important ones. Shorthands
// are expanded via css.ExpandShorthands.
func parseInlineStyleImportance(decl string) (map[string]string, map[string]string) {
	rawNormal := make(map[string]string)
	rawImportant := make(map[string]string)
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
		val, important := stripImportantSuffix(strings.TrimSpace(val))
		if prop == "" || val == "" {
			continue
		}
		// Expand each declaration individually, in source order, so a longhand
		// after a shorthand ("margin: 0; margin-top: 10px") deterministically
		// overrides it. Expanding the whole map at once would iterate in
		// random map order.
		target := rawNormal
		if important {
			target = rawImportant
		}
		maps.Copy(target, css.ExpandShorthands(map[string]string{prop: val}))
	}
	return rawNormal, rawImportant
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

// cssBorderStyleToLineStyle maps a CSS border-style string to a consts.LineStyle.
// Unmapped or empty values default to consts.LineStyleSolid.
func cssBorderStyleToLineStyle(s string) consts.LineStyle {
	switch s {
	case "dashed":
		return consts.LineStyleDashed
	case "dotted":
		return consts.LineStyleDotted
	default:
		return consts.LineStyleSolid
	}
}
