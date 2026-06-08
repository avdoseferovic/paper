package css

import "strings"

func (s *ComputedStyle) applyBoxProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "padding-top":
		s.PaddingTop = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "padding-right":
		s.PaddingRight = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "padding-bottom":
		s.PaddingBottom = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "padding-left":
		s.PaddingLeft = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "margin-top":
		s.MarginTop = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "margin-right":
		s.MarginRight = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "margin-bottom":
		s.MarginBottom = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "margin-left":
		s.MarginLeft = ParseLengthCtx(ctx.val, s.FontSize, ctx.ctxWidth)
	case "display":
		if ctx.val == "inline-flex" {
			s.Display = "flex"
		} else {
			s.Display = ctx.val
		}
	case "visibility":
		s.Visibility = strings.ToLower(strings.TrimSpace(ctx.val))
	case "width":
		s.Width = ParseLengthCtx(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "height":
		s.Height = ParseLength(ctx.val, 0)
	case "min-width":
		s.MinWidth = ParseLengthCtx(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "max-width":
		s.MaxWidth = ParseLengthCtx(ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "min-height":
		s.MinHeight = ParseLength(ctx.val, 0)
	case "max-height":
		s.MaxHeight = ParseLength(ctx.val, 0)
	case "object-fit":
		s.ObjectFit = strings.ToLower(strings.TrimSpace(ctx.val))
	case "object-position":
		s.ObjectPosition = strings.ToLower(strings.TrimSpace(ctx.val))
	default:
		return false
	}
	return true
}
