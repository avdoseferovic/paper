package translate

import (
	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
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

type marginBox struct {
	child        core.Component
	marginTop    float64
	marginRight  float64
	marginBottom float64
	marginLeft   float64
}

func (m *marginBox) SetConfig(config *entity.Config) {
	if m.child != nil {
		m.child.SetConfig(config)
	}
}

func (m *marginBox) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "margin_box",
		Details: map[string]any{
			"margin_top":    m.marginTop,
			"margin_right":  m.marginRight,
			"margin_bottom": m.marginBottom,
			"margin_left":   m.marginLeft,
		},
	}
	n := node.New(str)
	if m.child != nil {
		n.AddNext(m.child.GetStructure())
	}
	return n
}

func (m *marginBox) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if m.child == nil {
		return m.marginTop + m.marginBottom
	}
	inner := m.innerCell(cell)
	return m.child.GetHeight(provider, &inner) + m.marginTop + m.marginBottom
}

func (m *marginBox) Render(provider core.Provider, cell *entity.Cell) {
	if m.child == nil {
		return
	}
	inner := m.innerCell(cell)
	m.child.Render(provider, &inner)
}

func (m *marginBox) innerCell(cell *entity.Cell) entity.Cell {
	inner := cell.Copy()
	inner.X += m.marginLeft
	inner.Y += m.marginTop
	inner.Width -= m.marginLeft + m.marginRight
	inner.Height -= m.marginTop + m.marginBottom
	if inner.Width < 0 {
		inner.Width = 0
	}
	if inner.Height < 0 {
		inner.Height = 0
	}
	return inner
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
// The pen is reset before painting the background AND before each child row so cellwriter
// chain nodes that rely on GetXY draw at the right position.
func (b *blockContainer) Render(provider core.Provider, cell *entity.Cell) {
	height := b.GetHeight(provider, cell)
	if cell.Height < height {
		height = cell.Height
	}
	pp, _ := provider.(core.PositionProvider)
	if b.style != nil {
		if pp != nil {
			pp.SetCursor(cell.X, cell.Y)
		}
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
		if pp != nil {
			pp.SetCursor(innerCell.X, innerCell.Y)
		}
		r.Render(provider, innerCell)
		innerCell.Y += h
	}
	if pp != nil {
		pp.SetCursor(cell.X, cell.Y)
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

// buildContainerRow wraps the given child rows into a single splittableContainerRow.
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
	return newSplittableContainerRow(container)
}

// splittableContainerRow wraps a blockContainer as a core.Row that also
// implements core.Splittable so maroto.addRow() can split it across pages.
type splittableContainerRow struct {
	container *blockContainer
	config    *entity.Config
}

func newSplittableContainerRow(c *blockContainer) *splittableContainerRow {
	return &splittableContainerRow{container: c}
}

func (s *splittableContainerRow) SetConfig(cfg *entity.Config) {
	s.config = cfg
	s.container.SetConfig(cfg)
}

func (s *splittableContainerRow) GetStructure() *node.Node[core.Structure] {
	// Build row → col → container structure to match the original row.New().Add(col.New().Add(container)) layout.
	rowNode := node.New(core.Structure{Type: "row"})
	colNode := node.New(core.Structure{Type: "col"})
	colNode.AddNext(s.container.GetStructure())
	rowNode.AddNext(colNode)
	return rowNode
}

func (s *splittableContainerRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if cell == nil {
		return 0
	}
	return s.container.GetHeight(provider, cell)
}

func (s *splittableContainerRow) Render(provider core.Provider, cell entity.Cell) {
	cell.Height = s.container.GetHeight(provider, &cell)
	s.container.Render(provider, &cell)
	// Advance the gofpdf cursor so the next row renders below this one.
	provider.CreateRow(cell.Height)
}

func (s *splittableContainerRow) Add(_ ...core.Col) core.Row       { return s }
func (s *splittableContainerRow) WithStyle(_ *props.Cell) core.Row { return s }

// GetColumns returns a single col wrapping the blockContainer to match the
// structure that callers (e.g. existing tests) expect.
func (s *splittableContainerRow) GetColumns() []core.Col {
	return []core.Col{col.New().Add(s.container)}
}

// SplitAt implements core.Splittable. It splits the container's child rows at
// the point where cumulative row heights would exceed remainingHeight.
// Returns (nil, self, true) when no child rows fit (push whole container to
// next page). Returns (self, nil, false) when the container fits entirely.
func (s *splittableContainerRow) SplitAt(provider core.Provider, remainingHeight float64) (first, rest core.Row, didSplit bool) {
	if s.container == nil {
		return nil, nil, false
	}

	// Use a dummy cell for height measurement (width only matters for word wrap).
	dummyCell := &entity.Cell{Width: 10000, Height: 10000}

	totalHeight := s.container.GetHeight(provider, dummyCell)
	if totalHeight <= remainingHeight {
		return nil, nil, false // fits — no split needed
	}

	// Greedy split: accumulate rows until they no longer fit.
	padding := s.container.paddingTop + s.container.paddingBottom
	available := remainingHeight - padding
	if available < 0 {
		available = 0
	}

	var firstRows, restRows []core.Row
	cumHeight := 0.0
	splitDone := false
	for _, r := range s.container.rows {
		rh := r.GetHeight(provider, dummyCell)
		if !splitDone && cumHeight+rh <= available+0.001 {
			firstRows = append(firstRows, r)
			cumHeight += rh
		} else {
			restRows = append(restRows, r)
			splitDone = true
		}
	}

	if len(firstRows) == 0 {
		// Nothing fits on the current page — push the whole container to next page.
		return nil, s, true
	}

	firstContainer := &blockContainer{
		rows:          firstRows,
		style:         s.container.style,
		paddingTop:    s.container.paddingTop,
		paddingRight:  s.container.paddingRight,
		paddingBottom: 0, // flat bottom at split point
		paddingLeft:   s.container.paddingLeft,
		config:        s.container.config,
	}

	var restRow core.Row
	if len(restRows) > 0 {
		restContainer := &blockContainer{
			rows:          restRows,
			style:         s.container.style,
			paddingTop:    0, // flat top at split point
			paddingRight:  s.container.paddingRight,
			paddingBottom: s.container.paddingBottom,
			paddingLeft:   s.container.paddingLeft,
			config:        s.container.config,
		}
		restRow = newSplittableContainerRow(restContainer)
	}

	return newSplittableContainerRow(firstContainer), restRow, true
}
