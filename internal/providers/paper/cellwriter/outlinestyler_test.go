package cellwriter_test

import (
	"testing"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestOutlineStyler_Apply(t *testing.T) {
	t.Parallel()

	t.Run("outline draws Rect outside cell bounds when OutlineWidth > 0", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetXY().Return(10.0, 5.0).Maybe()
		fpdf.EXPECT().GetLineWidth().Return(0.2).Maybe()
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0).Maybe()
		fpdf.EXPECT().GetFillColor().Return(255, 255, 255).Maybe()
		fpdf.EXPECT().SetLineWidth(0.5).Once()
		fpdf.EXPECT().SetDrawColor(255, 0, 0).Once()
		fpdf.EXPECT().Rect(
			9.75,
			4.75,
			20.5,
			10.5,
			"D",
		).Once()
		fpdf.EXPECT().SetLineWidth(0.2).Once()
		fpdf.EXPECT().SetDrawColor(0, 0, 0).Once()

		// No next node needed — test outline in isolation with nil next.
		sut := cellwriter.NewOutlineStyler(fpdf)

		prop := &props.Cell{
			OutlineWidth:  0.5,
			OutlineStyle:  consts.LineStyleSolid,
			OutlineColor:  &props.Color{Red: 255, Green: 0, Blue: 0},
			OutlineOffset: 0,
		}
		sut.Apply(20, 10, &entity.Config{}, prop)
	})

	t.Run("outline follows rounded cell radius", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetXY().Return(10.0, 5.0).Maybe()
		fpdf.EXPECT().GetLineWidth().Return(0.2).Maybe()
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0).Maybe()
		fpdf.EXPECT().GetFillColor().Return(255, 255, 255).Maybe()
		fpdf.EXPECT().SetLineWidth(0.5).Once()
		fpdf.EXPECT().SetDrawColor(255, 0, 0).Once()
		fpdf.EXPECT().RoundedRect(
			9.75,
			4.75,
			20.5,
			10.5,
			5.25,
			"1234",
			"D",
		).Once()
		fpdf.AssertNotCalled(t, "Rect", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		fpdf.EXPECT().SetLineWidth(0.2).Once()
		fpdf.EXPECT().SetDrawColor(0, 0, 0).Once()

		sut := cellwriter.NewOutlineStyler(fpdf)

		prop := &props.Cell{
			OutlineWidth: 0.5,
			OutlineStyle: consts.LineStyleSolid,
			OutlineColor: &props.Color{Red: 255, Green: 0, Blue: 0},
			BorderRadius: 999,
		}
		sut.Apply(20, 10, &entity.Config{}, prop)
	})

	t.Run("compact full-round outline draws ellipse primitive", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetXY().Return(0.0, 0.0).Maybe()
		fpdf.EXPECT().GetLineWidth().Return(0.2).Maybe()
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0).Maybe()
		fpdf.EXPECT().GetFillColor().Return(255, 255, 255).Maybe()
		fpdf.EXPECT().SetLineWidth(0.5).Once()
		fpdf.EXPECT().SetDrawColor(255, 0, 0).Once()
		fpdf.EXPECT().Ellipse(5.0, 5.0, 5.25, 5.25, 0.0, "D").Once()
		fpdf.AssertNotCalled(t, "RoundedRect", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		fpdf.EXPECT().SetLineWidth(0.2).Once()
		fpdf.EXPECT().SetDrawColor(0, 0, 0).Once()

		sut := cellwriter.NewOutlineStyler(fpdf)

		prop := &props.Cell{
			OutlineWidth: 0.5,
			OutlineStyle: consts.LineStyleSolid,
			OutlineColor: &props.Color{Red: 255, Green: 0, Blue: 0},
			BorderRadius: 999,
		}
		sut.Apply(10, 10, &entity.Config{}, prop)
	})

	t.Run("no Rect call when OutlineWidth is zero", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.AssertNotCalled(t, "Rect", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		sut := cellwriter.NewOutlineStyler(fpdf)
		sut.Apply(20, 10, &entity.Config{}, &props.Cell{})
	})

	t.Run("dashed outline still draws stroked path", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().GetXY().Return(10.0, 5.0).Maybe()
		fpdf.EXPECT().GetLineWidth().Return(0.2).Maybe()
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0).Maybe()
		fpdf.EXPECT().GetFillColor().Return(255, 255, 255).Maybe()
		fpdf.EXPECT().SetLineWidth(0.5).Once()
		fpdf.EXPECT().SetDrawColor(255, 0, 0).Once()
		fpdf.EXPECT().SetDashPattern([]float64{1, 1}, 0.0).Once()
		fpdf.EXPECT().Rect(9.75, 4.75, 20.5, 10.5, "D").Once()
		fpdf.EXPECT().SetDashPattern([]float64{1, 0}, 0.0).Once()
		fpdf.EXPECT().SetLineWidth(0.2).Once()
		fpdf.EXPECT().SetDrawColor(0, 0, 0).Once()

		sut := cellwriter.NewOutlineStyler(fpdf)

		prop := &props.Cell{
			OutlineWidth: 0.5,
			OutlineStyle: consts.LineStyleDashed,
			OutlineColor: &props.Color{Red: 255, Green: 0, Blue: 0},
		}
		sut.Apply(20, 10, &entity.Config{}, prop)
	})
}
