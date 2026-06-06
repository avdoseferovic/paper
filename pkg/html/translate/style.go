package translate

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
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

func computeInlineNodeStyle(sheet *stylesheet, n *dom.Node, parent *css.ComputedStyle) *css.ComputedStyle {
	s := inheritInlineStyle(parent)
	if sheet != nil && n.RawNode() != nil {
		sheet.applyToNodeCtx(n.RawNode(), s, parent, 0)
	}
	inline := n.InlineStyle()
	if inline != "" {
		for prop, val := range parseInlineStyle(inline) {
			s.Apply(prop, val, parent)
		}
	}
	return s
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
	s.Opacity = parent.Opacity
	s.LetterSpacing = parent.LetterSpacing
	s.TextTransform = parent.TextTransform
	s.VerticalAlign = parent.VerticalAlign
	if len(parent.Vars) > 0 {
		s.Vars = make(map[string]string, len(parent.Vars))
		for k, v := range parent.Vars {
			s.Vars[k] = v
		}
	}
	return s
}

func blockInlineStyle(style *css.ComputedStyle) *css.ComputedStyle {
	s := inheritInlineStyle(style)
	s.BackgroundColor = nil
	s.VerticalAlign = ""
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

func applyInlineStyleToRun(style *css.ComputedStyle, run *props.RichRun) {
	if style == nil || run == nil {
		return
	}
	if family := firstFontFamily(style.FontFamily); family != "" && run.Family == "" {
		run.Family = family
	}
	if style.FontWeight == "bold" || isItalicCSSFontStyle(style.FontStyle) {
		run.Style = mergeCSSFontStyle(run.Style, style.FontWeight, style.FontStyle)
	}
	if style.FontSize > 0 && run.Size == 0 {
		// FontSize is in mm; props.RichRun expects pt — convert.
		run.Size = style.FontSize / 0.352778
	}
	if style.Color != nil && run.Color == nil {
		run.Color = toPropsColor(style.Color, effectiveOpacity(style))
	}
	if style.BackgroundColor != nil && run.Background == nil {
		run.Background = toPropsColor(style.BackgroundColor, effectiveOpacity(style))
	}
	if style.LetterSpacing > 0 && run.LetterSpacing == 0 {
		run.LetterSpacing = style.LetterSpacing
	}
	applyTextDecoration(style.TextDecoration, run)
	if align := richRunVerticalAlignFromCSS(style.VerticalAlign); align != "" {
		run.VerticalAlign = align
	}
	if style.TextTransform != "" && style.TextTransform != "none" {
		run.Text = css.ApplyTextTransform(run.Text, style.TextTransform)
	}
	if len(style.TextShadows) > 0 && len(run.TextShadows) == 0 && run.TextShadow == nil {
		run.TextShadows = cssShadowsToProps(style.TextShadows)
		if len(run.TextShadows) > 0 {
			run.TextShadow = &run.TextShadows[0]
		}
	} else if style.TextShadow != nil && run.TextShadow == nil {
		run.TextShadow = cssShadowToProps(style.TextShadow)
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

func mergeCSSFontStyle(existing fontstyle.Type, weight, style string) fontstyle.Type {
	bold := strings.Contains(string(existing), string(fontstyle.Bold)) || weight == "bold"
	italic := strings.Contains(string(existing), string(fontstyle.Italic)) || isItalicCSSFontStyle(style)
	switch {
	case bold && italic:
		return fontstyle.BoldItalic
	case bold:
		return fontstyle.Bold
	case italic:
		return fontstyle.Italic
	default:
		return existing
	}
}

func applyTextDecoration(value string, run *props.RichRun) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "none" {
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
	case "baseline":
		return "baseline"
	case "sub":
		return "sub"
	case "super", "sup":
		return "super"
	default:
		return ""
	}
}

func generatedContentText(value string, n *dom.Node) (string, bool) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "normal", "none":
		return "", false
	}

	var out strings.Builder
	consumed := false
	for value != "" {
		value = strings.TrimLeft(value, " \t\r\n\f")
		if value == "" {
			break
		}
		lower := strings.ToLower(value)
		if strings.HasPrefix(lower, "attr(") {
			end := strings.IndexByte(value, ')')
			if end < 0 {
				return "", false
			}
			name := strings.Trim(strings.TrimSpace(value[len("attr("):end]), `'"`)
			if name != "" && n != nil {
				out.WriteString(n.Attr(name))
			}
			value = value[end+1:]
			consumed = true
			continue
		}
		if value[0] == '"' || value[0] == '\'' {
			text, rest, ok := readCSSContentString(value)
			if !ok {
				return "", false
			}
			out.WriteString(text)
			value = rest
			consumed = true
			continue
		}
		return "", false
	}
	return out.String(), consumed
}

func readCSSContentString(value string) (string, string, bool) {
	if value == "" {
		return "", "", false
	}
	quote := value[0]
	var out strings.Builder
	escaped := false
	for i := 1; i < len(value); i++ {
		ch := value[i]
		if escaped {
			switch ch {
			case 'a', 'A':
				out.WriteByte('\n')
			default:
				out.WriteByte(ch)
			}
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case quote:
			return out.String(), value[i+1:], true
		default:
			out.WriteByte(ch)
		}
	}
	return "", "", false
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
	if n != nil && n.Attr("hidden") != "" {
		return true
	}
	return n != nil &&
		(strings.Contains(n.InlineStyle(), "display:none") ||
			strings.Contains(n.InlineStyle(), "display: none"))
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
