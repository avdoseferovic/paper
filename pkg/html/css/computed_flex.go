package css

import (
	"strconv"
	"strings"
)

func (s *ComputedStyle) applyFlexProperty(ctx computedPropertyContext) bool {
	switch ctx.prop {
	case "flex-direction":
		s.FlexDirection = ctx.val
	case "justify-content":
		s.JustifyContent = ctx.val
	case "align-self":
		s.AlignSelf = strings.TrimSpace(ctx.val)
	case "flex-wrap":
		s.FlexWrap = strings.TrimSpace(ctx.val)
	case "order":
		v, err := strconv.Atoi(strings.TrimSpace(ctx.val))
		if err == nil {
			s.Order = v
		}
	case "align-items":
		s.AlignItems = ctx.val
	case "flex-grow":
		v, err := strconv.ParseFloat(ctx.val, 64)
		if err == nil {
			s.FlexGrow = v
		}
	case "flex-shrink":
		v, err := strconv.ParseFloat(ctx.val, 64)
		if err == nil {
			s.FlexShrink = v
		}
	case "flex-basis":
		s.applyFlexBasis(ctx.val, ctx.parentFontSize)
	case "gap":
		s.applyGap(ctx.val, ctx.parentFontSize)
	case "row-gap":
		s.RowGap = ParseLength(ctx.val, ctx.parentFontSize)
	case "column-gap":
		s.ColumnGap = ParseLength(ctx.val, ctx.parentFontSize)
	default:
		return false
	}
	return true
}

func (s *ComputedStyle) applyFlexBasis(val string, parentFontSize float64) {
	switch val {
	case cssValueAuto:
		s.FlexBasisAuto = true
		s.FlexBasis = 0
		s.FlexBasisPct = 0
	default:
		if pct, ok := ParsePercentage(val); ok {
			s.FlexBasisPct = pct * 100
			s.FlexBasis = 0
			s.FlexBasisAuto = false
		} else {
			s.FlexBasis = ParseLength(val, parentFontSize)
			s.FlexBasisAuto = false
			s.FlexBasisPct = 0
		}
	}
}

func (s *ComputedStyle) applyGap(val string, parentFontSize float64) {
	parts := strings.Fields(val)
	if len(parts) == 1 {
		v := ParseLength(parts[0], parentFontSize)
		s.RowGap = v
		s.ColumnGap = v
	} else if len(parts) >= 2 {
		s.RowGap = ParseLength(parts[0], parentFontSize)
		s.ColumnGap = ParseLength(parts[1], parentFontSize)
	}
}
