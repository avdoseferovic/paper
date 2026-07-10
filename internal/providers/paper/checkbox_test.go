package paper_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNewCheckbox(t *testing.T) {
	t.Parallel()
	// Act
	sut := gofpdf.NewCheckbox(nil, nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*paper.Checkbox", fmt.Sprintf("%T", sut))
}

func TestCheckbox_Add(t *testing.T) {
	t.Parallel()
	t.Run("when label is empty and checkbox is unchecked, should draw only the border rect", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 10, Y: 20}
		prop := &props.Checkbox{
			Checked: false,
			Top:     2,
			Left:    3,
			Size:    5,
		}

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(5.0, 10.0, 5.0, 10.0)
		// x = cell.X + prop.Left + left = 10 + 3 + 5 = 18
		// y = cell.Y + prop.Top + top = 20 + 2 + 10 = 32
		fpdf.EXPECT().Rect(18.0, 32.0, 5.0, 5.0, "D")

		font := mocks.NewFont(t)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("", cell, prop)
	})
	t.Run("when label is empty and checkbox is checked, should draw border rect and X mark lines", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0}
		prop := &props.Checkbox{
			Checked: true,
			Top:     0,
			Left:    0,
			Size:    10,
		}

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// x = 0 + 0 + 0 = 0, y = 0 + 0 + 0 = 0
		fpdf.EXPECT().Rect(0.0, 0.0, 10.0, 10.0, "D")
		// diagonal top-left to bottom-right
		fpdf.EXPECT().Line(0.0, 0.0, 10.0, 10.0)
		// diagonal top-right to bottom-left
		fpdf.EXPECT().Line(10.0, 0.0, 0.0, 10.0)

		font := mocks.NewFont(t)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("", cell, prop)
	})
	t.Run("when label is set and checkbox is unchecked, should draw border rect and label text", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 5, Y: 5}
		prop := &props.Checkbox{
			Checked: false,
			Top:     1,
			Left:    1,
			Size:    8,
		}

		fontHeight := 3.0
		y := 9.0

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(2.0, 3.0, 2.0, 3.0)
		fpdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// x = 5 + 1 + 2 = 8, y = 5 + 1 + 3 = 9
		fpdf.EXPECT().Rect(8.0, y, 8.0, 8.0, "D")
		// labelX = x + size + gap = 8 + 8 + 1 = 17
		// labelY centers the cap height on the box: y + size/2 + fontHeight*0.35
		fpdf.EXPECT().Text(17.0, y+prop.Size/2+fontHeight*0.35, "label")

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(fontHeight)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("label", cell, prop)
	})
	t.Run("when label is set and checkbox is checked, should draw border rect, X mark lines and label text", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0}
		prop := &props.Checkbox{
			Checked: true,
			Top:     0,
			Left:    0,
			Size:    10,
		}

		fontHeight := 4.0

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		fpdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// x = 0, y = 0
		fpdf.EXPECT().Rect(0.0, 0.0, 10.0, 10.0, "D")
		fpdf.EXPECT().Line(0.0, 0.0, 10.0, 10.0)
		fpdf.EXPECT().Line(10.0, 0.0, 0.0, 10.0)
		// labelX = 0 + 10 + 1 = 11
		// labelY = 0 + 10/2 + fontHeight*0.35
		fpdf.EXPECT().Text(11.0, 0.0+prop.Size/2+fontHeight*0.35, "option")

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(consts.FontFamilyArial, fontstyle.Normal, 12.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 12.0).Return(fontHeight)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("option", cell, prop)
	})
	t.Run("when label has non-ASCII chars and font is a core family, should translate the label", func(t *testing.T) {
		t.Parallel()
		// Regression: the label was drawn without cp1252 translation, producing
		// mojibake for non-ASCII text with core fonts.
		cell := &entity.Cell{X: 0, Y: 0}
		prop := &props.Checkbox{
			Checked: false,
			Top:     0,
			Left:    0,
			Size:    10,
		}

		fontHeight := 4.0

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		fpdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string {
			return strings.ReplaceAll(s, "é", "\xe9")
		})
		fpdf.EXPECT().Rect(0.0, 0.0, 10.0, 10.0, "D")
		// translated label, not the raw UTF-8 input
		fpdf.EXPECT().Text(11.0, 0.0+prop.Size/2+fontHeight*0.35, "caf\xe9")

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(consts.FontFamilyArial, fontstyle.Normal, 12.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 12.0).Return(fontHeight)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("café", cell, prop)
	})
	t.Run("when font is a custom UTF-8 family, should not translate the label", func(t *testing.T) {
		t.Parallel()
		cell := &entity.Cell{X: 0, Y: 0}
		prop := &props.Checkbox{
			Checked: false,
			Top:     0,
			Left:    0,
			Size:    10,
		}

		fontHeight := 4.0

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// no UnicodeTranslatorFromDescriptor expectation: custom fonts pass through
		fpdf.EXPECT().Rect(0.0, 0.0, 10.0, 10.0, "D")
		fpdf.EXPECT().Text(11.0, 0.0+prop.Size/2+fontHeight*0.35, "café")

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return("roboto", fontstyle.Normal, 12.0)
		font.EXPECT().GetHeight("roboto", fontstyle.Normal, 12.0).Return(fontHeight)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("café", cell, prop)
	})
	t.Run("when margins are set, should offset x and y by margin values", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0}
		prop := &props.Checkbox{
			Checked: false,
			Top:     0,
			Left:    0,
			Size:    5,
		}

		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(10.0, 15.0, 10.0, 15.0)
		// x = 0 + 0 + 10 = 10, y = 0 + 0 + 15 = 15
		fpdf.EXPECT().Rect(10.0, 15.0, 5.0, 5.0, "D")

		font := mocks.NewFont(t)

		sut := gofpdf.NewCheckbox(fpdf, font)

		// Act
		sut.Add("", cell, prop)
	})
}
