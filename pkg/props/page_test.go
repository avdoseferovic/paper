package props_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestPlace_IsValid(t *testing.T) {
	t.Parallel()
	t.Run("when place is left_top, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.LeftTop

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is top, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.Top

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is right_top, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.RightTop

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is left_bottom, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.LeftBottom

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is bottom, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.Bottom

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is right_bottom, should return valid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.RightBottom

		// Act/Assert
		assert.True(t, sut.IsValid())
	})
	t.Run("when place is invalid should return invalid", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := props.Place("invalid")

		// Act/Assert
		assert.False(t, sut.IsValid())
	})
}

func TestPage_GetNumberTextProp(t *testing.T) {
	t.Parallel()
	t.Run("when place is left bottom, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.LeftBottom

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignLeft, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when place is left top, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.LeftTop

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 0.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignLeft, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when place is right bottom, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.RightBottom

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignRight, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when place is right top, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.RightTop

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 0.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignRight, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when place is right bottom, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.RightBottom

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignRight, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when place is bottom, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.Bottom

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignCenter, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
	t.Run("when offset y is set, should adjust top placement", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.Top
		prop.OffsetY = 2.5

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 2.5, textProp.Top)
	})
	t.Run("when offset y is set on bottom placement, should adjust from bottom baseline", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.Bottom
		prop.OffsetY = -3.0

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 97.0, textProp.Top)
	})
	t.Run("when offset x is set, should shift center placement horizontally", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.Bottom
		prop.OffsetX = 4.75

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 4.75, textProp.Left)
		assert.Equal(t, -4.75, textProp.Right)
	})
	t.Run("when offset x is negative, should shift placement left", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.Bottom
		prop.OffsetX = -2.25

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, -2.25, textProp.Left)
		assert.Equal(t, 2.25, textProp.Right)
	})
	t.Run("when place is left bottom, should map correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.PageProp()
		prop.Place = props.LeftBottom

		// Act
		textProp := prop.GetNumberTextProp(100)

		// Assert
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 0.0, textProp.Left)
		assert.Equal(t, 0.0, textProp.Right)
		assert.Equal(t, consts.FontFamilyHelvetica, textProp.Family)
		assert.Equal(t, fontstyle.Bold, textProp.Style)
		assert.Equal(t, 14.0, textProp.Size)
		assert.Equal(t, consts.AlignLeft, textProp.Align)
		assert.Equal(t, consts.BreakLineEmptySpace, textProp.BreakLineStrategy)
		assert.Equal(t, 0.0, textProp.VerticalPadding)
		assert.Equal(t, &props.Color{Red: 100, Green: 50, Blue: 200}, textProp.Color)
	})
}

func TestPage_GetPageString(t *testing.T) {
	t.Parallel()
	// Arrange
	prop := fixture.PageProp()

	// Act
	s := prop.GetPageString(10, 101)

	// Assert
	assert.Equal(t, "10 / 101", s)
}

func TestPageNumber_WithFont(t *testing.T) {
	t.Parallel()
	t.Run("when font already defined, should keep it", func(t *testing.T) {
		t.Parallel()
		// Arrange
		pageNumber := &props.PageNumber{
			Color:  &props.RedColor,
			Size:   15,
			Style:  fontstyle.Bold,
			Family: consts.FontFamilyHelvetica,
		}

		font := &props.Font{
			Color:  &props.BlueColor,
			Size:   13,
			Style:  fontstyle.Italic,
			Family: consts.FontFamilyArial,
		}

		// Act
		pageNumber.WithFont(font)

		// Assert
		assert.Equal(t, &props.RedColor, pageNumber.Color)
		assert.Equal(t, 15.0, pageNumber.Size)
		assert.Equal(t, fontstyle.Bold, pageNumber.Style)
		assert.Equal(t, consts.FontFamilyHelvetica, pageNumber.Family)
	})
	t.Run("when font not defined, should apply", func(t *testing.T) {
		t.Parallel()
		// Arrange
		pageNumber := &props.PageNumber{}

		font := &props.Font{
			Color:  &props.BlueColor,
			Size:   13,
			Style:  fontstyle.Italic,
			Family: consts.FontFamilyArial,
		}

		// Act
		pageNumber.WithFont(font)

		// Assert
		assert.Equal(t, &props.BlueColor, pageNumber.Color)
		assert.Equal(t, 13.0, pageNumber.Size)
		assert.Equal(t, fontstyle.Italic, pageNumber.Style)
		assert.Equal(t, consts.FontFamilyArial, pageNumber.Family)
	})
}

func TestPageNumber_AppendMap(t *testing.T) {
	t.Parallel()
	t.Run("when append map, should append correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		pageNumber := &props.PageNumber{
			Pattern: "pattern",
			Place:   props.Bottom,
			Color:   &props.RedColor,
			Size:    15,
			Style:   fontstyle.Bold,
			Family:  consts.FontFamilyHelvetica,
			OffsetY: -3,
			OffsetX: 4.75,
		}

		m := make(map[string]any)

		// Act
		m = pageNumber.AppendMap(m)

		// Assert
		assert.Equal(t, "pattern", m["page_number_pattern"])
		assert.Equal(t, props.Bottom, m["page_number_place"])
		assert.Equal(t, consts.FontFamilyHelvetica, m["page_number_family"])
		assert.Equal(t, fontstyle.Bold, m["page_number_style"])
		assert.Equal(t, 15.0, m["page_number_size"])
		assert.Equal(t, -3.0, m["page_number_offset_y"])
		assert.Equal(t, 4.75, m["page_number_offset_x"])
		assert.Equal(t, "RGB(255, 0, 0)", m["page_number_color"])
	})
}
