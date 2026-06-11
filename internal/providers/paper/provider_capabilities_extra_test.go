package paper_test

import (
	"math"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestProvider_WithCharSpacing(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := gofpdf.New(&gofpdf.Dependencies{})
	calls := 0

	// Act
	sut.(core.CharSpacingProvider).WithCharSpacing(2.5, func() { calls++ })

	// Assert — no-op wrapper must still run fn exactly once.
	assert.Equal(t, 1, calls)
}

func TestProvider_RegisterFont(t *testing.T) {
	t.Parallel()
	// Arrange
	fontBytes := []byte{1, 2, 3}

	fpdf := newPDF(t)
	fpdf.On("AddUTF8FontFromBytes", "custom", "B", fontBytes).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

	// Act
	sut.(core.LateFontProvider).RegisterFont("custom", "B", fontBytes)
}

func TestProvider_AddLink(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.On("AddLink").Return(7).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

	// Act
	linkID := sut.(core.LinkProvider).AddLink()

	// Assert
	assert.Equal(t, 7, linkID)
}

func TestProvider_SetLink(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().SetLink(3, 11.5, 2).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

	// Act
	sut.(core.LinkProvider).SetLink(3, 11.5, 2)
}

func TestProvider_Link(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().Link(1.0, 2.0, 3.0, 4.0, 9).Once()

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

	// Act
	sut.(core.LinkProvider).Link(1, 2, 3, 4, 9)
}

func TestProvider_WithAlpha_NaNIsTreatedAsOpaque(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().SetAlpha(1.0, "Normal").Times(2)

	sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})
	called := false

	// Act
	sut.(core.AlphaProvider).WithAlpha(math.NaN(), func() { called = true })

	// Assert
	assert.True(t, called)
}

func watermarkFontMock(t *testing.T) *mocks.Font {
	t.Helper()
	font := mocks.NewFont(t)
	font.EXPECT().GetColor().Return(&props.Color{Red: 1, Green: 2, Blue: 3}).Once()
	font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
	font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Maybe()
	return font
}

func TestProvider_AddWatermark_RendersRotatedText(t *testing.T) {
	t.Parallel()
	// Arrange
	font := watermarkFontMock(t)

	fpdf := newPDF(t)
	fpdf.EXPECT().GetStringWidth("DRAFT").Return(50.0).Once()
	fpdf.EXPECT().GetMargins().Return(5.0, 10.0, 0.0, 0.0).Times(2)
	fpdf.EXPECT().SetAlpha(props.DefaultWatermarkAlpha, "Normal").Once()
	fpdf.EXPECT().SetAlpha(1.0, "Normal").Once()
	fpdf.EXPECT().TransformBegin().Once()
	// Rotation pivot: cell center + page margins → (50+5, 50+10).
	fpdf.EXPECT().TransformRotate(props.DefaultWatermarkAngle, 55.0, 60.0).Once()
	// x = centerX - width/2 + left = 50 - 25 + 5.
	fpdf.EXPECT().Text(30.0, mock.AnythingOfType("float64"), "DRAFT").Once()
	fpdf.EXPECT().TransformEnd().Once()

	text := gofpdf.NewText(fpdf, mocks.NewMath(t), font)
	sut := gofpdf.New(&gofpdf.Dependencies{
		PDF:  fpdf,
		Text: text,
		Font: font,
		Cfg:  &entity.Config{},
	})

	// Act
	sut.(core.WatermarkProvider).AddWatermark(
		&entity.Cell{X: 0, Y: 0, Width: 100, Height: 100},
		&props.Watermark{Text: "DRAFT"},
	)
}

func TestProvider_AddWatermark_ScalesDownWideText(t *testing.T) {
	t.Parallel()
	// Arrange
	font := mocks.NewFont(t)
	font.EXPECT().GetColor().Return(&props.Color{}).Once()
	font.EXPECT().SetFont(mock.AnythingOfType("string"), mock.AnythingOfType("fontstyle.Type"), mock.AnythingOfType("float64")).Maybe()
	// Custom watermark color → set once, restored once.
	font.EXPECT().SetColor(mock.AnythingOfType("*props.Color")).Times(2)

	fpdf := newPDF(t)
	// Text is far wider than the 10x10mm cell diagonal → size is rescaled and
	// the string re-measured a second time.
	fpdf.EXPECT().GetStringWidth("WIDE WATERMARK").Return(200.0).Times(2)
	fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Times(2)
	fpdf.EXPECT().SetAlpha(0.5, "Normal").Once()
	fpdf.EXPECT().SetAlpha(1.0, "Normal").Once()
	fpdf.EXPECT().TransformBegin().Once()
	fpdf.EXPECT().TransformRotate(30.0, 5.0, 5.0).Once()
	fpdf.EXPECT().Text(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), "WIDE WATERMARK").Once()
	fpdf.EXPECT().TransformEnd().Once()

	text := gofpdf.NewText(fpdf, mocks.NewMath(t), font)
	sut := gofpdf.New(&gofpdf.Dependencies{
		PDF:  fpdf,
		Text: text,
		Font: font,
		Cfg:  &entity.Config{},
	})

	// Act
	sut.(core.WatermarkProvider).AddWatermark(
		&entity.Cell{X: 0, Y: 0, Width: 10, Height: 10},
		&props.Watermark{Text: "WIDE WATERMARK", Alpha: 0.5, Angle: 30, Color: &props.Color{Red: 200}},
	)
}

func TestProvider_DrawFilledCircle_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("nil cell is a no-op", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.ShapeProvider).DrawFilledCircle(nil, &props.Color{Red: 1})

		fpdf.AssertNotCalled(t, "Circle")
	})

	t.Run("zero-sized cell is a no-op", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.ShapeProvider).DrawFilledCircle(&entity.Cell{Width: 0, Height: 5}, nil)
		sut.(core.ShapeProvider).DrawFilledCircle(&entity.Cell{Width: 5, Height: 0}, nil)

		fpdf.AssertNotCalled(t, "Circle")
	})

	t.Run("nil fill defaults to black and radius uses smaller dimension", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetFillColor().Return(9, 9, 9).Once()
		fpdf.EXPECT().SetFillColor(0, 0, 0).Once()
		fpdf.EXPECT().GetMargins().Return(1.0, 2.0, 0.0, 0.0).Once()
		// Cell 10x4 → radius = 2 (height/2), cx = 0+5+1, cy = 0+2+2.
		fpdf.EXPECT().Circle(6.0, 4.0, 2.0, "F").Once()
		fpdf.EXPECT().SetFillColor(9, 9, 9).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.ShapeProvider).DrawFilledCircle(&entity.Cell{X: 0, Y: 0, Width: 10, Height: 4}, nil)
	})
}
