package table

// normaliseSpans ensures all cells have Colspan >= 1 and Rowspan >= 1.
func normaliseSpans(cells [][]Cell) {
	for r := range cells {
		for c := range cells[r] {
			if cells[r][c].Colspan < 1 {
				cells[r][c].Colspan = 1
			}
			if cells[r][c].Rowspan < 1 {
				cells[r][c].Rowspan = 1
			}
		}
	}
}

// deriveColCount scans all rows to find the true column count accounting for colspans.
func deriveColCount(cells [][]Cell) (int, error) {
	maxCols := 0
	for _, row := range cells {
		total := 0
		for _, cell := range row {
			total += cell.Colspan
		}
		if total > maxCols {
			maxCols = total
		}
	}
	if maxCols == 0 {
		return 0, ErrTableEmpty
	}
	return maxCols, nil
}

// buildGrid fills an occupation matrix: grid[r][c] = flat index of source cell, -1 if spanned.
func buildGrid(cells [][]Cell, rowCount, colCount int) ([][]int, error) {
	occ := make([][]int, rowCount)
	for r := range rowCount {
		occ[r] = make([]int, colCount)
		for c := range colCount {
			occ[r][c] = -1
		}
	}

	flat := 0
	for r, row := range cells {
		col := 0
		for _, cell := range row {
			for col < colCount && occ[r][col] != -1 {
				col++
			}
			if col >= colCount {
				return nil, ErrTableSpanOverlap
			}
			err := markOccupied(occ, r, col, cell, flat, rowCount, colCount)
			if err != nil {
				return nil, err
			}
			col += cell.Colspan
			flat++
		}
	}
	return occ, nil
}

// spannedMarker indicates a grid slot occupied by a cell whose origin is elsewhere.
// Distinct from empty (-1) so Render can skip rendering at non-origin slots.
const spannedMarker = -2

func markOccupied(occ [][]int, startR, startC int, cell Cell, flatIdx, rowCount, colCount int) error {
	for dr := range cell.Rowspan {
		if startR+dr >= rowCount {
			break
		}
		for dc := range cell.Colspan {
			if startC+dc >= colCount {
				break
			}
			if dr == 0 && dc == 0 {
				occ[startR][startC] = flatIdx
				continue
			}
			if occ[startR+dr][startC+dc] != -1 {
				return ErrTableSpanOverlap
			}
			occ[startR+dr][startC+dc] = spannedMarker
		}
	}
	return nil
}
