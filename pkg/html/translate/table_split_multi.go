package translate

import (
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

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
	if height <= remainingHeight {
		if threshold <= 0 || remainingHeight-height >= threshold {
			return nil, nil, false
		}
		return r.carryFittingLastRow(provider, remainingHeight, width, threshold)
	}
	if first, rest, split := r.splitFirstRow(provider, remainingHeight, width); split {
		return first, rest, true
	}
	if first, rest, split := r.splitMiddleRow(provider, remainingHeight, width); split {
		return first, rest, true
	}
	if first, rest, split := r.splitBetweenRows(provider, remainingHeight, width); split {
		return first, rest, true
	}
	return nil, r, true
}

func (r *splittableMultiTableRow) carryFittingLastRow(provider core.Provider, remainingHeight, width, threshold float64) (core.Row, core.Row, bool) {
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
	first, rest, split := last.carryWrappedTailToNextPage(provider, remainingHeight-prefixHeight, width, threshold)
	if !split || first == nil || rest == nil {
		return nil, nil, false
	}
	firstRow, firstOK := first.(*splittableTableRow)
	restRow, restOK := rest.(*splittableTableRow)
	if !firstOK || !restOK {
		return nil, nil, false
	}
	firstCells := append(append([][]table.Cell{}, r.cells[:len(r.cells)-1]...), firstRow.cells)
	firstFragment, err := r.rebuild(firstCells)
	if err != nil || firstFragment.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
		return nil, nil, false
	}
	restFragment, err := r.rebuild([][]table.Cell{restRow.cells})
	if err != nil {
		return nil, nil, false
	}
	return firstFragment, restFragment, true
}

func (r *splittableMultiTableRow) splitFirstRow(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	// If even the first row cannot fit, keep its later text and the remaining
	// rows together on the next page instead of moving the whole table.
	firstTable, err := table.New([][]table.Cell{r.cells[0]}, r.opts...)
	if err != nil {
		return nil, nil, false
	}
	firstRow := newSplittableTableRow(firstTable, r.cells[0], r.opts)
	firstRow.SetConfig(r.config)
	first, rest, split := firstRow.SplitAt(provider, remainingHeight, width)
	if !split || first == nil || rest == nil {
		return nil, nil, false
	}
	firstPart, firstOK := first.(*splittableTableRow)
	restPart, restOK := rest.(*splittableTableRow)
	if !firstOK || !restOK {
		return nil, nil, false
	}
	firstFragment, err := r.rebuild([][]table.Cell{firstPart.cells})
	if err != nil || firstFragment.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight+0.001 {
		return nil, nil, false
	}
	restCells := append([][]table.Cell{restPart.cells}, r.cells[1:]...)
	restFragment, err := r.rebuild(restCells)
	if err != nil {
		return nil, nil, false
	}
	return firstFragment, restFragment, true
}

func (r *splittableMultiTableRow) splitMiddleRow(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	for count := len(r.cells) - 1; count > 0; count-- {
		prefix, err := r.rebuild(r.cells[:count])
		if err != nil {
			continue
		}
		prefixHeight := prefix.GetHeight(provider, &entity.Cell{Width: width})
		if prefixHeight >= remainingHeight {
			continue
		}
		partialTable, err := table.New([][]table.Cell{r.cells[count]}, r.opts...)
		if err != nil {
			continue
		}
		partialRow := newSplittableTableRow(partialTable, r.cells[count], r.opts)
		partialRow.SetConfig(r.config)
		// Keep shorter rows intact at a page edge. A tall wrapped row
		// has enough content to leave a useful fragment on both pages.
		// A marked short result can also move alone when its label fits near
		// the page edge.
		tall := tableCellsWrapAtLeastThreeLines(provider, r.cells[count], r.table.ColumnWidths(), width)
		if r.rowFragmentEdge > 0 && !tall {
			tall = tableCellsHaveExplicitLines(r.cells[count])
		}
		rowHeight := partialRow.GetHeight(provider, &entity.Cell{Width: width})
		available := remainingHeight - prefixHeight
		nearCarry := r.rowFragmentEdge > 0 && rowHeight <= available && partialRow.NearBoundaryThreshold() > available-rowHeight
		if !tall && !nearCarry {
			continue
		}
		edge := 0.0
		if tall {
			edge = r.rowFragmentEdge
		}
		partialAvailable := available - edge
		if rowHeight <= partialAvailable && !nearCarry {
			continue
		}
		first, rest, split := partialRow.SplitAt(provider, partialAvailable, width)
		if !split || first == nil || rest == nil {
			continue
		}
		firstRow, firstOK := first.(*splittableTableRow)
		restRow, restOK := rest.(*splittableTableRow)
		if !firstOK || !restOK {
			continue
		}
		firstCells := append(append([][]table.Cell{}, r.cells[:count]...), firstRow.cells)
		firstFragment, err := r.rebuild(firstCells)
		if err != nil || firstFragment.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight-edge+0.001 {
			continue
		}
		restCells := append([][]table.Cell{restRow.cells}, r.cells[count+1:]...)
		restFragment, err := r.rebuild(restCells)
		if err == nil {
			return firstFragment, restFragment, true
		}
	}
	return nil, nil, false
}

func (r *splittableMultiTableRow) splitBetweenRows(provider core.Provider, remainingHeight, width float64) (core.Row, core.Row, bool) {
	for count := len(r.cells) - 1; count > 0; count-- {
		if count == 1 && r.keepFirstRowWithNext {
			continue
		}
		first, err := r.rebuild(r.cells[:count])
		if err != nil || first.GetHeight(provider, &entity.Cell{Width: width}) > remainingHeight {
			continue
		}
		rest, err := r.rebuild(r.cells[count:])
		if err == nil {
			return first, rest, true
		}
	}
	return nil, nil, false
}
