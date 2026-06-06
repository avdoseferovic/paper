package table

import "math"

// WithColumnWidths configures relative column widths. Values are normalized to
// fractions that sum to 1. Non-positive or missing values fall back to the
// average explicit width so partially specified tables remain usable.
func WithColumnWidths(widths []float64) Option {
	return func(t *Table) {
		t.columnWidths = normalizeColumnWidths(widths, t.colCount)
	}
}

func normalizeColumnWidths(widths []float64, colCount int) []float64 {
	if colCount <= 0 || len(widths) == 0 {
		return nil
	}
	out := make([]float64, colCount)
	sum := 0.0
	explicit := 0
	for i := 0; i < colCount && i < len(widths); i++ {
		w := widths[i]
		if w <= 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			continue
		}
		out[i] = w
		sum += w
		explicit++
	}
	if sum <= 0 || explicit == 0 {
		return nil
	}
	fallback := sum / float64(explicit)
	for i := range out {
		if out[i] > 0 {
			continue
		}
		out[i] = fallback
		sum += fallback
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

func (t *Table) columnWidth(totalWidth float64, col int) float64 {
	if t == nil || t.colCount <= 0 || col < 0 || col >= t.colCount {
		return 0
	}
	if len(t.columnWidths) == t.colCount {
		return totalWidth * t.columnWidths[col]
	}
	return totalWidth / float64(t.colCount)
}

func (t *Table) columnSpanWidth(totalWidth float64, startCol, span int) float64 {
	if span <= 0 {
		span = 1
	}
	width := 0.0
	for c := startCol; c < startCol+span && c < t.colCount; c++ {
		width += t.columnWidth(totalWidth, c)
	}
	return width
}

func (t *Table) originColumn(flatIndex int) int {
	for r := range t.grid {
		for c, slot := range t.grid[r] {
			if slot == flatIndex {
				return c
			}
		}
	}
	return 0
}
