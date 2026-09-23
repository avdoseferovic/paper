package translate

import (
	"strings"
	"unicode"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// splittableMultiTableRow preserves a table's shared column grid while
// allowing marked rows to continue at a page boundary.
type splittableMultiTableRow struct {
	core.Row
	table  *table.Table
	cells  [][]table.Cell
	opts   []table.Option
	config *entity.Config
}

func newSplittableMultiTableRow(tbl *table.Table, cells [][]table.Cell, opts []table.Option) *splittableMultiTableRow {
	return &splittableMultiTableRow{Row: row.New().Add(col.New().Add(tbl)), table: tbl, cells: cells, opts: opts}
}

func (r *splittableMultiTableRow) SetConfig(config *entity.Config) {
	r.config = config
	r.Row.SetConfig(config)
}

func (r *splittableMultiTableRow) NearBoundaryThreshold() float64 {
	if len(r.cells) == 0 {
		return 0
	}
	threshold := 0.0
	for _, cell := range r.cells[len(r.cells)-1] {
		threshold = max(threshold, cell.CarryContentNearPageEnd)
	}
	return threshold
}

func (r *splittableMultiTableRow) SplitAt(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	if len(r.cells) == 0 {
		return nil, nil, false
	}
	if len(r.cells) == 1 {
		last := newSplittableTableRow(r.table, r.cells[0], r.opts)
		last.SetConfig(r.config)
		return last.SplitAt(provider, remainingHeight, width)
	}
	height := r.GetHeight(provider, &entity.Cell{Width: width})
	for _, cells := range r.cells {
		for _, cell := range cells {
			if cell.Rowspan > 1 {
				if height > remainingHeight {
					return nil, r, true
				}
				return nil, nil, false
			}
		}
	}
	threshold := r.NearBoundaryThreshold()
	if height <= remainingHeight && (threshold <= 0 || remainingHeight-height >= threshold) {
		return nil, nil, false
	}
	prefix, err := r.rebuild(r.cells[:len(r.cells)-1])
	if err != nil {
		return nil, nil, false
	}
	prefixHeight := prefix.GetHeight(provider, &entity.Cell{Width: width})
	lastTable, err := table.New([][]table.Cell{r.cells[len(r.cells)-1]}, r.opts...)
	if err != nil {
		return nil, nil, false
	}
	last := newSplittableTableRow(lastTable, r.cells[len(r.cells)-1], r.opts)
	last.SetConfig(r.config)
	if height <= remainingHeight {
		first, rest, split := last.carryWrappedTailToNextPage(provider, remainingHeight-prefixHeight, width, threshold)
		if !split || first == nil || rest == nil {
			return nil, nil, false
		}
		firstCells := append(append([][]table.Cell{}, r.cells[:len(r.cells)-1]...), first.(*splittableTableRow).cells)
		firstFragment, err := r.rebuild(firstCells)
		if err != nil || firstFragment.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
			return nil, nil, false
		}
		restFragment, err := r.rebuild([][]table.Cell{rest.(*splittableTableRow).cells})
		if err != nil {
			return nil, nil, false
		}
		return firstFragment, restFragment, true
	}
	for count := len(r.cells) - 1; count > 0; count-- {
		first, err := r.rebuild(r.cells[:count])
		if err != nil || first.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight {
			continue
		}
		rest, err := r.rebuild(r.cells[count:])
		if err == nil {
			return first, rest, true
		}
	}
	return nil, r, true
}

func (r *splittableMultiTableRow) rebuild(cells [][]table.Cell) (*splittableMultiTableRow, error) {
	tbl, err := table.New(cells, r.opts...)
	if err != nil {
		return nil, err
	}
	fragment := newSplittableMultiTableRow(tbl, cells, r.opts)
	if r.config != nil {
		fragment.SetConfig(r.config)
	}
	return fragment, nil
}

// splittableTableRow lets text in a single HTML table row continue on the
// next page while keeping the cells in their original columns.
type splittableTableRow struct {
	core.Row
	table            *table.Table
	cells            []table.Cell
	opts             []table.Option
	config           *entity.Config
	keepWithNext     bool
	pageBreakPreview bool
}

func (r *splittableTableRow) KeepWithNext() bool { return r.keepWithNext }

func (r *splittableTableRow) PageBreakPreview(provider core.Provider, width float64) core.Row {
	if !r.pageBreakPreview {
		return nil
	}
	height := r.GetHeight(provider, &entity.Cell{Width: width})
	cells := make([]table.Cell, len(r.cells))
	for i, cell := range r.cells {
		cells[i] = cell
		cells[i].Content = nil
		cells[i].Height = height
	}
	preview, err := r.rebuild(cells)
	if err != nil {
		return nil
	}
	preview.keepWithNext = false
	return preview
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

func (r *splittableTableRow) NearBoundaryThreshold() float64 {
	threshold := 0.0
	for _, cell := range r.cells {
		threshold = max(threshold, cell.CarryContentNearPageEnd)
	}
	return threshold
}

func (r *splittableTableRow) SplitAt(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	if height := r.GetHeight(provider, &entity.Cell{Width: width}); height <= remainingHeight {
		if threshold := r.NearBoundaryThreshold(); threshold > 0 && remainingHeight-height < threshold {
			if first, rest, split := r.carryContentToNextPage(provider, remainingHeight, width); split {
				return first, rest, true
			}
			return r.carryWrappedTailToNextPage(provider, remainingHeight, width, threshold)
		}
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

// carryWrappedTailToNextPage keeps the final wrapped result line from sitting
// against the bottom edge when the row otherwise fits with very little slack.
func (r *splittableTableRow) carryWrappedTailToNextPage(provider core.Provider, remainingHeight, width, threshold float64) (core.Row, core.Row, bool) {
	if _, ok := provider.(core.RichTextMeasurer); !ok {
		return nil, nil, false
	}
	columnWidths := r.table.ColumnWidths()
	if len(columnWidths) != 0 && len(columnWidths) != len(r.cells) {
		return nil, nil, false
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	carried := false
	retained := false
	for index, cell := range r.cells {
		if cell.Colspan > 1 || cell.Rowspan > 1 {
			return nil, nil, false
		}
		firstCells[index], restCells[index] = cell, cell
		restCells[index].Content = nil
		restCells[index].Height = 0
		if cell.CarryContentNearPageEnd <= 0 || cell.Content == nil {
			retained = retained || cell.Content != nil
			continue
		}
		rt, ok := cell.Content.(*richtext.RichText)
		if !ok {
			return nil, nil, false
		}
		for _, run := range rt.Runs() {
			if run.ForceBreak || strings.Contains(run.Text, "\n") {
				return nil, nil, false
			}
		}
		cellWidth := width / float64(len(r.cells))
		if len(columnWidths) > 0 {
			cellWidth = width * columnWidths[index]
		}
		innerWidth := cellWidth
		if cell.Style != nil {
			innerWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
		}
		if innerWidth <= 0 || rt.GetHeight(provider, &entity.Cell{Width: innerWidth}) <= rt.GetHeight(provider, &entity.Cell{Width: 10000})+0.001 {
			return nil, nil, false
		}
		first, rest, split := splitTableCellText(provider, cell, remainingHeight-threshold, cellWidth, r.config)
		if !split || rest.Content == nil {
			return nil, nil, false
		}
		firstCells[index], restCells[index] = first, rest
		carried = true
	}
	if !carried || !retained {
		return nil, nil, false
	}
	first, err := r.rebuild(firstCells)
	if err != nil || first.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
		return nil, nil, false
	}
	rest, err := r.rebuild(restCells)
	if err != nil {
		return nil, nil, false
	}
	return first, rest, true
}

func (r *splittableTableRow) carryContentToNextPage(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	columnWidths := r.table.ColumnWidths()
	if len(columnWidths) != 0 && len(columnWidths) != len(r.cells) {
		return nil, nil, false
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	carried := false
	retained := false
	for index, cell := range r.cells {
		if cell.Colspan > 1 || cell.Rowspan > 1 {
			return nil, nil, false
		}
		if cell.Content != nil {
			rt, ok := cell.Content.(*richtext.RichText)
			if !ok {
				return nil, nil, false
			}
			for _, run := range rt.Runs() {
				if run.ForceBreak || strings.Contains(run.Text, "\n") {
					return nil, nil, false
				}
			}
			cellWidth := width / float64(len(r.cells))
			if len(columnWidths) > 0 {
				cellWidth = width * columnWidths[index]
			}
			if cell.Style != nil {
				cellWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
			}
			if cellWidth <= 0 || rt.GetHeight(provider, &entity.Cell{Width: cellWidth}) > rt.GetHeight(provider, &entity.Cell{Width: 10000})+0.001 {
				return nil, nil, false
			}
		}
		firstCells[index], restCells[index] = cell, cell
		if cell.CarryContentNearPageEnd > 0 && cell.Content != nil {
			firstCells[index].Content = nil
			firstCells[index].Height = 0
			if cell.ContinuationPaddingTop > 0 && cell.Style != nil {
				style := *cell.Style
				style.PaddingTop = cell.ContinuationPaddingTop
				restCells[index].Style = &style
			}
			carried = true
		} else {
			restCells[index].Content = nil
			restCells[index].Height = 0
			retained = retained || cell.Content != nil
		}
	}
	if !carried || !retained {
		return nil, nil, false
	}
	if remainingHeight-r.GetHeight(provider, &entity.Cell{Width: width}) < r.PageBoundaryAllowance() {
		return nil, r, true
	}
	first, err := r.rebuild(firstCells)
	if err != nil || first.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
		return nil, nil, false
	}
	rest, err := r.rebuild(restCells)
	if err != nil {
		return nil, nil, false
	}
	return first, rest, true
}

func (r *splittableTableRow) rebuild(cells []table.Cell) (*splittableTableRow, error) {
	tbl, err := table.New([][]table.Cell{cells}, r.opts...)
	if err != nil {
		return nil, err
	}
	fragment := newSplittableTableRow(tbl, cells, r.opts)
	fragment.keepWithNext = r.keepWithNext
	fragment.pageBreakPreview = r.pageBreakPreview
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
	type boundary struct {
		run, offset int
		explicit    bool
	}
	var explicitCuts, wordCuts []boundary
	for i, run := range runs {
		if run.ForceBreak {
			if i > 0 && i+1 < len(runs) {
				explicitCuts = append(explicitCuts, boundary{run: i, explicit: true})
			}
			continue
		}
		if run.Image != nil {
			continue
		}
		for offset, char := range run.Text {
			if unicode.IsSpace(char) {
				wordCuts = append(wordCuts, boundary{run: i, offset: offset})
			}
		}
	}
	splitAt := func(cut boundary) ([]props.RichRun, []props.RichRun) {
		if cut.explicit {
			return props.CloneRichRuns(runs[:cut.run]), props.CloneRichRuns(runs[cut.run+1:])
		}
		return splitParagraphRuns(runs, cut.run, cut.offset)
	}
	latestFitting := func(cuts []boundary) (boundary, bool) {
		low, high := 0, len(cuts)
		for low < high {
			middle := (low + high) / 2
			firstRuns, restRuns := splitAt(cuts[middle])
			if len(firstRuns) > 0 && len(restRuns) > 0 && measured(firstRuns) <= innerHeight {
				low = middle + 1
			} else {
				high = middle
			}
		}
		if low == 0 {
			return boundary{}, false
		}
		return cuts[low-1], true
	}
	cut, ok := latestFitting(explicitCuts)
	if !ok {
		cut, ok = latestFitting(wordCuts)
	}
	if !ok {
		return first, cell, false
	}
	firstRuns, restRuns := splitAt(cut)
	first.Content = rt.CloneWithRuns(firstRuns)
	rest.Content = rt.CloneWithRuns(restRuns)
	if cell.ContinuationPaddingTop > 0 && rest.Style != nil {
		style := *rest.Style
		style.PaddingTop = cell.ContinuationPaddingTop
		rest.Style = &style
	}
	return first, rest, true
}
