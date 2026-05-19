package translate

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// blockContainer wraps multiple child Rows under a single background+border+padding box.
// Mirrors flexCellContent's structure but adds styled fill/border painting on Render.
type blockContainer struct {
	rows          []core.Row
	style         *props.Cell
	paddingTop    float64
	paddingRight  float64
	paddingBottom float64
	paddingLeft   float64
	config        *entity.Config
	cachedHeight  float64
}

// SetConfig propagates the config to every child row.
func (b *blockContainer) SetConfig(config *entity.Config) {
	b.config = config
	for _, r := range b.rows {
		r.SetConfig(config)
	}
	// Invalidate the height cache when config changes.
	b.cachedHeight = 0
}

// GetStructure returns a "container" structure with all child row structures attached.
func (b *blockContainer) GetStructure() *node.Node[core.Structure] {
	details := map[string]any{"rows": len(b.rows)}
	if b.style != nil {
		for k, v := range b.style.ToMap() {
			details[k] = v
		}
	}
	str := core.Structure{Type: "container", Details: details}
	n := node.New(str)
	for _, r := range b.rows {
		n.AddNext(r.GetStructure())
	}
	return n
}

// GetHeight returns the sum of child row heights plus top+bottom padding.
func (b *blockContainer) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if b.cachedHeight > 0 {
		return b.cachedHeight
	}
	inner := cell.Copy()
	inner.Width = cell.Width - b.paddingLeft - b.paddingRight
	if inner.Width < 0 {
		inner.Width = 0
	}
	total := b.paddingTop + b.paddingBottom
	for _, r := range b.rows {
		total += r.GetHeight(provider, &inner)
	}
	b.cachedHeight = total
	return total
}

// Render paints the background+border once, then renders each child row offset by padding.
func (b *blockContainer) Render(provider core.Provider, cell *entity.Cell) {
	height := b.GetHeight(provider, cell)
	if cell.Height < height {
		// The outer row was sized to our GetHeight, so cell.Height should match.
		// Use the larger of the two so we paint the full visible area.
		height = cell.Height
	}
	if b.style != nil {
		provider.CreateCol(cell.Width, height, b.config, b.style)
	}
	innerCell := cell.Copy()
	innerCell.X += b.paddingLeft
	innerCell.Y += b.paddingTop
	innerCell.Width = cell.Width - b.paddingLeft - b.paddingRight
	if innerCell.Width < 0 {
		innerCell.Width = 0
	}
	for _, r := range b.rows {
		h := r.GetHeight(provider, &innerCell)
		innerCell.Height = h
		r.Render(provider, innerCell)
		innerCell.Y += h
	}
}

// shouldUseContainer reports whether a container's computed style has anything
// worth painting around its children: background, border on any side, or padding.
func shouldUseContainer(style *css.ComputedStyle) bool {
	if style == nil {
		return false
	}
	if style.BackgroundColor != nil {
		return true
	}
	if style.BorderTopWidth > 0 || style.BorderRightWidth > 0 ||
		style.BorderBottomWidth > 0 || style.BorderLeftWidth > 0 {
		return true
	}
	if style.PaddingTop > 0 || style.PaddingRight > 0 ||
		style.PaddingBottom > 0 || style.PaddingLeft > 0 {
		return true
	}
	return false
}

// buildContainerRow wraps the given child rows into a single Row containing one
// Col containing a blockContainer with the style/padding from the computed style.
func buildContainerRow(style *css.ComputedStyle, childRows []core.Row) core.Row {
	cellStyle := blockCellStyle(style)
	container := &blockContainer{
		rows:          childRows,
		style:         cellStyle,
		paddingTop:    style.PaddingTop,
		paddingRight:  style.PaddingRight,
		paddingBottom: style.PaddingBottom,
		paddingLeft:   style.PaddingLeft,
	}
	c := col.New().Add(container)
	return row.New().Add(c)
}
