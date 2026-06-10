package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestCell_HasBorderRadius(t *testing.T) {
	t.Parallel()

	t.Run("nil cell", func(t *testing.T) {
		t.Parallel()
		var c *props.Cell
		assert.False(t, c.HasBorderRadius())
	})

	t.Run("zero cell", func(t *testing.T) {
		t.Parallel()
		c := &props.Cell{}
		assert.False(t, c.HasBorderRadius())
	})

	t.Run("uniform radius", func(t *testing.T) {
		t.Parallel()
		c := &props.Cell{BorderRadius: 4}
		assert.True(t, c.HasBorderRadius())
	})

	t.Run("per-corner only", func(t *testing.T) {
		t.Parallel()
		c := &props.Cell{BorderRadiusTopLeft: 2}
		assert.True(t, c.HasBorderRadius())
	})
}

func TestCell_EffectiveRadii(t *testing.T) {
	t.Parallel()

	t.Run("nil cell returns zero", func(t *testing.T) {
		t.Parallel()
		var c *props.Cell
		tl, tr, br, bl := c.EffectiveRadii()
		assert.Equal(t, 0.0, tl)
		assert.Equal(t, 0.0, tr)
		assert.Equal(t, 0.0, br)
		assert.Equal(t, 0.0, bl)
	})

	t.Run("uniform fills all corners", func(t *testing.T) {
		t.Parallel()
		c := &props.Cell{BorderRadius: 3.5}
		tl, tr, br, bl := c.EffectiveRadii()
		assert.Equal(t, 3.5, tl)
		assert.Equal(t, 3.5, tr)
		assert.Equal(t, 3.5, br)
		assert.Equal(t, 3.5, bl)
	})

	t.Run("per-corner overrides uniform", func(t *testing.T) {
		t.Parallel()
		c := &props.Cell{BorderRadius: 1, BorderRadiusTopLeft: 5, BorderRadiusBottomRight: 7}
		tl, tr, br, bl := c.EffectiveRadii()
		assert.Equal(t, 5.0, tl)
		assert.Equal(t, 1.0, tr) // falls back to uniform
		assert.Equal(t, 7.0, br)
		assert.Equal(t, 1.0, bl) // falls back to uniform
	})
}

func TestCell_ToMap(t *testing.T) {
	t.Parallel()
	t.Run("when cell is nil, should return nil", func(t *testing.T) {
		t.Parallel()
		// Arrange
		var sut *props.Cell

		// Act
		m := sut.ToMap()

		// Assert
		assert.Nil(t, m)
	})
	t.Run("when cell is filled, should return map filled correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := fixture.CellProp()

		// Act
		m := sut.ToMap()

		// Assert
		assert.Equal(t, border.Left, m["prop_border_type"])
		assert.Equal(t, 0.6, m["prop_border_thickness"])
		assert.Equal(t, consts.LineStyleDashed, m["prop_border_line_style"])
		assert.Equal(t, "RGB(255, 100, 50)", m["prop_background_color"])
		assert.Equal(t, "RGB(200, 80, 60)", m["prop_border_color"])
	})
}
