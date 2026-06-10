package cellwriter

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestBackgroundImageSize(t *testing.T) {
	t.Parallel()

	t.Run("contain", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("contain", 40, 20, 10, 10)
		assert.InDelta(t, 10.0, w, 0.001)
		assert.InDelta(t, 5.0, h, 0.001)
	})

	t.Run("cover", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("cover", 40, 20, 10, 10)
		assert.InDelta(t, 20.0, w, 0.001)
		assert.InDelta(t, 10.0, h, 0.001)
	})

	t.Run("explicit width preserves aspect ratio", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("50%", 40, 20, 20, 10)
		assert.InDelta(t, 10.0, w, 0.001)
		assert.InDelta(t, 5.0, h, 0.001)
	})

	t.Run("explicit width and height", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("5mm 2mm", 40, 20, 20, 10)
		assert.InDelta(t, 5.0, w, 0.001)
		assert.InDelta(t, 2.0, h, 0.001)
	})
}

func TestBackgroundImagePosition(t *testing.T) {
	t.Parallel()

	x, y := backgroundImagePosition("center bottom", 10, 20, 100, 50, 40, 10)
	assert.InDelta(t, 40.0, x, 0.001)
	assert.InDelta(t, 60.0, y, 0.001)

	x, y = backgroundImagePosition("25% 50%", 0, 0, 100, 50, 20, 10)
	assert.InDelta(t, 20.0, x, 0.001)
	assert.InDelta(t, 20.0, y, 0.001)
}

func TestBackgroundImageRepeat(t *testing.T) {
	t.Parallel()

	repeatX, repeatY := backgroundImageRepeat("repeat-x")
	assert.True(t, repeatX)
	assert.False(t, repeatY)

	repeatX, repeatY = backgroundImageRepeat("no-repeat")
	assert.False(t, repeatX)
	assert.False(t, repeatY)
}

func TestTileStart(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0.0, tileStart(10, 0, 5))
	assert.Equal(t, -2.0, tileStart(18, 0, 5))
}
