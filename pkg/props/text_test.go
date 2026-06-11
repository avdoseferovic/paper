package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestText_MakeValid(t *testing.T) {
	t.Parallel()
	t.Run("when family is not defined, should define arial", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Family: "",
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, consts.FontFamilyArial, prop.Family)
	})
	t.Run("when style is not defined, should define normal", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Style: "",
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, fontstyle.Normal, prop.Style)
	})
	t.Run("when size is zero, should define 10.0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Size: 0.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 10.0, prop.Size)
	})
	t.Run("when align is not defined, should define left", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Align: "",
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, consts.AlignLeft, prop.Align)
	})
	t.Run("when top is less than 0, should become 0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Top: -5.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 0.0, prop.Top)
	})
	t.Run("when left is less than 0, should become 0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Left: -5.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 0.0, prop.Left)
	})
	t.Run("when right is less than 0, should become 0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Right: -5.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 0.0, prop.Right)
	})
	t.Run("when vertical padding is less than 0, should become 0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			VerticalPadding: -5.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 0.0, prop.VerticalPadding)
	})
	t.Run("when color is nil, should inherit color from font", func(t *testing.T) {
		t.Parallel()
		// Arrange
		color := &props.Color{Red: 100, Green: 50, Blue: 200}
		prop := props.Text{
			Color: nil,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal, Color: color})

		// Assert
		assert.Equal(t, color, prop.Color)
	})
	t.Run("when bottom is less than 0, should become 0", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			Bottom: -5.0,
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, 0.0, prop.Bottom)
	})
	t.Run("when break line strategy is empty, should apply empty space strategy", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{
			BreakLineStrategy: "",
		}

		// Act
		prop.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		// Assert
		assert.Equal(t, consts.BreakLineEmptySpace, prop.BreakLineStrategy)
	})
}

func TestText_ToMap(t *testing.T) {
	t.Parallel()
	t.Run("when all fields are zero/empty, should return empty map", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{}

		// Act
		m := prop.ToMap()

		// Assert
		assert.Empty(t, m)
	})
	t.Run("when text is filled, should return map filled", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.TextProp()

		// Act
		m := prop.ToMap()

		// Assert
		assert.Equal(t, 12.0, m["prop_top"])
		assert.Equal(t, 13.0, m["prop_bottom"])
		assert.Equal(t, 3.0, m["prop_left"])
		assert.Equal(t, consts.AlignRight, m["prop_align"])
		assert.Equal(t, consts.BreakLineDash, m["prop_breakline_strategy"])
		assert.Equal(t, 20.0, m["prop_vertical_padding"])
		assert.Equal(t, "RGB(100, 50, 200)", m["prop_color"])
		assert.Equal(t, "https://www.google.com", m["prop_hyperlink"])
	})
	t.Run("when right is set, should include right in map", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := props.Text{Right: 5}

		// Act
		m := prop.ToMap()

		// Assert
		assert.Equal(t, 5.0, m["prop_right"])
	})
}
