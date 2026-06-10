package paper_test

import (
	"fmt"
	"testing"

	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNewFont(t *testing.T) {
	t.Parallel()
	// Arrange
	size := 10.0
	family := consts.FontFamilyArial
	style := fontstyle.Bold

	fpdf := newPDF(t)
	fpdf.EXPECT().SetFont(family, string(style), size)

	// Act
	font := gofpdf.NewFont(fpdf, size, family, style)

	// Assert
	assert.NotNil(t, font)
	assert.Equal(t, "*paper.Font", fmt.Sprintf("%T", font))
	assert.Equal(t, family, font.GetFamily())
	assert.Equal(t, style, font.GetStyle())
	assert.Equal(t, size, font.GetSize())
	assert.Equal(t, &props.Color{Red: 0, Green: 0, Blue: 0}, font.GetColor())
}

func TestFont_GetHeight(t *testing.T) {
	t.Parallel()
	// Arrange
	size := 10.0
	family := consts.FontFamilyArial
	style := fontstyle.Bold

	fpdf := newPDF(t)
	fpdf.EXPECT().SetFont(family, string(style), size)
	font := gofpdf.NewFont(fpdf, size, family, style)

	// Act
	height := font.GetHeight(family, style, size)

	// Assert
	assert.Equal(t, 3.527777777777778, height)
}

func TestFont_SetFamily(t *testing.T) {
	t.Parallel()
	// Arrange
	size := 10.0
	family := consts.FontFamilyArial
	style := fontstyle.Bold

	fpdf := newPDF(t)
	fpdf.EXPECT().SetFont(family, string(style), size)
	fpdf.EXPECT().SetFont(consts.FontFamilyHelvetica, string(style), size)
	font := gofpdf.NewFont(fpdf, size, family, style)

	// Act
	font.SetFamily(consts.FontFamilyHelvetica)

	// Assert
	assert.Equal(t, consts.FontFamilyHelvetica, font.GetFamily())
}

func TestFont_SetStyle(t *testing.T) {
	t.Parallel()
	// Arrange
	size := 10.0
	family := consts.FontFamilyArial
	style := fontstyle.Bold

	fpdf := newPDF(t)
	fpdf.EXPECT().SetFont(family, string(style), size)
	fpdf.EXPECT().SetFontStyle(string(fontstyle.BoldItalic))
	font := gofpdf.NewFont(fpdf, size, family, style)

	// Act
	font.SetStyle(fontstyle.BoldItalic)

	// Assert
	assert.Equal(t, fontstyle.BoldItalic, font.GetStyle())
}

func TestFont_SetSize(t *testing.T) {
	t.Parallel()
	// Arrange
	size := 10.0
	family := consts.FontFamilyArial
	style := fontstyle.Bold

	fpdf := newPDF(t)
	fpdf.EXPECT().SetFont(family, string(style), size)
	fpdf.EXPECT().SetFontSize(14.0)
	font := gofpdf.NewFont(fpdf, size, family, style)

	// Act
	font.SetSize(14.0)

	// Assert
	assert.Equal(t, 14.0, font.GetSize())
}

func TestFont_SetColor(t *testing.T) {
	t.Parallel()
	t.Run("when color is invalid, should not apply color", func(t *testing.T) {
		t.Parallel()
		// Arrange
		size := 10.0
		family := consts.FontFamilyArial
		style := fontstyle.Bold

		fpdf := newPDF(t)
		fpdf.EXPECT().SetFont(family, string(style), size)
		font := gofpdf.NewFont(fpdf, size, family, style)
		color := &props.Color{Red: 0, Green: 0, Blue: 0}

		// Act
		font.SetColor(nil)

		// Assert
		assert.Equal(t, color, font.GetColor())
	})
	t.Run("when color is valid, should apply color", func(t *testing.T) {
		t.Parallel()
		// Arrange
		size := 10.0
		family := consts.FontFamilyArial
		style := fontstyle.Bold

		fpdf := newPDF(t)
		fpdf.EXPECT().SetFont(family, string(style), size)
		fpdf.EXPECT().SetTextColor(200, 200, 200)
		font := gofpdf.NewFont(fpdf, size, family, style)
		color := &props.Color{Red: 200, Green: 200, Blue: 200}

		// Act
		font.SetColor(color)

		// Assert
		assert.Equal(t, color, font.GetColor())
	})
}
