package css

import "strings"

func (s *ComputedStyle) applyTypographyProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "letter-spacing":
		s.LetterSpacing = ParseLength(ctx.val, s.FontSize)
	case "text-transform":
		s.TextTransform = strings.ToLower(strings.TrimSpace(ctx.val))
	case "text-indent":
		s.TextIndent = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "white-space":
		s.WhiteSpace = strings.ToLower(strings.TrimSpace(ctx.val))
	case "page-break-before", "break-before":
		s.PageBreakBefore = strings.TrimSpace(ctx.val)
	case "page-break-after", "break-after":
		s.PageBreakAfter = strings.TrimSpace(ctx.val)
	case "page-break-inside", "break-inside":
		s.BreakInside = strings.TrimSpace(ctx.val)
	case "list-style-type":
		s.ListStyleType = strings.TrimSpace(ctx.val)
	case "vertical-align":
		// Stored implicitly via usage context; no field needed yet.
	default:
		return false
	}
	return true
}
