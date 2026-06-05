package cellwriter_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/internal/providers/paper/cellwriter"
	"github.com/johnfercher/maroto/v2/mocks"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewPerSideBorderStyler(t *testing.T) {
	t.Parallel()
	sut := cellwriter.NewPerSideBorderStyler(mocks.NewFpdf(t))
	assert.NotNil(t, sut)
}

func TestPerSideBorderStyler_Apply(t *testing.T) {
	t.Parallel()
	const w, h = 100.0, 200.0
	config := &entity.Config{}

	t.Run("when no per-side borders set, should pass through to next", func(t *testing.T) {
		t.Parallel()
		// Arrange: prop with legacy BorderType only — PerSideBorderStyler must not intercept
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		fpdf := mocks.NewFpdf(t)
		// Line must NOT be called when no per-side borders
		sut := cellwriter.NewPerSideBorderStyler(fpdf)
		sut.SetNext(next)

		prop := &props.Cell{}
		sut.Apply(w, h, config, prop)

		fpdf.AssertNumberOfCalls(t, "Line", 0)
	})

	t.Run("when only top border set, should draw one Line using GetXY cell origin", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		const cellX, cellY = 15.0, 30.0
		origLineWidth := 0.2
		fpdf := mocks.NewFpdf(t)
		fpdf.EXPECT().GetLineWidth().Return(origLineWidth)
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0)
		fpdf.EXPECT().GetXY().Return(cellX, cellY)
		// Expect exactly one Line call for the top side with real cell-relative coords
		fpdf.EXPECT().SetLineWidth(1.0)
		fpdf.EXPECT().SetDrawColor(255, 0, 0)
		fpdf.EXPECT().Line(cellX, cellY, cellX+w, cellY)
		// Restore original state
		fpdf.EXPECT().SetLineWidth(origLineWidth)
		fpdf.EXPECT().SetDrawColor(0, 0, 0)

		sut := cellwriter.NewPerSideBorderStyler(fpdf)
		sut.SetNext(next)

		prop := &props.Cell{
			BorderTopThickness: 1.0,
			BorderTopColor:     &props.Color{Red: 255},
		}
		sut.Apply(w, h, config, prop)

		fpdf.AssertNumberOfCalls(t, "Line", 1)
	})

	t.Run("when all four sides set with different thicknesses, should draw four Line calls using GetXY", func(t *testing.T) {
		t.Parallel()
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		origLineWidth := 0.2
		fpdf := mocks.NewFpdf(t)
		fpdf.EXPECT().GetLineWidth().Return(origLineWidth)
		fpdf.EXPECT().GetDrawColor().Return(0, 0, 0)
		fpdf.EXPECT().GetXY().Return(0.0, 0.0)
		fpdf.EXPECT().SetLineWidth(mock.AnythingOfType("float64")).Maybe()
		// SetDrawColor called once per side that has a color + once for restore
		fpdf.EXPECT().SetDrawColor(mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Maybe()
		fpdf.EXPECT().Line(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Times(4)

		sut := cellwriter.NewPerSideBorderStyler(fpdf)
		sut.SetNext(next)

		prop := &props.Cell{
			BorderTopThickness:    0.5,
			BorderTopColor:        &props.Color{Red: 255},
			BorderRightThickness:  1.0,
			BorderRightColor:      &props.Color{Green: 255},
			BorderBottomThickness: 1.5,
			BorderBottomColor:     &props.Color{Blue: 255},
			BorderLeftThickness:   2.0,
			BorderLeftColor:       &props.Color{Red: 128, Green: 128},
		}
		sut.Apply(w, h, config, prop)

		fpdf.AssertNumberOfCalls(t, "Line", 4)
	})

	t.Run("legacy BorderType still works after PerSideBorderStyler (regression)", func(t *testing.T) {
		t.Parallel()
		// When no per-side borders, PerSideBorderStyler passes through cleanly and
		// the next styler (which eventually calls CellFormat) handles legacy borders.
		next := mocks.NewCellWriter(t)
		next.EXPECT().Apply(w, h, config, mock.AnythingOfType("*props.Cell"))

		fpdf := mocks.NewFpdf(t)

		sut := cellwriter.NewPerSideBorderStyler(fpdf)
		sut.SetNext(next)

		prop := &props.Cell{
			// Legacy border — no per-side fields
			BorderThickness: 0.6,
		}
		sut.Apply(w, h, config, prop)

		fpdf.AssertNumberOfCalls(t, "Line", 0)       // raw Line NOT called
		fpdf.AssertNumberOfCalls(t, "CellFormat", 0) // CellFormat handled by next
	})
}
