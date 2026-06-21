package paper_test

import (
	"bytes"
	"image/png"
	"io"
	"testing"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	pdfcore "github.com/avdoseferovic/paper/internal/pdf"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestGradientRenderer_DrawGradient(t *testing.T) {
	t.Parallel()

	t.Run("horizontal linear gradient registers one-pixel-high strip and draws at full height", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(15.0, 10.0, 15.0, 10.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(
			mock.AnythingOfType("string"),
			mock.Anything,
			mock.Anything,
		).Run(func(_ string, _ pdfcore.ImageOptions, r io.Reader) {
			data, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("read registered image: %v", err)
			}
			cfg, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("decode registered image: %v", err)
			}
			if cfg.Height != 1 {
				t.Fatalf("horizontal gradient raster height = %d, want 1", cfg.Height)
			}
			if cfg.Width <= 1 {
				t.Fatalf("horizontal gradient raster width = %d, want > 1", cfg.Width)
			}
		}).Return(nil).Once()
		pdf.EXPECT().Image(
			mock.AnythingOfType("string"),
			15.0,
			10.0,
			50.0,
			20.0,
			false,
			"PNG",
			0,
			"",
		).Once()

		g := &props.Gradient{
			Kind:     props.GradientLinear,
			AngleDeg: 90,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 255, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
	})

	t.Run("horizontal linear gradient reuses strip image across different heights", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Once()
		pdf.EXPECT().Image(mock.AnythingOfType("string"), 0.0, 0.0, 60.0, 10.0, false, "PNG", 0, "").Once()
		pdf.EXPECT().Image(mock.AnythingOfType("string"), 0.0, 0.0, 60.0, 20.0, false, "PNG", 0, "").Once()

		g := &props.Gradient{
			Kind:     props.GradientLinear,
			AngleDeg: 90,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 10, Green: 20, Blue: 30}, Position: 0},
				{Color: props.Color{Red: 40, Green: 50, Blue: 60}, Position: 0.5},
				{Color: props.Color{Red: 70, Green: 80, Blue: 90}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 0, Y: 0, Width: 60, Height: 10}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 60, 10)
		gr.DrawGradient(cell, g, 60, 20)
	})

	t.Run("identical gradient on second call reuses imgName (no second RegisterImageOptionsReader)", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(0.0, 0.0, 0.0, 0.0).Maybe()
		// RegisterImageOptionsReader called exactly ONCE even though DrawGradient is called twice
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Once()
		pdf.EXPECT().Image(mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything, false, "PNG", 0, "").Times(2)

		g := &props.Gradient{
			Kind:     props.GradientLinear,
			AngleDeg: 45,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 0, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 255, Green: 255, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 0, Y: 0, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
		gr.DrawGradient(cell, g, 50, 20)
	})

	t.Run("raster fallback Image x/y includes margin offsets", func(t *testing.T) {
		t.Parallel()
		pdf := newPDF(t)
		pdf.EXPECT().GetMargins().Return(15.0, 10.0, 15.0, 10.0).Maybe()
		pdf.EXPECT().RegisterImageOptionsReader(mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil).Maybe()
		// x should be cell.X + leftMargin = 5 + 15 = 20
		// y should be cell.Y + topMargin  = 3 + 10 = 13
		pdf.EXPECT().Image(mock.AnythingOfType("string"), 20.0, 13.0, 50.0, 20.0, false, "PNG", 0, "").Once()

		g := &props.Gradient{
			Kind: props.GradientLinear, AngleDeg: 45,
			Stops: []props.GradientStop{
				{Color: props.Color{Red: 255, Green: 0, Blue: 0}, Position: 0},
				{Color: props.Color{Red: 0, Green: 0, Blue: 255}, Position: 1},
			},
		}
		cell := &entity.Cell{X: 5, Y: 3, Width: 50, Height: 20}
		gr := gofpdf.NewGradientRenderer(pdf)
		gr.DrawGradient(cell, g, 50, 20)
	})
}
