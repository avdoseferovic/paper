package css

import (
	"strconv"
	"strings"
)

func (s *ComputedStyle) applyFontProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "font-family":
		s.FontFamily = strings.Trim(ctx.val, `'"`)
	case "font-size":
		if size, ok := parseFontSize(ctx.val, ctx.parentFontSize); ok {
			s.FontSize = size
		}
	case "font-weight":
		s.FontWeight = normFontWeight(ctx.val)
	case "font-style":
		s.FontStyle = ctx.val
	case "text-align":
		s.TextAlign = ctx.val
	case "text-decoration":
		s.TextDecoration = ctx.val
	case "line-height":
		if multiplier, ok := parseLineHeight(ctx.val, s.FontSize, ctx.parentFontSize); ok {
			s.LineHeight = multiplier
		}
	default:
		return false
	}
	return true
}

// parseFontSize resolves a font-size value in mm, handling percentages and
// the CSS absolute/relative size keywords. Unparseable values report !ok so
// the inherited size is preserved instead of collapsing to 0.
func parseFontSize(value string, parentFontSize float64) (float64, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	base := parentFontSize
	if base == 0 {
		base = defaultRemMM
	}
	if frac, ok := ParsePercentage(value); ok {
		return base * frac, frac > 0
	}
	// Keyword factors follow the browser px equivalents relative to 16px.
	switch value {
	case "xx-small":
		return base * 9 / 16, true
	case "x-small":
		return base * 10 / 16, true
	case "small":
		return base * 13 / 16, true
	case "medium":
		return base, true
	case "large":
		return base * 18 / 16, true
	case "x-large":
		return base * 24 / 16, true
	case "xx-large":
		return base * 32 / 16, true
	case "smaller":
		return base * 0.833, true
	case "larger":
		return base * 1.2, true
	}
	size := ParseLength(value, parentFontSize)
	if size <= 0 {
		return 0, false
	}
	return size, true
}

// parseLineHeight resolves a line-height value as a multiplier of the font
// size. Unitless numbers pass through; percentages and lengths convert to a
// multiplier so downstream layout (which expects a factor) stays correct.
func parseLineHeight(value string, fontSize, parentFontSize float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "normal") {
		return 1.0, true
	}
	if frac, ok := ParsePercentage(value); ok {
		return frac, frac > 0
	}
	if unitless, err := strconv.ParseFloat(value, 64); err == nil {
		return unitless, unitless > 0
	}
	base := fontSize
	if base == 0 {
		base = parentFontSize
	}
	if base == 0 {
		base = defaultRemMM
	}
	length := ParseLength(value, base)
	if length <= 0 {
		return 0, false
	}
	return length / base, true
}

func normFontWeight(val string) string {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "bold", "bolder", "700", "800", "900":
		return "bold"
	case "500", "600":
		return strings.TrimSpace(val)
	default:
		return "normal"
	}
}
