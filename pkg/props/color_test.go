package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestWhiteColor(t *testing.T) {
	t.Parallel()
	// Act
	sut := props.WhiteColor

	// Assert
	assert.Equal(t, 255, sut.Red)
	assert.Equal(t, 255, sut.Green)
	assert.Equal(t, 255, sut.Blue)
}

func TestBlackColor(t *testing.T) {
	t.Parallel()
	// Act
	sut := props.BlackColor

	// Assert
	assert.Equal(t, 0, sut.Red)
	assert.Equal(t, 0, sut.Green)
	assert.Equal(t, 0, sut.Blue)
}

func TestRedColor(t *testing.T) {
	t.Parallel()
	// Act
	sut := props.RedColor

	// Assert
	assert.Equal(t, 255, sut.Red)
	assert.Equal(t, 0, sut.Green)
	assert.Equal(t, 0, sut.Blue)
}

func TestGreenColor(t *testing.T) {
	t.Parallel()
	// Act
	sut := props.GreenColor

	// Assert
	assert.Equal(t, 0, sut.Red)
	assert.Equal(t, 255, sut.Green)
	assert.Equal(t, 0, sut.Blue)
}

func TestBlueColor(t *testing.T) {
	t.Parallel()
	// Act
	blue := props.BlueColor

	// Assert
	assert.Equal(t, 0, blue.Red)
	assert.Equal(t, 0, blue.Green)
	assert.Equal(t, 255, blue.Blue)
}

func TestColor_Alpha_NilMeansOpaque(t *testing.T) {
	t.Parallel()
	// Existing struct literals (96 in codebase) have nil Alpha — must stay safe.
	c := props.Color{Red: 255, Green: 0, Blue: 0}
	assert.Nil(t, c.Alpha, "Alpha should default to nil (opaque)")
}

func TestColor_Alpha_Pointer(t *testing.T) {
	t.Parallel()
	a := 0.5
	c := props.Color{Red: 0, Green: 0, Blue: 255, Alpha: &a}
	assert.NotNil(t, c.Alpha)
	assert.InDelta(t, 0.5, *c.Alpha, 0.001)
}

func TestColor_ToString(t *testing.T) {
	t.Parallel()
	t.Run("when prop is nil, should return empty", func(t *testing.T) {
		t.Parallel()
		// Arrange
		var prop *props.Color

		// Act
		s := prop.ToString()

		// Assert
		assert.Empty(t, s)
	})
	t.Run("when prop is filled, should return correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.ColorProp()

		// Act
		s := prop.ToString()

		// Assert
		assert.Equal(t, "RGB(100, 50, 200)", s)
	})
}
