package paper_test

import (
	"fmt"
	"strings"
	"testing"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
)

func TestNewText(t *testing.T) {
	t.Parallel()
	text := gofpdf.NewText(newPDF(t), mocks.NewMath(t), mocks.NewFont(t))

	assert.NotNil(t, text)
	assert.Equal(t, "*paper.Text", fmt.Sprintf("%T", text))
}

func TestGetLinesHeight(t *testing.T) {
	t.Parallel()
	t.Run("when a text that occupies two lines is sent with EmptySpaceStrategy, should two is returned", func(t *testing.T) {
		t.Parallel()
		textProp := &props.Text{}
		textProp.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(textProp.Family, textProp.Style, textProp.Size)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("text").Return(5.0)  // First token just returns text
		pdf.EXPECT().GetStringWidth(" text").Return(6.0) // subsequent tokens return leading space

		text := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		height := text.GetLinesQuantity("text text text text", textProp, 11)

		assert.Equal(t, 2, height)
	})

	t.Run("When a text that occupies two lines is sent with EmptySpaceStrategy, should two is returned", func(t *testing.T) {
		t.Parallel()
		textProp := &props.Text{BreakLineStrategy: consts.BreakLineDash}
		textProp.MakeValid(&props.Font{Family: consts.FontFamilyArial, Size: 10, Style: fontstyle.Normal})

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(textProp.Family, textProp.Style, textProp.Size)

		pdf := newPDF(t)
		pdf.EXPECT().GetStringWidth("t").Return(1)
		pdf.EXPECT().GetStringWidth(" ").Return(1)
		pdf.EXPECT().GetStringWidth(" - ").Return(1)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })

		text := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		height := text.GetLinesQuantity("tttt tttt tttt tttt", textProp, 11)

		assert.Equal(t, 2, height)
	})

	t.Run("when BreakLineDash with non-ASCII text, should cache translated lines so a later Add renders translated glyphs", func(t *testing.T) {
		t.Parallel()
		// Regression: GetLinesQuantity computed the dash-broken lines from the
		// RAW text but cached them under the TRANSLATED key, so a later Add()
		// hit the cache and rendered untranslated glyphs.
		cell := &entity.Cell{X: 0, Y: 0, Width: 8, Height: 100}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family:            consts.FontFamilyArial,
			Style:             fontstyle.Normal,
			Size:              10,
			Align:             consts.AlignLeft,
			BreakLineStrategy: consts.BreakLineDash,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		// cp1252-style translator: non-ASCII runes are replaced with '.'
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string {
			var b strings.Builder
			for _, r := range s {
				if r < 0x80 {
					b.WriteRune(r)
				} else {
					b.WriteByte('.')
				}
			}
			return b.String()
		})
		// dash breaking of the TRANSLATED ".." into [".-", "."]
		pdf.EXPECT().GetStringWidth(" - ").Return(2.0)
		pdf.EXPECT().GetStringWidth(".").Return(5.0)
		// Add: translated text does not fit → uses the cached lines
		pdf.EXPECT().GetStringWidth("..").Return(12.0)
		pdf.EXPECT().GetStringWidth(".-").Return(6.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// rendered lines must contain translated glyphs, not the raw "żż"
		pdf.EXPECT().Text(0.0, 5.0, ".-")
		pdf.EXPECT().Text(0.0, 10.0, ".")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act: measure first (populates the cache), then render
		lines := sut.GetLinesQuantity("żż", textProp, 8)
		sut.Add("żż", cell, textProp)

		assert.Equal(t, 2, lines)
	})
}

func TestText_Add(t *testing.T) {
	t.Parallel()
	t.Run("when single line with left align and no color and no hyperlink, should render text once", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Top:    0,
			Left:   0,
			Right:  0,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// x = cell.X + Left = 0, y = cell.Y + Top + fontHeight = 5
		// Text(x + left_margin, y + top_margin, text) = Text(0, 5, "hello")
		pdf.EXPECT().Text(0.0, 5.0, "hello")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello", cell, textProp)
	})
	t.Run("when single line with left align and color is set, should apply color and restore original", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		customColor := &props.Color{Red: 200, Green: 100, Blue: 50}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Color:  customColor,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(customColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		pdf.EXPECT().Text(0.0, 5.0, "hello")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello", cell, textProp)
	})
	t.Run("when single line with left align and hyperlink is set, should apply blue color and render link", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		url := "https://example.com"
		textProp := &props.Text{
			Family:    consts.FontFamilyArial,
			Style:     fontstyle.Normal,
			Size:      10,
			Align:     consts.AlignLeft,
			Hyperlink: &url,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(&props.BlueColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		pdf.EXPECT().Text(0.0, 5.0, "hello")
		// LinkString(x+left, y+top-fontHeight, textWidth, fontHeight, url) = LinkString(0, 5-5, 20, 5, url)
		pdf.EXPECT().LinkString(0.0, 0.0, 20.0, 5.0, url)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello", cell, textProp)
	})
	t.Run("when single line with right align, should offset text by full remaining width", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignRight,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// dx = (colWidth - textWidth) / 1 = (100 - 20) / 1 = 80
		// Text(dx + xColOffset + left, yColOffset + top, text) = Text(80, 5, "hello")
		pdf.EXPECT().Text(80.0, 5.0, "hello")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello", cell, textProp)
	})
	t.Run("when single line with center align, should offset text by half remaining width", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignCenter,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// dx = (colWidth - textWidth) / 2 = (100 - 20) / 2 = 40
		// Text(dx + xColOffset + left, yColOffset + top, text) = Text(40, 5, "hello")
		pdf.EXPECT().Text(40.0, 5.0, "hello")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello", cell, textProp)
	})
	t.Run("when single line with justify align, should render left aligned without stretching", func(t *testing.T) {
		t.Parallel()
		// SEMANTICS CHANGE: this test previously asserted that a single justified
		// line was stretched to the full column width. Per CSS text-align: justify
		// semantics (and matching justifyRichTextLines), the last line of a
		// paragraph — and therefore a single line — is start-aligned, never
		// stretched. Only non-final wrapped lines are justified.
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignJustify,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// initial width check in Add: fits in 100
		pdf.EXPECT().GetStringWidth("hello world").Return(30.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// single (= last) line: rendered left aligned as one string
		pdf.EXPECT().Text(0.0, 5.0, "hello world")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hello world", cell, textProp)
	})
	t.Run("when multiple lines with justify align, should stretch all lines except the last", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignJustify,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// full text does not fit in width=100
		pdf.EXPECT().GetStringWidth("w1 w2 w3 w4").Return(200.0)
		// line breaking: "w1"(40) + " w2"(45) = 85 fits; " w3"(45) wraps;
		// "w3"(40) + " w4"(45) = 85 fits → lines: ["w1 w2", "w3 w4"]
		pdf.EXPECT().GetStringWidth("w1").Return(40.0)
		pdf.EXPECT().GetStringWidth(" w2").Return(45.0)
		pdf.EXPECT().GetStringWidth(" w3").Return(45.0)
		pdf.EXPECT().GetStringWidth("w3").Return(40.0)
		pdf.EXPECT().GetStringWidth(" w4").Return(45.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// line 0 (justified): textNotSpaces="w1w2"(80), spaceWidth=(100-80)/1=20
		// "w1" at x=0, finishX=40, next x=40+20=60 → "w2" at x=60
		pdf.EXPECT().GetStringWidth("w1 w2").Return(85.0)
		pdf.EXPECT().GetStringWidth("w1w2").Return(80.0)
		pdf.EXPECT().GetStringWidth(" ").Return(5.0)
		pdf.EXPECT().GetStringWidth("w2").Return(45.0)
		pdf.EXPECT().Text(0.0, 5.0, "w1")
		pdf.EXPECT().Text(60.0, 5.0, "w2")
		// line 1 (last line of the paragraph): left aligned, NOT stretched
		pdf.EXPECT().GetStringWidth("w3 w4").Return(85.0)
		pdf.EXPECT().Text(0.0, 10.0, "w3 w4")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("w1 w2 w3 w4", cell, textProp)
	})
	t.Run("when family uses mixed case, should still apply code page translation", func(t *testing.T) {
		t.Parallel()
		// Regression: textToUnicode compared the family with == against the
		// lowercase constants, so "Helvetica" skipped cp1252 translation and
		// rendered mojibake while "helvetica" worked.
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: "Helvetica",
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont("Helvetica", fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight("Helvetica", fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string {
			return strings.ReplaceAll(s, "é", "\xe9")
		})
		pdf.EXPECT().GetStringWidth("caf\xe9").Return(20.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// translated text must be rendered, not the raw UTF-8 input
		pdf.EXPECT().Text(0.0, 5.0, "caf\xe9")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("café", cell, textProp)
	})
	t.Run("when clamping paddings, should not mutate the caller's prop", func(t *testing.T) {
		t.Parallel()
		// Regression: Add wrote the clamped Top/Left/Right back through the
		// *props.Text pointer, so a component rendered once in a small cell kept
		// the clamped values for every later render.
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Top:    100, // exceeds cell.Height=50
			Left:   150, // exceeds cell.Width=100
			Right:  150, // exceeds cell.Width=100
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("").Return(0.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// clamped internally: top=50, left=100 → Text(100, 55, "")
		pdf.EXPECT().Text(100.0, 55.0, "")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("", cell, textProp)

		// Assert: the caller's prop keeps its original values
		assert.Equal(t, 100.0, textProp.Top)
		assert.Equal(t, 150.0, textProp.Left)
		assert.Equal(t, 150.0, textProp.Right)
	})
	t.Run("when text exceeds cell width with empty space strategy, should split into multiple lines", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 40, Height: 100}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family:            consts.FontFamilyArial,
			Style:             fontstyle.Normal,
			Size:              10,
			Align:             consts.AlignLeft,
			BreakLineStrategy: consts.BreakLineEmptySpace,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// full text does not fit in width=40
		pdf.EXPECT().GetStringWidth("word1 word2").Return(60.0)
		// getLinesBreakingLineFromSpace: "word1" fits, " word2" doesn't → new line
		pdf.EXPECT().GetStringWidth("word1").Return(25.0)
		pdf.EXPECT().GetStringWidth(" word2").Return(30.0)
		pdf.EXPECT().GetStringWidth("word2").Return(25.0)
		// addLine margins and height (called once per line)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// line 0: y = 0 + fontHeight(5) + index(0)*5 = 5
		pdf.EXPECT().Text(0.0, 5.0, "word1")
		// line 1: y = 5 + 1*5 = 10
		pdf.EXPECT().Text(0.0, 10.0, "word2")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("word1 word2", cell, textProp)
	})
	t.Run("when text exceeds cell width with dash strategy, should split into lines with dashes", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 8, Height: 100}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family:            consts.FontFamilyArial,
			Style:             fontstyle.Normal,
			Size:              10,
			Align:             consts.AlignLeft,
			BreakLineStrategy: consts.BreakLineDash,
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// full text "ab" does not fit in width=8
		pdf.EXPECT().GetStringWidth("ab").Return(12.0)
		// getLinesBreakingLineWithDash: dashSize=2
		// 'a': 0+2 > 8-2=6? No. width("a")=5. content="a", size=5
		// 'b': 5+2=7 > 6? Yes. content="a-", lines=["a-"]. width("b")=5. content="b", size=5
		// end: lines=["a-","b"]
		pdf.EXPECT().GetStringWidth(" - ").Return(2.0)
		pdf.EXPECT().GetStringWidth("a").Return(5.0)
		pdf.EXPECT().GetStringWidth("b").Return(5.0)
		// addLine per line: lineWidth from loop
		pdf.EXPECT().GetStringWidth("a-").Return(6.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// line 0: y = 0 + 5 + 0*5 = 5
		pdf.EXPECT().Text(0.0, 5.0, "a-")
		// line 1: y = 5 + 1*5 = 10
		pdf.EXPECT().Text(0.0, 10.0, "b")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("ab", cell, textProp)
	})
	t.Run("when top exceeds cell height, should clamp top to cell height", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Top:    100, // exceeds cell.Height=50, gets clamped to 50
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hi").Return(5.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// top clamped to 50: y = cell.Y + 50 + fontHeight = 0 + 50 + 5 = 55
		pdf.EXPECT().Text(0.0, 55.0, "hi")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("hi", cell, textProp)
	})
	t.Run("when left exceeds cell width, should clamp left to cell width", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Left:   150, // exceeds cell.Width=100, gets clamped to 100
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// left clamped to 100, right=0 → width = 100-100-0 = 0
		// x = 0 + 100 = 100
		// GetStringWidth("") = 0 ≤ 0 → single line
		pdf.EXPECT().GetStringWidth("").Return(0.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		// Text(x + left_margin, y + top_margin) = Text(100, 5, "")
		pdf.EXPECT().Text(100.0, 5.0, "")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("", cell, textProp)
	})
	t.Run("when right exceeds cell width, should clamp right to cell width", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 50}
		originalColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		textProp := &props.Text{
			Family: consts.FontFamilyArial,
			Style:  fontstyle.Normal,
			Size:   10,
			Align:  consts.AlignLeft,
			Right:  150, // exceeds cell.Width=100, gets clamped to 100
		}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetHeight(consts.FontFamilyArial, fontstyle.Normal, 10.0).Return(5.0)
		font.EXPECT().GetColor().Return(originalColor)
		font.EXPECT().SetColor(originalColor)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		// right clamped to 100, left=0 → width = 100-0-100 = 0
		// x = 0 + 0 = 0
		// GetStringWidth("") = 0 ≤ 0 → single line
		pdf.EXPECT().GetStringWidth("").Return(0.0)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0)
		pdf.EXPECT().Text(0.0, 5.0, "")

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Act
		sut.Add("", cell, textProp)
	})
}

func TestMeasureString(t *testing.T) {
	t.Parallel()
	t.Run("should set font and return string width", func(t *testing.T) {
		t.Parallel()
		textProp := &props.Text{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 12}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 12.0)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetStringWidth("hello").Return(30.5)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		got := sut.MeasureString("hello", textProp)

		assert.Equal(t, 30.5, got)
	})
}

func TestMeasureStringCachesDefaultCodePageTranslator(t *testing.T) {
	t.Parallel()

	textProp := &props.Text{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 12}
	font := mocks.NewFont(t)
	font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 12.0).Twice()

	pdf := newPDF(t)
	pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Once()
	pdf.EXPECT().GetStringWidth("hello").Return(20.0).Once()
	pdf.EXPECT().GetStringWidth("world").Return(25.0).Once()

	sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

	assert.Equal(t, 20.0, sut.MeasureString("hello", textProp))
	assert.Equal(t, 25.0, sut.MeasureString("world", textProp))
}

func TestAddTextAt(t *testing.T) {
	t.Parallel()
	t.Run("should set font and render text at absolute position", func(t *testing.T) {
		t.Parallel()
		textProp := &props.Text{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 12}

		font := mocks.NewFont(t)
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 12.0)

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s })
		pdf.EXPECT().GetMargins().Return(5.0, 10.0, 5.0, 5.0)
		pdf.EXPECT().Text(15.0, 25.0, "hello") // x+left=10+5=15, y+top=15+10=25

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		sut.AddTextAt(10, 15, "hello", textProp)
	})
}

func TestAddRichText(t *testing.T) {
	t.Parallel()
	t.Run("should restore font state after rendering", func(t *testing.T) {
		t.Parallel()
		origColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}

		font := mocks.NewFont(t)
		// Capture original state
		font.EXPECT().GetFont().Return(consts.FontFamilyArial, fontstyle.Normal, 10.0)
		font.EXPECT().GetColor().Return(origColor)
		// Runs: run 0 Helvetica Bold 12, run 1 Courier Normal 10
		font.EXPECT().SetFont(consts.FontFamilyHelvetica, fontstyle.Bold, 12.0).Maybe()
		font.EXPECT().SetFont(consts.FontFamilyCourier, fontstyle.Normal, 10.0).Maybe()
		font.EXPECT().SetFont(consts.FontFamilyArial, fontstyle.Normal, 10.0).Maybe() // restore
		font.EXPECT().GetHeight(consts.FontFamilyHelvetica, fontstyle.Bold, 12.0).Return(5.0).Maybe()
		font.EXPECT().GetHeight(consts.FontFamilyCourier, fontstyle.Normal, 10.0).Return(4.0).Maybe()
		font.EXPECT().SetColor(origColor).Maybe()
		font.EXPECT().GetColor().Return(origColor).Maybe()

		pdf := newPDF(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
		pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).Maybe()

		runs := []props.RichRun{
			{Text: "hello ", Family: consts.FontFamilyHelvetica, Style: fontstyle.Bold, Size: 12},
			{Text: "world", Family: consts.FontFamilyCourier, Style: fontstyle.Normal, Size: 10},
		}
		richProp := &props.RichText{}
		richProp.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)

		// Should not panic; font state restored via defer
		sut.AddRichText(runs, cell, richProp)
	})

	t.Run("all method-local state — concurrent calls on separate instances do not race", func(t *testing.T) {
		t.Parallel()
		// Each goroutine uses its own Text instance sharing no struct-level mutable state
		done := make(chan struct{}, 4)
		for range 4 {
			go func() {
				defer func() { done <- struct{}{} }()
				origColor := &props.Color{Red: 0, Green: 0, Blue: 0}
				cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}

				font := mocks.NewFont(t)
				font.EXPECT().GetFont().Return(consts.FontFamilyArial, fontstyle.Normal, 10.0)
				font.EXPECT().GetColor().Return(origColor)
				font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
				font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
				font.EXPECT().GetColor().Return(origColor).Maybe()
				font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

				pdf := newPDF(t)
				pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
				pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(5.0).Maybe()
				pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
				pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).Maybe()

				runs := []props.RichRun{{Text: "hi", Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 10}}
				richProp := &props.RichText{}
				richProp.MakeValid(nil)

				sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
				sut.AddRichText(runs, cell, richProp)
			}()
		}
		for range 4 {
			<-done
		}
	})
}
