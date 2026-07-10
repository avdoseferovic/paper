package css

import "strings"

func (s *ComputedStyle) applyTransformProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "transform":
		s.Transform = strings.TrimSpace(ctx.val)
	case "transform-origin":
		s.TransformOrigin = strings.TrimSpace(ctx.val)
	default:
		return false
	}
	return true
}
