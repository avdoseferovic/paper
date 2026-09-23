package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestSingleTableRowSplitsTextAtPageBoundary(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td>one two three four five six</td><td>yes</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	sp, ok := rows[0].(core.Splittable)
	require.True(t, ok)
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	rows[0].SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	first, rest, split := sp.SplitAt(provider, 2, 100)
	require.True(t, split)
	require.NotNil(t, first)
	require.NotNil(t, rest)
	assert.True(t, first.GetHeight(provider, &entity.Cell{Width: 100}) <= 2.001)
	assert.Equal(t, "one two three four", tableCellText(first, 0))
	assert.Equal(t, "yes", tableCellText(first, 1))
	assert.Equal(t, "five six", tableCellText(rest, 0))
	assert.Equal(t, "", tableCellText(rest, 1))
}

func TestStyledTableCellContinuesWithStyle(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td><strong>one two three four five six</strong></td><td>yes</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	rows[0].SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	first, rest, split := rows[0].(core.Splittable).SplitAt(provider, 2, 100)
	require.True(t, split)
	for _, fragment := range []core.Row{first, rest} {
		runs := fragment.(*splittableTableRow).cells[0].Content.(*richtext.RichText).Runs()
		require.Len(t, runs, 1)
		assert.Equal(t, fontstyle.Bold, runs[0].Style)
	}
}

func tableCellText(r core.Row, index int) string {
	fragment := r.(*splittableTableRow)
	rt, ok := fragment.cells[index].Content.(*richtext.RichText)
	if !ok {
		return ""
	}
	var value string
	for _, run := range rt.Runs() {
		value += run.Text
	}
	return value
}
