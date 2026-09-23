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
	table                *table.Table
	cells                [][]table.Cell
	opts                 []table.Option
	config               *entity.Config
	keepFirstRowWithNext bool
	rowFragmentEdge      float64
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

func tableCellsWrapAtLeastThreeLines(provider core.Provider, cells []table.Cell, columnWidths []float64, width float64) bool {
	cellWidths, ok := tableCellWidths(cells, columnWidths, width)
	if !ok {
		return false
	}
	for index, cell := range cells {
		rt, ok := cell.Content.(*richtext.RichText)
		if !ok {
			continue
		}
		cellWidth := cellWidths[index]
		if cell.Style != nil {
			cellWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
		}
		if cellWidth <= 0 {
			continue
		}
		lineHeight := rt.GetHeight(provider, &entity.Cell{Width: 10000})
		if lineHeight > 0 && rt.GetHeight(provider, &entity.Cell{Width: cellWidth}) >= 2.5*lineHeight {
			return true
		}
	}
	return false
}

func tableCellsHaveExplicitLines(cells []table.Cell) bool {
	for _, cell := range cells {
		if contentHasThreeExplicitLines(cell.Content) {
			return true
		}
	}
	return false
}

func contentHasThreeExplicitLines(content core.Component) bool {
	switch value := content.(type) {
	case *richtext.RichText:
		breaks := 0
		for _, run := range value.Runs() {
			if run.ForceBreak {
				breaks++
			} else {
				breaks += strings.Count(run.Text, "\n")
			}
		}
		return breaks >= 2
	case *nestedTableComponent:
		return tableCellsHaveExplicitLines(value.row.cells)
	default:
		return false
	}
}

func tableCellWidths(cells []table.Cell, columnWidths []float64, width float64) ([]float64, bool) {
	columnCount := 0
	for _, cell := range cells {
		if cell.Rowspan > 1 {
			return nil, false
		}
		columnCount += max(1, cell.Colspan)
	}
	if columnCount == 0 || len(columnWidths) != 0 && len(columnWidths) != columnCount {
		return nil, false
	}
	widths := make([]float64, len(cells))
	column := 0
	for index, cell := range cells {
		for range max(1, cell.Colspan) {
			if len(columnWidths) == 0 {
				widths[index] += width / float64(columnCount)
			} else {
				widths[index] += width * columnWidths[column]
			}
			column++
		}
	}
	return widths, true
}

func (r *splittableMultiTableRow) rebuild(cells [][]table.Cell) (*splittableMultiTableRow, error) {
	tbl, err := table.New(cells, r.opts...)
	if err != nil {
		return nil, err
	}
	fragment := newSplittableMultiTableRow(tbl, cells, r.opts)
	fragment.rowFragmentEdge = r.rowFragmentEdge
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
	cellWidths, ok := tableCellWidths(r.cells, r.table.ColumnWidths(), width)
	if !ok {
		return nil, r, true
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	anyContinuation := false
	for index, cell := range r.cells {
		first, rest, split := splitTableCellText(provider, cell, remainingHeight, cellWidths[index], r.config)
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
	cellWidths, ok := tableCellWidths(r.cells, r.table.ColumnWidths(), width)
	if !ok {
		return nil, nil, false
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	carried := false
	retained := false
	for index, cell := range r.cells {
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
		innerWidth := cellWidths[index]
		if cell.Style != nil {
			innerWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
		}
		if innerWidth <= 0 || rt.GetHeight(provider, &entity.Cell{Width: innerWidth}) <= rt.GetHeight(provider, &entity.Cell{Width: 10000})+0.001 {
			return nil, nil, false
		}
		first, rest, split := splitTableCellText(provider, cell, remainingHeight-threshold, cellWidths[index], r.config)
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
	cellWidths, ok := tableCellWidths(r.cells, r.table.ColumnWidths(), width)
	if !ok {
		return nil, nil, false
	}
	firstCells := make([]table.Cell, len(r.cells))
	restCells := make([]table.Cell, len(r.cells))
	carried := false
	retained := false
	for index, cell := range r.cells {
		if cell.Content != nil {
			cellWidth := cellWidths[index]
			if cell.Style != nil {
				cellWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
			}
			if cellWidth <= 0 || !contentIsSingleLine(provider, cell.Content, cellWidth) {
				return nil, nil, false
			}
		}
		firstCells[index], restCells[index] = cell, cell
		if cell.CarryContentNearPageEnd > 0 && cell.Content != nil {
			firstCells[index].Content = nil
			firstCells[index].Height = 0
			setContinuationPadding(&restCells[index], cell.ContinuationPaddingTop)
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

func contentIsSingleLine(provider core.Provider, content core.Component, width float64) bool {
	switch value := content.(type) {
	case *richtext.RichText:
		for _, run := range value.Runs() {
			if run.ForceBreak || strings.Contains(run.Text, "\n") {
				return false
			}
		}
		return value.GetHeight(provider, &entity.Cell{Width: width}) <= value.GetHeight(provider, &entity.Cell{Width: 10000})+0.001
	case *nestedTableComponent:
		cellWidths, ok := tableCellWidths(value.row.cells, value.ColumnWidths(), width)
		if !ok {
			return false
		}
		for index, inner := range value.row.cells {
			if inner.Content == nil {
				continue
			}
			innerWidth := cellWidths[index]
			if inner.Style != nil {
				innerWidth -= inner.Style.PaddingLeft + inner.Style.PaddingRight
			}
			if innerWidth <= 0 || !contentIsSingleLine(provider, inner.Content, innerWidth) {
				return false
			}
		}
		return true
	default:
		return false
	}
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
	innerWidth := width
	innerHeight := available
	if cell.Style != nil {
		innerWidth -= cell.Style.PaddingLeft + cell.Style.PaddingRight
		innerHeight -= cell.Style.PaddingTop + cell.Style.PaddingBottom
	}
	if innerWidth <= 0 || innerHeight <= 0 {
		return first, cell, false
	}
	if nested, ok := cell.Content.(*nestedTableComponent); ok {
		firstContent, restContent, split := nested.splitAt(provider, innerHeight, innerWidth)
		if !split {
			return first, cell, false
		}
		first.Content = firstContent
		rest.Content = restContent
		setContinuationPadding(&rest, cell.ContinuationPaddingTop)
		return first, rest, true
	}
	rt, ok := cell.Content.(*richtext.RichText)
	if !ok {
		return first, rest, false
	}
	return splitRichTextCell(provider, cell, first, rest, rt, innerWidth, innerHeight, config)
}

func splitRichTextCell(provider core.Provider, cell, first, rest table.Cell, rt *richtext.RichText, innerWidth, innerHeight float64, config *entity.Config) (table.Cell, table.Cell, bool) {
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
	setContinuationPadding(&rest, cell.ContinuationPaddingTop)
	return first, rest, true
}

func setContinuationPadding(cell *table.Cell, padding float64) {
	if padding <= 0 {
		return
	}
	style := &props.Cell{}
	if cell.Style != nil {
		cloned := *cell.Style
		style = &cloned
	}
	style.PaddingTop = padding
	cell.Style = style
}
