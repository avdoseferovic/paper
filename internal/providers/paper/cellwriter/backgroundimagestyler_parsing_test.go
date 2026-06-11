package cellwriter

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestParseBackgroundLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		base     float64
		expected float64
	}{
		{"percent of base", "50%", 200, 100},
		{"invalid percent", "abc%", 200, 0},
		{"millimetres", "5mm", 100, 5},
		{"centimetres", "2cm", 100, 20},
		{"points", "10pt", 100, 3.52778},
		{"pixels", "10px", 100, 2.64583},
		{"bare number", "7", 100, 7},
		{"garbage", "junk", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.expected, parseBackgroundLength(tt.value, tt.base), 0.0001)
		})
	}
}

func TestParseBackgroundOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		free     float64
		expected float64
	}{
		{"left is zero", "left", 10, 0},
		{"top is zero", "top", 10, 0},
		{"center is half", "center", 10, 5},
		{"right is full", "right", 10, 10},
		{"bottom is full", "bottom", 10, 10},
		{"percent of free space", "25%", 8, 2},
		{"invalid percent", "x%", 8, 0},
		{"length fallback", "3mm", 8, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.expected, parseBackgroundOffset(tt.value, tt.free), 0.0001)
		})
	}
}

func TestBackgroundImagePosition_SingleToken(t *testing.T) {
	t.Parallel()

	// cell at (10, 20), 100x50, image 40x10 → spaceX=60, spaceY=40.
	tests := []struct {
		name      string
		value     string
		expectedX float64
		expectedY float64
	}{
		{"empty defaults to origin", "", 10, 20},
		{"left", "left", 10, 40},
		{"right", "right", 70, 40},
		{"top", "top", 40, 20},
		{"bottom", "bottom", 40, 60},
		{"center", "center", 40, 40},
		{"offset value centers vertically", "10%", 16, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			x, y := backgroundImagePosition(tt.value, 10, 20, 100, 50, 40, 10)
			assert.InDelta(t, tt.expectedX, x, 0.0001)
			assert.InDelta(t, tt.expectedY, y, 0.0001)
		})
	}
}

func TestNormalizeBackgroundPositionTokens(t *testing.T) {
	t.Parallel()

	t.Run("vertical first token is swapped", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeBackgroundPositionTokens("top", "left")
		assert.Equal(t, "left", x)
		assert.Equal(t, "top", y)
	})

	t.Run("horizontal second token is swapped", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeBackgroundPositionTokens("center", "right")
		assert.Equal(t, "right", x)
		assert.Equal(t, "center", y)
	})

	t.Run("already ordered tokens are kept", func(t *testing.T) {
		t.Parallel()
		x, y := normalizeBackgroundPositionTokens("left", "bottom")
		assert.Equal(t, "left", x)
		assert.Equal(t, "bottom", y)
	})
}

func TestBackgroundImageRepeat_AllValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value            string
		repeatX, repeatY bool
	}{
		{"", true, true},
		{"repeat", true, true},
		{"REPEAT-Y", false, true},
		{"weird-value", true, true},
	}

	for _, tt := range tests {
		t.Run("value "+tt.value, func(t *testing.T) {
			t.Parallel()
			x, y := backgroundImageRepeat(tt.value)
			assert.Equal(t, tt.repeatX, x)
			assert.Equal(t, tt.repeatY, y)
		})
	}
}

func TestBackgroundImageSize_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero cell dimensions return zero", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("cover", 40, 20, 0, 10)
		assert.Equal(t, 0.0, w)
		assert.Equal(t, 0.0, h)
	})

	t.Run("zero image dimensions return zero", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("cover", 0, 20, 10, 10)
		assert.Equal(t, 0.0, w)
		assert.Equal(t, 0.0, h)
	})

	t.Run("empty value keeps image dimensions", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("", 40, 20, 10, 10)
		assert.InDelta(t, 40.0, w, 0.001)
		assert.InDelta(t, 20.0, h, 0.001)
	})

	t.Run("auto keeps image dimensions", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("auto", 40, 20, 10, 10)
		assert.InDelta(t, 40.0, w, 0.001)
		assert.InDelta(t, 20.0, h, 0.001)
	})

	t.Run("auto first token keeps image dimensions even with explicit height", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("auto 5mm", 40, 20, 20, 10)
		assert.InDelta(t, 40.0, w, 0.001)
		assert.InDelta(t, 20.0, h, 0.001)
	})

	t.Run("non-positive computed size falls back to image dimensions", func(t *testing.T) {
		t.Parallel()
		w, h := backgroundImageSize("0mm", 40, 20, 20, 10)
		assert.InDelta(t, 40.0, w, 0.001)
		assert.InDelta(t, 20.0, h, 0.001)
	})
}

func TestTileStart_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("non-positive size returns start", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 10.0, tileStart(10, 0, 0))
	})

	t.Run("start at or before min returns start", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 5.0, tileStart(5, 5, 4))
		assert.Equal(t, 3.0, tileStart(3, 5, 4))
	})
}
