package paper_test

import (
	"errors"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/merror"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// richTextProviderSetup builds a provider whose Text dependency is a real
// *paper.Text (so the richText fast path is active) backed by mocks.
func richTextProviderSetup(t *testing.T) (core.Provider, *pdfMock, *mocks.Font) {
	t.Helper()

	font := mocks.NewFont(t)
	font.EXPECT().GetFont().Return("roboto", fontstyle.Normal, 10.0).Maybe()
	font.EXPECT().GetColor().Return(&props.Color{}).Maybe()
	font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
	font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
	font.EXPECT().GetHeight(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Return(4.0).Maybe()

	fpdf := newPDF(t)

	text := gofpdf.NewText(fpdf, mocks.NewMath(t), font)
	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf, Text: text, Font: font})
	return sut, fpdf, font
}

func TestProvider_MeasureString(t *testing.T) {
	t.Parallel()

	t.Run("without richtext renderer returns zero", func(t *testing.T) {
		t.Parallel()
		sut := gofpdf.New(&gofpdf.Dependencies{Text: mocks.NewText(t)})

		got := sut.(core.RichTextProvider).MeasureString("hello", nil)

		assert.Equal(t, 0.0, got)
	})

	t.Run("delegates to the richtext renderer", func(t *testing.T) {
		t.Parallel()
		sut, fpdf, _ := richTextProviderSetup(t)
		fpdf.EXPECT().GetStringWidth("hello").Return(42.0).Once()

		got := sut.(core.RichTextProvider).MeasureString("hello", &props.Text{Family: "roboto", Size: 12})

		assert.Equal(t, 42.0, got)
	})
}

func TestProvider_AddTextAt(t *testing.T) {
	t.Parallel()

	t.Run("without richtext renderer is a no-op", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()
		sut := gofpdf.New(&gofpdf.Dependencies{})

		sut.(core.RichTextProvider).AddTextAt(1, 2, "hi", &props.Text{})
	})

	t.Run("draws baseline text at absolute position including margins", func(t *testing.T) {
		t.Parallel()
		sut, fpdf, _ := richTextProviderSetup(t)
		fpdf.EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0).Once()
		fpdf.EXPECT().Text(15.0, 27.0, "hi").Once()

		sut.(core.RichTextProvider).AddTextAt(10, 20, "hi", &props.Text{Family: "roboto", Size: 12})
	})
}

func TestProvider_AddRichText(t *testing.T) {
	t.Parallel()

	t.Run("without richtext renderer is a no-op even with outline", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf, Text: mocks.NewText(t)})

		prop := &props.RichText{Outline: &props.Outline{Level: 1}}
		sut.(core.RichTextProvider).AddRichText([]props.RichRun{{Text: "x"}}, &entity.Cell{Width: 50}, prop)

		fpdf.AssertNotCalled(t, "Bookmark")
	})

	t.Run("with outline bookmarks the joined run text and renders", func(t *testing.T) {
		t.Parallel()
		sut, fpdf, _ := richTextProviderSetup(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		fpdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()
		fpdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("string")).Maybe()
		fpdf.EXPECT().Bookmark("Hello World", 2, 3.0).Once()

		prop := &props.RichText{Outline: &props.Outline{Level: 2}}
		prop.MakeValid(nil)

		runs := []props.RichRun{
			{Text: "Hello ", Family: "roboto", Size: 10},
			{Text: "World", Family: "roboto", Size: 10},
		}
		sut.(core.RichTextProvider).AddRichText(runs, &entity.Cell{X: 0, Y: 3, Width: 50, Height: 20}, prop)
	})
}

func TestProvider_MeasureRichText(t *testing.T) {
	t.Parallel()

	t.Run("without richtext renderer returns zero", func(t *testing.T) {
		t.Parallel()
		sut := gofpdf.New(&gofpdf.Dependencies{})

		got := sut.(core.RichTextMeasurer).MeasureRichText([]props.RichRun{{Text: "x"}}, &entity.Cell{Width: 50}, nil)

		assert.Equal(t, 0.0, got)
	})

	t.Run("returns single line height for a short run", func(t *testing.T) {
		t.Parallel()
		sut, fpdf, _ := richTextProviderSetup(t)
		fpdf.EXPECT().GetStringWidth(mock.AnythingOfType("string")).Return(8.0).Maybe()

		got := sut.(core.RichTextMeasurer).MeasureRichText(
			[]props.RichRun{{Text: "hi", Family: "roboto", Size: 10}},
			&entity.Cell{Width: 50, Height: 20},
			nil,
		)

		assert.InDelta(t, 4.0, got, 0.001)
	})
}

func TestProvider_GetLinesQuantity(t *testing.T) {
	t.Parallel()
	// Arrange
	prop := &props.Text{}

	text := mocks.NewText(t)
	text.EXPECT().GetLinesQuantity("body", prop, 80.0).Return(3).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{Text: text})

	// Act
	got := sut.GetLinesQuantity("body", prop, 80)

	// Assert
	assert.Equal(t, 3, got)
}

func TestProvider_AddText_WithOutline_Bookmarks(t *testing.T) {
	t.Parallel()
	// Arrange
	cell := &entity.Cell{Y: 2}
	prop := &props.Text{Top: 5, Outline: &props.Outline{Level: 1}}

	text := mocks.NewText(t)
	text.EXPECT().Add("body", cell, prop).Once()

	fpdf := newPDF(t)
	fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Once()
	fpdf.EXPECT().Bookmark("body", 1, 7.0).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf, Text: text})

	// Act
	sut.AddText("body", cell, prop)
}

func TestProvider_AddImageFromFile(t *testing.T) {
	t.Parallel()

	t.Run("when image cannot be loaded, should apply error message and record issue", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := &props.Rect{}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("missing.png", extension.Png).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().LoadImage("missing.png", extension.Png).Return(errors.New("anyError2")).Once()

		text := mocks.NewText(t)
		text.EXPECT().Add("could not load image", cell, merror.DefaultErrorText).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{Cache: cache, Text: text})

		// Act
		sut.AddImageFromFile("missing.png", cell, prop)

		// Assert
		issues := sut.(core.RenderIssueProvider).RenderIssues()
		assert.Len(t, issues, 1)
		assert.Equal(t, "image.load", issues[0].Operation)
	})

	t.Run("when image is cached, should add it to the document", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := &props.Rect{}
		imageBytes := []byte{1, 2, 3}
		cfg := &entity.Config{Margins: &entity.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("asset.jpg", extension.Jpg).Return(&entity.Image{Bytes: imageBytes}, nil).Once()

		image := mocks.NewImage(t)
		image.EXPECT().
			Add(&entity.Image{Bytes: imageBytes, Extension: extension.Jpg}, cell, cfg.Margins, prop, extension.Jpg, false).
			Return(nil).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{Cache: cache, Image: image, Cfg: cfg})

		// Act
		sut.AddImageFromFile("asset.jpg", cell, prop)

		// Assert
		assert.Len(t, sut.(core.RenderIssueProvider).RenderIssues(), 0)
	})
}
