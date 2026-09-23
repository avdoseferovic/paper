package translate

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
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

func TestTableCellUsesContinuationPaddingOnlyOnLaterFragment(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td>label</td><td data-continuation-padding-top="4.6pt" style="padding-top: 2.2pt">one two three four five six</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	rows[0].SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	row := rows[0].(*splittableTableRow)
	first, rest, split := row.SplitAt(provider, 2, 100)
	require.True(t, split)
	assert.InDelta(t, css.ParseLength("2.2pt", 0), row.cells[1].Style.PaddingTop, 0.001)
	assert.InDelta(t, css.ParseLength("2.2pt", 0), first.(*splittableTableRow).cells[1].Style.PaddingTop, 0.001)
	assert.InDelta(t, css.ParseLength("4.6pt", 0), rest.(*splittableTableRow).cells[1].Style.PaddingTop, 0.001)
}

func TestTableRowKeepsFollowingRowWhenRequested(t *testing.T) {
	doc, err := dom.Parse(`<table style="break-after: avoid"><tr><th>Heading</th></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].(*splittableTableRow).KeepWithNext())
}

func TestMarginWrappedTablePreservesKeepWithNext(t *testing.T) {
	doc, err := dom.Parse(`<table style="margin-left: 5pt; break-after: avoid"><tr><th>Heading</th></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	keep, ok := rows[0].(core.KeepWithNext)
	require.True(t, ok)
	assert.True(t, keep.KeepWithNext())
}

func TestTableCanCarryResultCellAcrossNearBoundary(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td>label</td><td data-carry-content-near-page-end="3.5pt" style="padding-top: 2.2pt" data-continuation-padding-top="4.6pt">answer</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0].(*splittableTableRow)
	row.SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	height := row.GetHeight(provider, &entity.Cell{Width: 100})
	first, rest, split := row.SplitAt(provider, height+css.ParseLength("1pt", 0), 100)
	require.True(t, split)
	assert.Equal(t, "label", tableCellText(first, 0))
	assert.Equal(t, "", tableCellText(first, 1))
	assert.Equal(t, "", tableCellText(rest, 0))
	assert.Equal(t, "answer", tableCellText(rest, 1))
	assert.InDelta(t, css.ParseLength("4.6pt", 0), rest.(*splittableTableRow).cells[1].Style.PaddingTop, 0.001)
}

func TestTableKeepsMultilineCellsTogetherNearBoundary(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td>long label text that wraps over several lines in the available column</td><td data-carry-content-near-page-end="3.5pt">answer</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	row := rows[0].(*splittableTableRow)
	row.SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &widthAwareTableProvider{cursorProvider: &cursorProvider{}}
	height := row.GetHeight(provider, &entity.Cell{Width: 100})
	_, _, split := row.SplitAt(provider, height+css.ParseLength("1pt", 0), 100)
	assert.False(t, split)
}

type widthAwareTableProvider struct{ *cursorProvider }

func (p *widthAwareTableProvider) MeasureRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) float64 {
	var words int
	for _, run := range runs {
		words += len(strings.Fields(run.Text))
	}
	if cell.Width < 1000 && words > 5 {
		return 2 + prop.Top + prop.Bottom
	}
	return 1 + prop.Top + prop.Bottom
}

func TestTablePushesWholeRowWhenPaintedEdgeWillNotFit(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td style="border: 1pt solid black">label</td><td data-carry-content-near-page-end="3.5pt">answer</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	row := rows[0].(*splittableTableRow)
	row.SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	height := row.GetHeight(provider, &entity.Cell{Width: 100})
	first, rest, split := row.SplitAt(provider, height+css.ParseLength("0.2pt", 0), 100)
	require.True(t, split)
	assert.Nil(t, first)
	assert.Equal(t, core.Row(row), rest)
}

func TestTableKeepsForcedBreakCellTogetherNearBoundary(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td>first<br>second</td><td data-carry-content-near-page-end="3.5pt">answer</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	row := rows[0].(*splittableTableRow)
	row.SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	height := row.GetHeight(provider, &entity.Cell{Width: 100})
	_, _, split := row.SplitAt(provider, height+css.ParseLength("1pt", 0), 100)
	assert.False(t, split)
}

func TestTableDoesNotCarryEveryCellNearBoundary(t *testing.T) {
	doc, err := dom.Parse(`<table><tr><td data-carry-content-near-page-end="3.5pt">label</td><td data-carry-content-near-page-end="3.5pt">answer</td></tr></table>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	row := rows[0].(*splittableTableRow)
	row.SetConfig(&entity.Config{MaxGridSize: 12, DefaultFont: &props.Font{}})
	provider := &paragraphMeasureProvider{cursorProvider: &cursorProvider{}}
	height := row.GetHeight(provider, &entity.Cell{Width: 100})
	_, _, split := row.SplitAt(provider, height+css.ParseLength("1pt", 0), 100)
	assert.False(t, split)
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
