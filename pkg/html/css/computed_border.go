package css

import "strings"

func (s *ComputedStyle) applyBorderProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "outline-width":
		s.OutlineWidth = ParseLength(ctx.val, ctx.parentFontSize)
	case "outline-style":
		s.OutlineStyle = strings.TrimSpace(ctx.val)
	case "outline-color":
		s.OutlineColor = ParseColor(ctx.val)
	case "outline-offset":
		s.OutlineOffset = ParseLength(ctx.val, ctx.parentFontSize)
	case "outline":
		parseOutlineShorthand(ctx.val, s, ctx.parentFontSize)
	case "border-top-width":
		s.BorderTopWidth = ParseLength(ctx.val, 0)
	case "border-right-width":
		s.BorderRightWidth = ParseLength(ctx.val, 0)
	case "border-bottom-width":
		s.BorderBottomWidth = ParseLength(ctx.val, 0)
	case "border-left-width":
		s.BorderLeftWidth = ParseLength(ctx.val, 0)
	case "border-top-style":
		s.BorderTopStyle = ctx.val
	case "border-right-style":
		s.BorderRightStyle = ctx.val
	case "border-bottom-style":
		s.BorderBottomStyle = ctx.val
	case "border-left-style":
		s.BorderLeftStyle = ctx.val
	case "border-top-color":
		s.BorderTopColor = ParseColor(ctx.val)
	case "border-right-color":
		s.BorderRightColor = ParseColor(ctx.val)
	case "border-bottom-color":
		s.BorderBottomColor = ParseColor(ctx.val)
	case "border-left-color":
		s.BorderLeftColor = ParseColor(ctx.val)
	case "border-color":
		c := ParseColor(ctx.val)
		s.BorderTopColor, s.BorderRightColor, s.BorderBottomColor, s.BorderLeftColor = c, c, c, c
	case "border-width":
		w := ParseLength(ctx.val, 0)
		s.BorderTopWidth, s.BorderRightWidth, s.BorderBottomWidth, s.BorderLeftWidth = w, w, w, w
	case "border-style":
		s.BorderTopStyle, s.BorderRightStyle, s.BorderBottomStyle, s.BorderLeftStyle = ctx.val, ctx.val, ctx.val, ctx.val
	case "border-radius":
		s.BorderRadius = ParseLength(ctx.val, 0)
	case "border-top-left-radius":
		s.BorderRadiusTopLeft = ParseLength(ctx.val, 0)
	case "border-top-right-radius":
		s.BorderRadiusTopRight = ParseLength(ctx.val, 0)
	case "border-bottom-left-radius":
		s.BorderRadiusBottomLeft = ParseLength(ctx.val, 0)
	case "border-bottom-right-radius":
		s.BorderRadiusBottomRight = ParseLength(ctx.val, 0)
	default:
		return false
	}
	return true
}

// parseOutlineShorthand parses "outline: <width> <style> <color>" in any order.
func parseOutlineShorthand(val string, s *ComputedStyle, parentFontSize float64) {
	styleKeywords := map[string]bool{
		"solid": true, "dashed": true, "dotted": true,
		"none": true, "hidden": true, "double": true,
		"groove": true, "ridge": true, "inset": true, "outset": true,
	}
	for token := range strings.FieldsSeq(val) {
		lower := strings.ToLower(token)
		if styleKeywords[lower] {
			s.OutlineStyle = lower
			continue
		}
		if c := ParseColor(token); c != nil {
			s.OutlineColor = c
			continue
		}
		if l := ParseLength(token, parentFontSize); l > 0 {
			s.OutlineWidth = l
		}
	}
}
