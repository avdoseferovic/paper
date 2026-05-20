package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputedStyle_Opacity(t *testing.T) {
	t.Parallel()
	s := NewComputedStyle()

	// Default opacity is 1 (fully opaque)
	assert.Equal(t, 1.0, s.Opacity)

	// Apply opacity via CSS
	s.Apply("opacity", "0.5", nil)
	assert.InDelta(t, 0.5, s.Opacity, 0.001)

	// opacity: 1
	s2 := NewComputedStyle()
	s2.Apply("opacity", "1", nil)
	assert.Equal(t, 1.0, s2.Opacity)

	// opacity: 0
	s3 := NewComputedStyle()
	s3.Apply("opacity", "0", nil)
	assert.Equal(t, 0.0, s3.Opacity)
}

func TestComputedStyle_OpacityPercentage(t *testing.T) {
	t.Parallel()
	s := NewComputedStyle()
	s.Apply("opacity", "50%", nil)
	assert.InDelta(t, 0.5, s.Opacity, 0.001)
}

func TestComputedStyle_Opacity_Clamped(t *testing.T) {
	t.Parallel()
	// Opacity > 1 clamps to 1
	s := NewComputedStyle()
	s.Apply("opacity", "2.0", nil)
	assert.InDelta(t, 1.0, s.Opacity, 0.001)

	// Opacity < 0 clamps to 0
	s2 := NewComputedStyle()
	s2.Apply("opacity", "-0.5", nil)
	assert.InDelta(t, 0.0, s2.Opacity, 0.001)
}

func TestRGBAColor_AlphaField(t *testing.T) {
	t.Parallel()
	// rgba parsing stores alpha on RGBColor
	c := ParseColor("rgba(255, 0, 0, 0.5)")
	require.NotNil(t, c)
	assert.InDelta(t, 0.5, c.A, 0.001)

	// rgb() → A=1
	c2 := ParseColor("rgb(0, 128, 0)")
	require.NotNil(t, c2)
	assert.Equal(t, 1.0, c2.A)
}
