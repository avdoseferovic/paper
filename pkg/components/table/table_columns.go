package table

import (
	"math"
	"slices"
	"strings"

	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// WithColumnWidths configures relative column widths. Values are normalized to
// fractions that sum to 1. Non-positive or missing values fall back to the
// average explicit width so partially specified tables remain usable.
func WithColumnWidths(widths []float64) Option {
	return func(t *Table) {
		t.columnWidths = normalizeColumnWidths(widths, t.colCount)
	}
}

// WithWidth sets a preferred rendered table width in millimetres. When the
// parent cell is wider, the table uses this width; when the parent is narrower,
// it clamps to the parent width.
func WithWidth(width float64) Option {
	return func(t *Table) {
		if width > 0 && !math.IsNaN(width) && !math.IsInf(width, 0) {
			t.preferredWidth = width
		}
	}
}

// WithAlign positions a preferred-width table inside a wider parent cell.
func WithAlign(align string) Option {
	return func(t *Table) {
		switch strings.ToLower(strings.TrimSpace(align)) {
		case "center", "right":
			t.align = strings.ToLower(strings.TrimSpace(align))
		default:
			t.align = ""
		}
	}
}

// WithBorderCollapse merges adjacent cell borders, matching CSS
// border-collapse: collapse. Interior cell edges are drawn once instead of
// twice; border-spacing is ignored by CSS in this mode.
func WithBorderCollapse(collapse bool) Option {
	return func(t *Table) {
		t.borderCollapse = collapse
	}
}

// WithBorderSpacing reserves horizontal and vertical spacing between table
// cells, matching CSS border-spacing for separate-border tables.
func WithBorderSpacing(x, y float64) Option {
	return func(t *Table) {
		if x > 0 && !math.IsNaN(x) && !math.IsInf(x, 0) {
			t.borderSpacingX = x
		}
		if y > 0 && !math.IsNaN(y) && !math.IsInf(y, 0) {
			t.borderSpacingY = y
		}
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
	totalWidth = t.contentColumnsWidth(totalWidth)
	if len(t.columnWidths) == t.colCount {
		return totalWidth * t.columnWidths[col]
	}
	return totalWidth / float64(t.colCount)
}

func (t *Table) contentColumnsWidth(totalWidth float64) float64 {
	if t == nil || t.colCount <= 1 || t.borderSpacingX <= 0 {
		return totalWidth
	}
	spacing := t.borderSpacingX * float64(t.colCount-1)
	if spacing >= totalWidth {
		return 0
	}
	return totalWidth - spacing
}

func (t *Table) tableCell(cell *entity.Cell) entity.Cell {
	if cell == nil {
		return entity.Cell{}
	}
	out := cell.Copy()
	if t == nil || t.preferredWidth <= 0 || t.preferredWidth >= out.Width {
		return out
	}
	delta := out.Width - t.preferredWidth
	switch t.align {
	case "right":
		out.X += delta
	case "center":
		out.X += delta / 2
	}
	out.Width = t.preferredWidth
	return out
}

func (t *Table) columnSpanWidth(totalWidth float64, startCol, span int) float64 {
	if span <= 0 {
		span = 1
	}
	width := 0.0
	endCol := min(startCol+span, t.colCount)
	for c := startCol; c < endCol; c++ {
		width += t.columnWidth(totalWidth, c)
	}
	if t != nil && t.borderSpacingX > 0 && endCol-startCol > 1 {
		width += t.borderSpacingX * float64(endCol-startCol-1)
	}
	return width
}

func (t *Table) originColumn(flatIndex int) int {
	for r := range t.grid {
		if c := slices.Index(t.grid[r], flatIndex); c >= 0 {
			return c
		}
	}

	return 0
}
