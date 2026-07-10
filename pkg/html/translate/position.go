package translate

import (
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type offsetRow struct {
	child   core.Row
	offsetX float64
	offsetY float64
}

type absoluteRow struct {
	child          core.Row
	x              float64
	y              float64
	right          float64
	width          float64
	rightAligned   bool
	position       string
	relativeToCell bool
	zIndex         int
	zIndexSet      bool
}

func (r *offsetRow) SetConfig(config *entity.Config) {
	if r.child != nil {
		r.child.SetConfig(config)
	}
}

func (r *absoluteRow) SetConfig(config *entity.Config) {
	if r.child != nil {
		r.child.SetConfig(config)
	}
}

func (r *offsetRow) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "offset_row",
		Details: map[string]any{
			"offset_x": r.offsetX,
			"offset_y": r.offsetY,
		},
	}
	n := node.New(str)
	if r.child != nil {
		n.AddNext(r.child.GetStructure())
	}
	return n
}

func (r *absoluteRow) GetStructure() *node.Node[core.Structure] {
	position := r.position
	if position == "" {
		position = positionAbsolute
	}
	details := map[string]any{
		"position":      position,
		"x":             r.x,
		"y":             r.y,
		"right":         r.right,
		"width":         r.width,
		"right_aligned": r.rightAligned,
	}
	if r.relativeToCell {
		details["relative_to_cell"] = true
	}
	if r.zIndexSet {
		details["z_index"] = r.zIndex
	}
	str := core.Structure{
		Type:    "absolute_row",
		Details: details,
	}
	n := node.New(str)
	if r.child != nil {
		n.AddNext(r.child.GetStructure())
	}
	return n
}

func (r *offsetRow) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if r.child == nil {
		return 0
	}
	return r.child.GetHeight(provider, cell)
}

func (r *absoluteRow) GetHeight(core.Provider, *entity.Cell) float64 {
	return 0
}

func (r *offsetRow) Render(provider core.Provider, cell entity.Cell) {
	if r.child == nil {
		return
	}
	shifted := cell.Copy()
	shifted.X += r.offsetX
	shifted.Y += r.offsetY
	r.child.Render(provider, shifted)
}

func (r *absoluteRow) Render(provider core.Provider, cell entity.Cell) {
	if r.child == nil {
		return
	}
	target := cell.Copy()
	target.X = r.absoluteX(cell)
	if r.rightAligned {
		target.X = r.rightAlignedX(cell)
	}
	target.Y = r.absoluteY(cell)
	if r.width > 0 {
		target.Width = r.width
	} else {
		target.Width = r.remainingWidth(cell, target.X)
	}
	target.Height = r.child.GetHeight(provider, &target)
	r.child.Render(provider, target)
	if pp, ok := provider.(core.PositionProvider); ok {
		pp.SetCursor(cell.X, cell.Y)
	}
}

func (r *offsetRow) Add(cols ...core.Col) core.Row {
	if r.child != nil {
		r.child = r.child.Add(cols...)
	}
	return r
}

func (r *absoluteRow) Add(cols ...core.Col) core.Row {
	if r.child != nil {
		r.child = r.child.Add(cols...)
	}
	return r
}

func (r *offsetRow) WithStyle(style *props.Cell) core.Row {
	if r.child != nil {
		r.child = r.child.WithStyle(style)
	}
	return r
}

func (r *absoluteRow) WithStyle(style *props.Cell) core.Row {
	if r.child != nil {
		r.child = r.child.WithStyle(style)
	}
	return r
}

func (r *offsetRow) GetColumns() []core.Col {
	if r.child == nil {
		return nil
	}
	return r.child.GetColumns()
}

func (r *absoluteRow) GetColumns() []core.Col {
	if r.child == nil {
		return nil
	}
	return r.child.GetColumns()
}

func (r *offsetRow) SplitAt(provider core.Provider, remainingHeight float64, width float64) (core.Row, core.Row, bool) {
	splittable, ok := r.child.(core.Splittable)
	if !ok {
		return nil, nil, false
	}
	first, rest, didSplit := splittable.SplitAt(provider, remainingHeight, width)
	if !didSplit {
		return nil, nil, false
	}
	return r.wrapSplit(first), r.wrapSplit(rest), true
}

func (r *offsetRow) wrapSplit(row core.Row) core.Row {
	if row == nil {
		return nil
	}
	return &offsetRow{
		child:   row,
		offsetX: r.offsetX,
		offsetY: r.offsetY,
	}
}

func (r *absoluteRow) SplitAt(core.Provider, float64, float64) (core.Row, core.Row, bool) {
	return nil, nil, false
}

func (r *absoluteRow) IsFixedOverlay() bool {
	return r.position == positionFixed
}

func (r *absoluteRow) RenderLayer() int {
	if r.zIndexSet && r.zIndex < 0 {
		return r.zIndex
	}
	if r.zIndexSet && r.zIndex > 0 {
		return r.zIndex + 1
	}
	return 1
}

func (r *absoluteRow) absoluteX(cell entity.Cell) float64 {
	if r.relativeToCell {
		return cell.X + r.x
	}
	return r.x
}

func (r *absoluteRow) absoluteY(cell entity.Cell) float64 {
	if r.relativeToCell {
		return cell.Y + r.y
	}
	return r.y
}

func (r *absoluteRow) rightAlignedX(cell entity.Cell) float64 {
	x := cell.Width - r.right - r.width
	if r.relativeToCell {
		x += cell.X
	}
	minX := 0.0
	if r.relativeToCell {
		minX = cell.X
	}
	if x < minX {
		return minX
	}
	return x
}

func (r *absoluteRow) remainingWidth(cell entity.Cell, targetX float64) float64 {
	startX := 0.0
	if r.relativeToCell {
		startX = cell.X
	}
	used := targetX - startX
	if used < 0 {
		used = 0
	}
	width := cell.Width - used
	if width < 0 {
		return 0
	}
	return width
}

func applyRelativePosition(style *css.ComputedStyle, rows []core.Row) []core.Row {
	offsetX, offsetY, ok := relativePositionOffset(style)
	if !ok || len(rows) == 0 {
		return rows
	}
	out := make([]core.Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, &offsetRow{
			child:   r,
			offsetX: offsetX,
			offsetY: offsetY,
		})
	}
	return out
}

func relativePositionOffset(style *css.ComputedStyle) (float64, float64, bool) {
	if style == nil || style.Position != positionRelative {
		return 0, 0, false
	}
	offsetX := style.Left
	if offsetX == 0 && style.Right != 0 {
		offsetX = -style.Right
	}
	offsetY := style.Top
	if offsetY == 0 && style.Bottom != 0 {
		offsetY = -style.Bottom
	}
	return offsetX, offsetY, offsetX != 0 || offsetY != 0
}

func newAbsolutePositionRow(style *css.ComputedStyle, rows []core.Row) core.Row {
	if len(rows) == 0 {
		return nil
	}
	child := rows[0]
	if len(rows) > 1 {
		child = &combinedRow{rows: rows}
	}
	return &absoluteRow{
		child:        child,
		x:            style.Left,
		y:            style.Top,
		right:        style.Right,
		width:        style.Width,
		rightAligned: style.Left == 0 && style.Right != 0,
		position:     style.Position,
		zIndex:       style.ZIndex,
		zIndexSet:    style.ZIndexSet,
	}
}
