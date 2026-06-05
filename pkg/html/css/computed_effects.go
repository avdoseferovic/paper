package css

import (
	"strconv"
	"strings"
)

func (s *ComputedStyle) applyEffectsProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "color":
		s.Color = ParseColor(ctx.val)
	case "background-color":
		s.BackgroundColor = ParseColor(ctx.val)
	case "box-shadow":
		shadows, err := ParseShadow(ctx.val)
		if err == nil {
			s.BoxShadow = shadows
		} else if s.unsupportedHandler != nil {
			s.unsupportedHandler(ctx.prop, ctx.val)
		}
	case "text-shadow":
		shadows, err := ParseShadow(ctx.val)
		if err == nil && len(shadows) > 0 {
			s.TextShadow = &shadows[0]
		} else if s.unsupportedHandler != nil {
			s.unsupportedHandler(ctx.prop, ctx.val)
		}
	case "background-image":
		s.applyBackgroundImage(ctx)
	case "opacity":
		s.applyOpacity(ctx.val)
	default:
		return false
	}
	return true
}

func (s *ComputedStyle) applyBackgroundImage(ctx computedPropertyContext) {
	switch {
	case strings.HasPrefix(ctx.val, "linear-gradient("):
		g, err := ParseLinearGradient(ctx.val)
		if err == nil {
			s.BackgroundGradient = &Gradient{Kind: GradientLinear, Linear: g}
		} else if s.unsupportedHandler != nil {
			s.unsupportedHandler(ctx.prop, ctx.val)
		}
	case strings.HasPrefix(ctx.val, "radial-gradient("):
		g, err := ParseRadialGradient(ctx.val)
		if err == nil {
			s.BackgroundGradient = &Gradient{Kind: GradientRadial, Radial: g}
		} else if s.unsupportedHandler != nil {
			s.unsupportedHandler(ctx.prop, ctx.val)
		}
	default:
		if s.unsupportedHandler != nil {
			s.unsupportedHandler(ctx.prop, ctx.val)
		}
	}
}

func (s *ComputedStyle) applyOpacity(val string) {
	trimmed := strings.TrimSpace(val)
	v, err := strconv.ParseFloat(strings.TrimSuffix(trimmed, "%"), 64)
	if err != nil {
		return
	}
	if strings.HasSuffix(trimmed, "%") {
		v /= 100.0
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.Opacity = v
}
