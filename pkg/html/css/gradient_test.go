package css_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLinearGradient(t *testing.T) {
	t.Parallel()

	t.Run("to right two stops", func(t *testing.T) {
		t.Parallel()
		g, err := css.ParseLinearGradient("linear-gradient(to right, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 90.0, g.AngleDeg, 0.1) // "to right" == 90deg
		require.Len(t, g.Stops, 2)
		assert.Equal(t, 255, g.Stops[0].Color.R)
		assert.Equal(t, 0, g.Stops[0].Color.G)
		assert.Equal(t, 0, g.Stops[0].Color.B)
		assert.Equal(t, 0, g.Stops[1].Color.R)
		assert.Equal(t, 0, g.Stops[1].Color.G)
		assert.Equal(t, 255, g.Stops[1].Color.B)
	})

	t.Run("45deg three stops with positions", func(t *testing.T) {
		t.Parallel()
		g, err := css.ParseLinearGradient("linear-gradient(45deg, #ff0000 0%, #00ff00 50%, #0000ff 100%)")
		require.NoError(t, err)
		assert.InDelta(t, 45.0, g.AngleDeg, 0.1)
		require.Len(t, g.Stops, 3)
		assert.InDelta(t, 0.0, g.Stops[0].Position, 0.01)
		assert.InDelta(t, 0.5, g.Stops[1].Position, 0.01)
		assert.InDelta(t, 1.0, g.Stops[2].Position, 0.01)
	})

	t.Run("to bottom implicit positions", func(t *testing.T) {
		t.Parallel()
		g, err := css.ParseLinearGradient("linear-gradient(to bottom, white, black)")
		require.NoError(t, err)
		assert.InDelta(t, 180.0, g.AngleDeg, 0.1) // "to bottom" == 180deg
		require.Len(t, g.Stops, 2)
		assert.InDelta(t, 0.0, g.Stops[0].Position, 0.01)
		assert.InDelta(t, 1.0, g.Stops[1].Position, 0.01)
	})

	t.Run("invalid gradient returns error", func(t *testing.T) {
		t.Parallel()
		_, err := css.ParseLinearGradient("linear-gradient(weird stuff)")
		assert.Error(t, err)
	})
}

func TestParseRadialGradient(t *testing.T) {
	t.Parallel()

	t.Run("circle at center two stops", func(t *testing.T) {
		t.Parallel()
		g, err := css.ParseRadialGradient("radial-gradient(circle at center, white, black)")
		require.NoError(t, err)
		assert.True(t, g.Circle)
		require.Len(t, g.Stops, 2)
		assert.Equal(t, 255, g.Stops[0].Color.R)
		assert.Equal(t, 255, g.Stops[0].Color.G)
		assert.Equal(t, 255, g.Stops[0].Color.B)
	})

	t.Run("default (no qualifier) two stops", func(t *testing.T) {
		t.Parallel()
		g, err := css.ParseRadialGradient("radial-gradient(red, blue)")
		require.NoError(t, err)
		require.Len(t, g.Stops, 2)
	})
}
