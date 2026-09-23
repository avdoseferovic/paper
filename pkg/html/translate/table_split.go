package translate

import (
	"unicode"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// splittableTableRow lets text in a single HTML table row continue on the
// next page while keeping the cells in their original columns.
type splittableTableRow struct {
	core.Row
	table  *table.Table
	cells  []table.Cell
	opts   []table.Option
	config *entity.Config
}

func newSplittableTableRow(tbl *table.Table, cells []table.Cell, opts []table.Option) *splittableTableRow {
	return &splittableTableRow{
		Row:   row.New().Add(col.New().Add(tbl)),
		table: tbl, cells: cells, opts: opts,
	}
}

func (r *splittableTableRow) SetConfig(config *entity.Config) {
	r.config = config
	r.Row.SetConfig(config)
}

func (r *splittableTableRow) PageBoundaryAllowance() float64 {
	allowance := 0.0
	for _, cell := range r.cells {
		if cell.Style == nil {
			continue
		}
		border := cell.Style.BorderTopThickness + cell.Style.BorderBottomThickness
		if border == 0 {
			border = 2 * cell.Style.BorderThickness
		}
		allowance = max(allowance, border)
	}
	return allowance
}

func (r *splittableTableRow) SplitAt(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	if r.GetHeight(provider, &entity.Cell{Width: width}) <= remainingHeight {
		return nil, nil, false
	}
	if _, ok := provider.(core.RichTextMeasurer); !ok || remainingHeight <= 0 || len(r.cells) == 0 {
		return nil, r, true
	}
	columnWidths := r.table.ColumnWidths()
	if len(columnWidths) != 0 && len(columnWidths) != len(r.cells) {
		return nil, r, true
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	anyContinuation := false
	for index, cell := range r.cells {
		if cell.Colspan > 1 || cell.Rowspan > 1 {
			return nil, r, true
		}
		cellWidth := width / float64(len(r.cells))
		if len(columnWidths) > 0 {
			cellWidth = width * columnWidths[index]
		}
		first, rest, split := splitTableCellText(provider, cell, remainingHeight, cellWidth, r.config)
		if !split && rest.Content != nil {
			return nil, r, true
		}
		firstCells[index], restCells[index] = first, rest
		anyContinuation = anyContinuation || split
	}
	if !anyContinuation {
		return nil, r, true
	}
	first, err := r.rebuild(firstCells)
	if err != nil {
		return nil, r, true
	}
	rest, err := r.rebuild(restCells)
	if err != nil {
		return nil, r, true
	}
	if first.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
		return nil, r, true
	}
	return first, rest, true
}

func (r *splittableTableRow) rebuild(cells []table.Cell) (*splittableTableRow, error) {
	tbl, err := table.New([][]table.Cell{cells}, r.opts...)
	if err != nil {
		return nil, err
	}
	fragment := newSplittableTableRow(tbl, cells, r.opts)
	if r.config != nil {
		fragment.SetConfig(r.config)
	}
	return fragment, nil
}

func splitTableCellText(provider core.Provider, cell table.Cell, available, width float64, config *entity.Config) (table.Cell, table.Cell, bool) {
	first, rest := cell, cell
	rest.Content = nil
	rest.Height = 0
	rt, ok := cell.Content.(*richtext.RichText)
	if !ok {
		return first, rest, false
	}
	innerWidth := width
	innerHeight := available
	if cell.Style != nil {
		innerWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
		innerHeight -= cell.Style.PaddingTop + cell.Style.PaddingBottom
	}
	if innerWidth <= 0 || innerHeight <= 0 {
		return first, cell, false
	}
	measured := func(runs []props.RichRun) float64 {
		candidate := rt.CloneWithRuns(runs)
		if config != nil {
			candidate.SetConfig(config)
		}
		return candidate.GetHeight(provider, &entity.Cell{Width: innerWidth})
	}
	runs := rt.Runs()
	if measured(runs) <= innerHeight {
		return first, rest, false
	}
	type boundary struct{ run, offset int }
	var cuts []boundary
	for i, run := range runs {
		if run.Image != nil || run.ForceBreak {
			continue
		}
		for offset, char := range run.Text {
			if unicode.IsSpace(char) {
				cuts = append(cuts, boundary{i, offset})
			}
		}
	}
	low, high := 0, len(cuts)
	for low < high {
		middle := (low + high) / 2
		firstRuns, restRuns := splitParagraphRuns(runs, cuts[middle].run, cuts[middle].offset)
		if len(firstRuns) > 0 && len(restRuns) > 0 && measured(firstRuns) <= innerHeight {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == 0 {
		return first, cell, false
	}
	firstRuns, restRuns := splitParagraphRuns(runs, cuts[low-1].run, cuts[low-1].offset)
	first.Content = rt.CloneWithRuns(firstRuns)
	rest.Content = rt.CloneWithRuns(restRuns)
	return first, rest, true
}
