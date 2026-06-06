package css_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/stretchr/testify/assert"
)

func TestApplyCtx_CalcPercentWidth(t *testing.T) {
	t.Parallel()

	t.Run("width calc(100% - 20mm) with ctxWidth 170mm gives 150mm", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.ApplyCtx("width", "calc(100% - 20mm)", nil, 170.0)
		assert.InDelta(t, 150.0, s.Width, 0.01)
	})

	t.Run("padding-left 10% with ctxWidth 200mm gives 20mm", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.ApplyCtx("padding-left", "10%", nil, 200.0)
		assert.InDelta(t, 20.0, s.PaddingLeft, 0.01)
	})

	t.Run("margin-right calc(5% + 2mm) with ctxWidth 100mm gives 7mm", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.ApplyCtx("margin-right", "calc(5% + 2mm)", nil, 100.0)
		assert.InDelta(t, 7.0, s.MarginRight, 0.01)
	})

	t.Run("width 50% with ctxWidth 0 gives 0 (backward compatible)", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.ApplyCtx("width", "50%", nil, 0)
		assert.InDelta(t, 0.0, s.Width, 0.01)
	})

	t.Run("min/max width use ctxWidth for percentages", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.ApplyCtx("min-width", "25%", nil, 80)
		s.ApplyCtx("max-width", "calc(50% - 5mm)", nil, 80)
		assert.InDelta(t, 20.0, s.MinWidth, 0.01)
		assert.InDelta(t, 35.0, s.MaxWidth, 0.01)
	})

	t.Run("Apply wraps ApplyCtx with ctxWidth=0 — same as before", func(t *testing.T) {
		t.Parallel()
		s1 := css.NewComputedStyle()
		s1.Apply("padding-top", "5mm", nil)
		s2 := css.NewComputedStyle()
		s2.ApplyCtx("padding-top", "5mm", nil, 0)
		assert.Equal(t, s1.PaddingTop, s2.PaddingTop)
	})
}
