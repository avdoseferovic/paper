package table

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestWithBorderCollapseMergesInteriorEdges(t *testing.T) {
	t.Parallel()

	style := &props.Cell{BorderType: border.Full, BorderThickness: 0.5}
	cells := [][]Cell{
		{{Style: style}, {Style: style}},
		{{Style: style}, {Style: style}},
	}
	tbl, err := New(cells, WithBorderCollapse(true))
	require.NoError(t, err)

	topLeft := tbl.collapsedCellStyle(&cells[0][0], 0, 0)
	require.NotNil(t, topLeft)
	assert.Equal(t, 0.0, topLeft.BorderRightThickness, "interior right edge collapses")
	assert.Equal(t, 0.0, topLeft.BorderBottomThickness, "interior bottom edge collapses")
	assert.True(t, topLeft.BorderTopThickness > 0, "exterior top edge stays")
	assert.True(t, topLeft.BorderLeftThickness > 0, "exterior left edge stays")

	bottomRight := tbl.collapsedCellStyle(&cells[1][1], 1, 1)
	require.NotNil(t, bottomRight)
	assert.True(t, bottomRight.BorderRightThickness > 0, "exterior right edge stays")
	assert.True(t, bottomRight.BorderBottomThickness > 0, "exterior bottom edge stays")
}

func TestCollapsedCellStyleDisabledKeepsOriginal(t *testing.T) {
	t.Parallel()

	style := &props.Cell{BorderType: border.Full, BorderThickness: 0.5}
	cells := [][]Cell{{{Style: style}}}
	tbl, err := New(cells)
	require.NoError(t, err)

	got := tbl.collapsedCellStyle(&cells[0][0], 0, 0)
	assert.Equal(t, style, got, "without border-collapse the original style is used as-is")
}
