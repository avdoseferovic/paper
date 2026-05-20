package translate

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
)

// flexCellContent is a core.Component that stacks multiple rows vertically
// inside a flex item's column. Used for non-leaf flex items so headings and
// paragraphs render with their own formatting instead of being flattened.
type flexCellContent struct {
	rows []core.Row
}

func newFlexCellContent(rows []core.Row) core.Component {
	return &flexCellContent{rows: rows}
}

// SetConfig propagates the config to every child row.
func (f *flexCellContent) SetConfig(config *entity.Config) {
	for _, r := range f.rows {
		r.SetConfig(config)
	}
}

// GetStructure returns a single structure node with all child row structures attached.
func (f *flexCellContent) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "flex_cell",
		Details: map[string]any{"rows": len(f.rows)},
	}
	n := node.New(str)
	for _, r := range f.rows {
		n.AddNext(r.GetStructure())
	}
	return n
}

// GetHeight sums child row heights (rows are full-width inside the flex col).
func (f *flexCellContent) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	total := 0.0
	for _, r := range f.rows {
		total += r.GetHeight(provider, cell)
	}
	return total
}

// Render draws each row sequentially, advancing the Y cursor.
// Before each sub-row the gofpdf pen is reset to the sub-row's origin so
// cellwriter chain nodes that rely on GetXY (perSideBorder, borderRadius)
// draw at the right position even after CellFormat/Ln has drifted the pen.
func (f *flexCellContent) Render(provider core.Provider, cell *entity.Cell) {
	pp, _ := provider.(core.PositionProvider)
	innerCell := cell.Copy()
	for _, r := range f.rows {
		h := r.GetHeight(provider, &innerCell)
		innerCell.Height = h
		if pp != nil {
			pp.SetCursor(innerCell.X, innerCell.Y)
		}
		r.Render(provider, innerCell)
		innerCell.Y += h
	}
}
