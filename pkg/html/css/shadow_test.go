package css_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseShadow(t *testing.T) {
	t.Parallel()

	t.Run("x y color", func(t *testing.T) {
		t.Parallel()
		shadows, err := css.ParseShadow("2mm 3mm rgba(0,0,0,0.5)")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		assert.InDelta(t, 2.0, shadows[0].OffsetX, 0.01)
		assert.InDelta(t, 3.0, shadows[0].OffsetY, 0.01)
		assert.InDelta(t, 0.0, shadows[0].BlurRadius, 0.01)
		assert.False(t, shadows[0].Inset)
		assert.NotNil(t, shadows[0].Color)
	})

	t.Run("x y blur color", func(t *testing.T) {
		t.Parallel()
		shadows, err := css.ParseShadow("0 4mm 6mm #00000033")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		assert.InDelta(t, 0.0, shadows[0].OffsetX, 0.01)
		assert.InDelta(t, 4.0, shadows[0].OffsetY, 0.01)
		assert.InDelta(t, 6.0, shadows[0].BlurRadius, 0.01)
	})

	t.Run("inset keyword", func(t *testing.T) {
		t.Parallel()
		shadows, err := css.ParseShadow("inset 0 2mm 4mm red")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		assert.True(t, shadows[0].Inset)
	})

	t.Run("two shadows comma-separated", func(t *testing.T) {
		t.Parallel()
		shadows, err := css.ParseShadow("2mm 0 red, -2mm 0 blue")
		require.NoError(t, err)
		assert.Len(t, shadows, 2)
	})

	t.Run("empty value returns error", func(t *testing.T) {
		t.Parallel()
		_, err := css.ParseShadow("")
		assert.Error(t, err)
	})
}
