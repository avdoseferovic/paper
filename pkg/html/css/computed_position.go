package css

import (
	"strconv"
	"strings"
)

func (s *ComputedStyle) applyPositionProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "position":
		s.applyPosition(ctx.val)
	case "top":
		s.applyPositionOffset(&s.Top, ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case cssValueRight:
		s.applyPositionOffset(&s.Right, ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "bottom":
		s.applyPositionOffset(&s.Bottom, ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case cssValueLeft:
		s.applyPositionOffset(&s.Left, ctx.val, ctx.parentFontSize, ctx.ctxWidth)
	case "z-index":
		s.applyZIndex(ctx.val)
	default:
		return false
	}
	return true
}

func (s *ComputedStyle) applyPosition(value string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "static", "relative", "absolute", "fixed", "sticky":
		s.Position = strings.ToLower(strings.TrimSpace(value))
	}
}

// applyPositionOffset resolves a top/right/bottom/left declaration. CSS "auto"
// keeps the zero value, matching consumers that treat 0 as unset.
func (s *ComputedStyle) applyPositionOffset(target *float64, value string, parentFontSize, ctxWidth float64) {
	if strings.EqualFold(strings.TrimSpace(value), cssValueAuto) {
		return
	}
	*target = ParseLengthCtx(value, parentFontSize, ctxWidth)
}

func (s *ComputedStyle) applyZIndex(value string) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, cssValueAuto) {
		return
	}
	z, err := strconv.Atoi(value)
	if err != nil {
		return
	}
	s.ZIndex = z
	s.ZIndexSet = true
}
