package cellwriter

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/props"
)

// sidePDFStub implements perSideBorderPDF, recording calls for behavior assertions.
type sidePDFStub struct {
	lines  [][4]float64
	dashes [][]float64
	colors [][3]int
	widths []float64
}

func (s *sidePDFStub) GetDrawColor() (int, int, int) { return 0, 0, 0 }
func (s *sidePDFStub) GetLineWidth() float64         { return 0.2 }
func (s *sidePDFStub) GetXY() (float64, float64)     { return 0, 0 }

func (s *sidePDFStub) Line(x1, y1, x2, y2 float64) {
	s.lines = append(s.lines, [4]float64{x1, y1, x2, y2})
}

func (s *sidePDFStub) SetDashPattern(dashArray []float64, _ float64) {
	s.dashes = append(s.dashes, dashArray)
}

func (s *sidePDFStub) SetDrawColor(r, g, b int) {
	s.colors = append(s.colors, [3]int{r, g, b})
}

func (s *sidePDFStub) SetLineWidth(width float64) {
	s.widths = append(s.widths, width)
}

func newSideStyler() *perSideBorderStyler {
	return &perSideBorderStyler{stylerTemplate: stylerTemplate{name: "perSideBorderStyler"}}
}

func TestPerSideBorderStyler_DrawSide(t *testing.T) {
	t.Parallel()

	t.Run("zero thickness draws nothing", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}

		newSideStyler().drawSide(stub, 0, &props.Color{Red: 1}, consts.LineStyleSolid, &props.Cell{}, 0, 0, 10, 0)

		assert.Len(t, stub.lines, 0)
		assert.Len(t, stub.widths, 0)
	})

	t.Run("solid side uses side color without dash pattern", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}

		newSideStyler().drawSide(stub, 0.5, &props.Color{Red: 1, Green: 2, Blue: 3}, consts.LineStyleSolid, &props.Cell{}, 0, 0, 10, 0)

		require.Len(t, stub.lines, 1)
		assert.Equal(t, [4]float64{0, 0, 10, 0}, stub.lines[0])
		require.Len(t, stub.widths, 1)
		assert.Equal(t, 0.5, stub.widths[0])
		require.Len(t, stub.colors, 1)
		assert.Equal(t, [3]int{1, 2, 3}, stub.colors[0])
		assert.Len(t, stub.dashes, 0)
	})

	t.Run("dashed side sets and resets the dash pattern", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}

		newSideStyler().drawSide(stub, 1, nil, consts.LineStyleDashed, &props.Cell{}, 0, 0, 10, 0)

		require.Len(t, stub.dashes, 2)
		assert.Equal(t, []float64{1, 1}, stub.dashes[0])
		assert.Equal(t, []float64{1, 0}, stub.dashes[1])
		assert.Len(t, stub.lines, 1)
	})

	t.Run("dotted side sets and resets the dash pattern", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}

		newSideStyler().drawSide(stub, 1, nil, consts.LineStyleDotted, &props.Cell{}, 0, 0, 0, 10)

		require.Len(t, stub.dashes, 2)
		assert.Equal(t, []float64{0.4, 0.4}, stub.dashes[0])
		assert.Equal(t, []float64{1, 0}, stub.dashes[1])
	})

	t.Run("nil side color falls back to the uniform border color", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}
		prop := &props.Cell{BorderColor: &props.Color{Red: 4, Green: 5, Blue: 6}}

		newSideStyler().drawSide(stub, 1, nil, consts.LineStyleSolid, prop, 0, 0, 10, 0)

		require.Len(t, stub.colors, 1)
		assert.Equal(t, [3]int{4, 5, 6}, stub.colors[0])
	})

	t.Run("no colors leaves the draw color untouched", func(t *testing.T) {
		t.Parallel()
		stub := &sidePDFStub{}

		newSideStyler().drawSide(stub, 1, nil, consts.LineStyleSolid, &props.Cell{}, 0, 0, 10, 0)

		assert.Len(t, stub.colors, 0)
		assert.Len(t, stub.lines, 1)
	})
}
