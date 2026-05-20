package cellwriter_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/internal/providers/gofpdf/cellwriter"
	"github.com/johnfercher/maroto/v2/mocks"
	"github.com/johnfercher/maroto/v2/pkg/consts/linestyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/stretchr/testify/mock"
)

func TestOutlineStyler_Apply(t *testing.T) {
	t.Parallel()

	t.Run("outline draws Rect outside cell bounds when OutlineWidth > 0", func(t *testing.T) {
		t.Parallel()
		fpdf := mocks.NewFpdf(t)
		fpdf.EXPECT().GetXY().Return(10.0, 5.0).Maybe()
		fpdf.EXPECT().GetLineWidth().Return(0.2).Maybe()
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0).Maybe()
		fpdf.EXPECT().SetLineWidth(mock.AnythingOfType("float64")).Maybe()
		fpdf.EXPECT().SetDrawColor(mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Maybe()
		fpdf.EXPECT().Rect(
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"),
			"D",
		).Once()

		// No next node needed — test outline in isolation with nil next.
		sut := cellwriter.NewOutlineStyler(fpdf)

		prop := &props.Cell{
			OutlineWidth:  0.5,
			OutlineStyle:  linestyle.Solid,
			OutlineColor:  &props.Color{Red: 255, Green: 0, Blue: 0},
			OutlineOffset: 0,
		}
		sut.Apply(20, 10, &entity.Config{}, prop)
	})

	t.Run("no Rect call when OutlineWidth is zero", func(t *testing.T) {
		t.Parallel()
		fpdf := mocks.NewFpdf(t)
		fpdf.AssertNotCalled(t, "Rect", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		sut := cellwriter.NewOutlineStyler(fpdf)
		sut.Apply(20, 10, &entity.Config{}, &props.Cell{})
	})
}
