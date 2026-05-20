package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
)

func TestToPropsColor_NilAlphaForOpaque(t *testing.T) {
	t.Parallel()
	c := &css.RGBColor{R: 100, G: 50, B: 200, A: 1.0}
	out := toPropsColor(c, 1.0)
	require.NotNil(t, out)
	assert.Equal(t, 100, out.Red)
	assert.Equal(t, 50, out.Green)
	assert.Equal(t, 200, out.Blue)
	assert.Nil(t, out.Alpha, "opaque colors must leave Alpha nil")
}

func TestToPropsColor_ExplicitAlpha(t *testing.T) {
	t.Parallel()
	c := &css.RGBColor{R: 0, G: 0, B: 255, A: 0.5}
	out := toPropsColor(c, 1.0)
	require.NotNil(t, out)
	require.NotNil(t, out.Alpha)
	assert.InDelta(t, 0.5, *out.Alpha, 0.001)
}

func TestToPropsColor_OpacityMultiplied(t *testing.T) {
	t.Parallel()
	c := &css.RGBColor{R: 0, G: 0, B: 255, A: 0.8}
	out := toPropsColor(c, 0.5) // opacity:0.5 → effective alpha 0.4
	require.NotNil(t, out)
	require.NotNil(t, out.Alpha)
	assert.InDelta(t, 0.4, *out.Alpha, 0.001)
}

func TestToPropsColor_OpacityOneOpaqueColor(t *testing.T) {
	t.Parallel()
	// opacity:1 + A=1 must produce Alpha=nil so existing render paths skip SetAlpha.
	c := &css.RGBColor{R: 10, G: 20, B: 30, A: 1.0}
	out := toPropsColor(c, 1.0)
	require.NotNil(t, out)
	assert.Nil(t, out.Alpha)
}

func TestToPropsColor_OpacityReducesOpaque(t *testing.T) {
	t.Parallel()
	// Opaque color under opacity:0.5 yields Alpha=0.5
	c := &css.RGBColor{R: 255, G: 255, B: 255, A: 1.0}
	out := toPropsColor(c, 0.5)
	require.NotNil(t, out)
	require.NotNil(t, out.Alpha)
	assert.InDelta(t, 0.5, *out.Alpha, 0.001)
}

func TestToPropsColor_NilInput(t *testing.T) {
	t.Parallel()
	assert.Nil(t, toPropsColor(nil, 1.0))
	assert.Nil(t, toPropsColor(nil, 0.5))
}

func TestEffectiveOpacity_Clamps(t *testing.T) {
	t.Parallel()
	s := css.NewComputedStyle()
	s.Opacity = 1.5
	assert.Equal(t, 1.0, effectiveOpacity(s))

	s.Opacity = -0.5
	assert.Equal(t, 0.0, effectiveOpacity(s))

	s.Opacity = 0.3
	assert.InDelta(t, 0.3, effectiveOpacity(s), 0.001)
}
