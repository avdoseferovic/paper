package paper_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestProvider_Bookmark(t *testing.T) {
	t.Parallel()

	t.Run("forwards title and level with top margin added to y", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(10.0, 15.0, 10.0, 20.0).Once()
		fpdf.EXPECT().Bookmark("Chapter 1", 2, 45.0).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.OutlineProvider).Bookmark("Chapter 1", 2, 30.0)
	})

	t.Run("clamps negative level to zero", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Once()
		fpdf.EXPECT().Bookmark("Top", 0, 5.0).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.OutlineProvider).Bookmark("Top", -3, 5.0)
	})
}

func TestProvider_AddWatermark(t *testing.T) {
	t.Parallel()

	t.Run("is a no-op without a richtext renderer", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})
		cell := entity.NewRootCell(210, 297, entity.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10})

		sut.(core.WatermarkProvider).AddWatermark(&cell, &props.Watermark{Text: "DRAFT"})
		fpdf.AssertNotCalled(t, "TransformBegin")
	})

	t.Run("is a no-op for nil cell, nil prop, and empty text", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})
		cell := entity.NewRootCell(210, 297, entity.Margins{})

		wp := sut.(core.WatermarkProvider)
		wp.AddWatermark(nil, &props.Watermark{Text: "X"})
		wp.AddWatermark(&cell, nil)
		wp.AddWatermark(&cell, &props.Watermark{})
		fpdf.AssertNotCalled(t, "TransformBegin")
	})
}

func TestWatermark_ToMap_ShouldExposeText(t *testing.T) {
	t.Parallel()

	m := (&props.Watermark{Text: "DRAFT"}).ToMap(nil)

	assert.Equal(t, "DRAFT", m["config_watermark"])
}
