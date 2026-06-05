package paper_test

import (
	"testing"

	gofpdf "github.com/johnfercher/maroto/v2/internal/providers/paper"
	"github.com/johnfercher/maroto/v2/mocks"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// baseRichTextSetup creates a minimal set of pdf + font mocks for AddRichText.
// The caller adds expectations on top of what baseRichTextSetup declares.
func baseRichTextSetup(t *testing.T) (*mocks.Fpdf, *mocks.Font) {
	t.Helper()
	origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

	font := mocks.NewFont(t)
	font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
	font.EXPECT().GetColor().Return(origColor)
	font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
	font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
	font.EXPECT().GetColor().Return(origColor).Maybe()
	font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

	pdf := mocks.NewFpdf(t)
	pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
	pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
	pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
	pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).Maybe()

	return pdf, font
}

func TestAddRichText_LocalAnchor(t *testing.T) {
	t.Parallel()

	t.Run("run with LocalAnchor calls Link with y-lineHeight correction", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)

		// lineHeight from font.GetHeight = 4.0 mm (set in baseRichTextSetup)
		// GetStringWidth returns 8.0 for any string
		// Expected: Link(x=0, y=4-4=0, w=8, h=4, id=42)
		pdf.EXPECT().Link(
			mock.AnythingOfType("float64"), // x
			0.0,                            // y == baseline(4) - lineHeight(4) = 0
			8.0,                            // width == GetStringWidth
			4.0,                            // height == lineHeight
			42,
		).Once()

		resolver := func(name string) int { return 42 }
		prop := &props.RichText{AnchorResolver: resolver}
		prop.MakeValid(nil)

		runs := []props.RichRun{{Text: "click", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10, LocalAnchor: "section1"}}
		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
	})

	t.Run("run without LocalAnchor does not call Link", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)

		pdf.AssertNotCalled(t, "Link")

		prop := &props.RichText{}
		prop.MakeValid(nil)

		runs := []props.RichRun{{Text: "plain", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}
		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
	})
}

func TestAddRichText_LetterSpacing(t *testing.T) {
	t.Parallel()

	t.Run("run with LetterSpacing renders character by character", func(t *testing.T) {
		t.Parallel()
		origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
		font.EXPECT().GetColor().Return(origColor)
		font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
		font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
		font.EXPECT().GetColor().Return(origColor).Maybe()
		font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
		// Measurement pass: word "ab" + individual char measurements
		pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(2.0).Maybe()
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		// Render pass: 2 individual Text calls for "a" and "b"
		textCallCount := 0
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
			Run(func(x, y float64, s string) { textCallCount++ }).Maybe()

		runs := []props.RichRun{{Text: "ab", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10, LetterSpacing: 0.5}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)

		// "ab" has 2 runes → 2 Text() calls in the render pass
		if textCallCount != 2 {
			t.Errorf("expected 2 Text calls for 2-rune word with LetterSpacing, got %d", textCallCount)
		}
	})
}

func TestAddRichText_TextShadow(t *testing.T) {
	t.Parallel()

	t.Run("run with TextShadow draws shadow text before normal text", func(t *testing.T) {
		t.Parallel()
		origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
		font.EXPECT().GetColor().Return(origColor)
		font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
		font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
		font.EXPECT().GetColor().Return(origColor).Maybe()
		font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

		textCallOrder := []string{}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
		pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().SetTextColor(mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).
			Run(func(r, g, b int) { textCallOrder = append(textCallOrder, "setcolor") }).Maybe()
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
			Run(func(x, y float64, s string) { textCallOrder = append(textCallOrder, "text") }).Maybe()

		shadowColor := &props.Color{Red: 0, Green: 0, Blue: 0}
		runs := []props.RichRun{{
			Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10,
			TextShadow: &props.Shadow{OffsetX: 2, OffsetY: 2, Color: shadowColor},
		}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)

		// Order must be: setcolor(shadow), text(shadow), setcolor(restore), text(normal)
		if assert.GreaterOrEqual(t, len(textCallOrder), 4, "expected setcolor,text,setcolor,text: %v", textCallOrder) {
			assert.Equal(t, "setcolor", textCallOrder[0])
			assert.Equal(t, "text", textCallOrder[1])
			assert.Equal(t, "setcolor", textCallOrder[2])
			assert.Equal(t, "text", textCallOrder[3])
		}
	})

	t.Run("run without TextShadow does not set extra text color", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)
		textColorCalls := 0
		pdf.EXPECT().SetTextColor(mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).
			Run(func(r, g, b int) { textColorCalls++ }).Maybe()
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).Maybe()

		runs := []props.RichRun{{Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
		assert.Zero(t, textColorCalls)
	})
}

func TestAddRichText_WhiteSpace(t *testing.T) {
	t.Parallel()

	t.Run("nowrap keeps tokens on one rendered line", func(t *testing.T) {
		t.Parallel()
		origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
		font.EXPECT().GetColor().Return(origColor)
		font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
		font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
		font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

		var ys []float64
		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
		pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
			Run(func(_, y float64, _ string) { ys = append(ys, y) }).Maybe()

		prop := &props.RichText{WhiteSpace: "nowrap"}
		prop.MakeValid(nil)
		runs := []props.RichRun{{Text: "one two three", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 10, Height: 20}, prop)

		assert.NotEmpty(t, ys)
		for _, y := range ys {
			assert.Equal(t, ys[0], y)
		}
	})
}

func TestAddRichText_FirstLineIndent(t *testing.T) {
	t.Parallel()

	t.Run("offsets only the first rendered line", func(t *testing.T) {
		t.Parallel()
		origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

		font := mocks.NewFont(t)
		font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
		font.EXPECT().GetColor().Return(origColor)
		font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
		font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
		font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

		var xs []float64
		var texts []string
		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
		pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
			Run(func(x, _ float64, text string) {
				xs = append(xs, x)
				texts = append(texts, text)
			}).Maybe()

		prop := &props.RichText{FirstLineIndent: 5}
		prop.MakeValid(nil)
		runs := []props.RichRun{{Text: "one two", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 16, Height: 20}, prop)

		require.GreaterOrEqual(t, len(xs), 2)
		assert.Equal(t, "one", texts[0])
		assert.Equal(t, 5.0, xs[0])
		assert.Equal(t, "two", texts[len(texts)-1])
		assert.Equal(t, 0.0, xs[len(xs)-1])
	})
}

func TestAddRichText_Align(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		align align.Type
		wantX float64
	}{
		{name: "center", align: align.Center, wantX: 10},
		{name: "right", align: align.Right, wantX: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			origColor := &props.Color{Red: 0, Green: 0, Blue: 0}

			font := mocks.NewFont(t)
			font.EXPECT().GetFont().Return(fontfamily.Arial, fontstyle.Normal, 10.0)
			font.EXPECT().GetColor().Return(origColor)
			font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
			font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
			font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

			var xs []float64
			pdf := mocks.NewFpdf(t)
			pdf.EXPECT().UnicodeTranslatorFromDescriptor("").Return(func(s string) string { return s }).Maybe()
			pdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(10.0).Maybe()
			pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
			pdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).
				Run(func(x, _ float64, _ string) { xs = append(xs, x) }).Maybe()

			prop := &props.RichText{Align: tt.align}
			prop.MakeValid(nil)
			runs := []props.RichRun{{Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}

			sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
			sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 30, Height: 20}, prop)

			require.NotEmpty(t, xs)
			assert.Equal(t, tt.wantX, xs[0])
		})
	}
}

func TestAddRichText_Background(t *testing.T) {
	t.Parallel()

	t.Run("run with Background paints SetFillColor and Rect before text", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)

		// Background = yellow (no alpha)
		bg := &props.Color{Red: 255, Green: 255, Blue: 0}
		pdf.EXPECT().SetFillColor(255, 255, 0).Once()
		pdf.EXPECT().Rect(
			mock.AnythingOfType("float64"), // x
			mock.AnythingOfType("float64"), // y (= baseline - lineHeight)
			mock.AnythingOfType("float64"), // width
			mock.AnythingOfType("float64"), // lineHeight
			"F",
		).Once()
		// Fill color is reset to white after background rect
		pdf.EXPECT().SetFillColor(255, 255, 255).Once()

		runs := []props.RichRun{{Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10, Background: bg}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
	})

	t.Run("run without Background produces no SetFillColor or Rect calls", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)

		// Explicitly assert no SetFillColor and no Rect from background path
		pdf.AssertNotCalled(t, "SetFillColor", mock.Anything, mock.Anything, mock.Anything)
		pdf.AssertNotCalled(t, "Rect", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		runs := []props.RichRun{{Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
	})

	t.Run("run with semi-transparent Background wraps Rect in SetAlpha calls", func(t *testing.T) {
		t.Parallel()
		pdf, font := baseRichTextSetup(t)

		alpha := 0.4
		bg := &props.Color{Red: 255, Green: 0, Blue: 0, Alpha: &alpha}
		pdf.EXPECT().SetFillColor(255, 0, 0).Once()
		pdf.EXPECT().SetAlpha(0.4, "Normal").Once()
		pdf.EXPECT().Rect(
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			"F",
		).Once()
		pdf.EXPECT().SetAlpha(1.0, "Normal").Once()
		pdf.EXPECT().SetFillColor(255, 255, 255).Once()

		runs := []props.RichRun{{Text: "hello", Family: fontfamily.Arial, Style: fontstyle.Normal, Size: 10, Background: bg}}
		prop := &props.RichText{}
		prop.MakeValid(nil)

		sut := gofpdf.NewText(pdf, mocks.NewMath(t), font)
		sut.AddRichText(runs, &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}, prop)
	})
}
