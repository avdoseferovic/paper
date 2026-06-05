package core

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// ShapeProvider is a narrow optional capability interface for components that
// need to draw primitive geometric shapes (currently used by htmllist's
// DecimalCircle marker). Components detect support via a type assertion and
// fall back to text-only rendering when unavailable — see the same pattern
// used for RichTextProvider in pkg/components/htmllist/htmllist.go.
type ShapeProvider interface {
	// DrawFilledCircle draws a filled circle inscribed in the given cell using
	// fill.FillColor (or BlackColor when nil). The cell's center and width
	// dimensions determine the circle's position and diameter.
	DrawFilledCircle(cell *entity.Cell, fill *props.Color)
}

// PositionProvider lets composite components (blockContainer, flexCellContent)
// reset the underlying pen to a known X/Y before invoking a sub-component's
// Render. Without this, cellwriter chain nodes that use GetXY() as the cell
// origin (perSideBorder, borderRadius) draw at the wrong position when the
// pen has drifted due to prior CellFormat/Ln calls.
type PositionProvider interface {
	SetCursor(x, y float64)
}
