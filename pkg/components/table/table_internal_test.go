package table

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestDistributeSpanSurplus(t *testing.T) {
	t.Parallel()

	t.Run("scales existing row heights proportionally", func(t *testing.T) {
		t.Parallel()
		heights := []float64{2, 6}

		distributeSpanSurplus(heights, 4, 8)

		assert.InDelta(t, 3, heights[0], 0.0001)
		assert.InDelta(t, 9, heights[1], 0.0001)
	})

	t.Run("splits surplus evenly when current heights are zero", func(t *testing.T) {
		t.Parallel()
		heights := []float64{0, 0, 0}

		distributeSpanSurplus(heights, 9, 0)

		assert.Equal(t, []float64{3, 3, 3}, heights)
	})
}

func TestPaddedTableCellClampsNegativeInnerSize(t *testing.T) {
	t.Parallel()

	cell := paddedTableCell(1, 2, 3, 4, &props.Cell{
		PaddingTop:    3,
		PaddingRight:  3,
		PaddingBottom: 3,
		PaddingLeft:   3,
	})

	assert.Equal(t, 4.0, cell.X)
	assert.Equal(t, 5.0, cell.Y)
	assert.Zero(t, cell.Width)
	assert.Zero(t, cell.Height)
}

func TestNormalizeColumnWidths(t *testing.T) {
	t.Parallel()

	assert.Nil(t, normalizeColumnWidths(nil, 2))

	got := normalizeColumnWidths([]float64{1, 3}, 2)
	assert.InDeltaSlice(t, []float64{0.25, 0.75}, got, 0.0001)

	got = normalizeColumnWidths([]float64{2}, 3)
	assert.InDeltaSlice(t, []float64{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0}, got, 0.0001)
}
