package css

import "strings"

func (s *ComputedStyle) applyFontProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "font-family":
		s.FontFamily = strings.Trim(ctx.val, `'"`)
	case "font-size":
		s.FontSize = ParseLength(ctx.val, ctx.parentFontSize)
	case "font-weight":
		s.FontWeight = normFontWeight(ctx.val)
	case "font-style":
		s.FontStyle = ctx.val
	case "text-align":
		s.TextAlign = ctx.val
	case "text-decoration":
		s.TextDecoration = ctx.val
	case "line-height":
		s.LineHeight = ParseLength(ctx.val, s.FontSize)
	default:
		return false
	}
	return true
}

func normFontWeight(val string) string {
	switch val {
	case "bold", "bolder", "700", "800", "900":
		return "bold"
	default:
		return "normal"
	}
}
