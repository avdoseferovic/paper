package translate

import (
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// inlineAxisBox aligns a child component along the inline (horizontal) axis
// of its cell — the counterpart to crossAxisBox for justify-items alignment.
// The child renders at the given intrinsic width; the leftover cell width is
// distributed according to align (flex-start / center / flex-end).
type inlineAxisBox struct {
	child core.Component
	align string
	width float64 // mm; intrinsic child width
}

func (b *inlineAxisBox) SetConfig(config *entity.Config) {
	if b.child != nil {
		b.child.SetConfig(config)
	}
}

func (b *inlineAxisBox) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type:    "inline_axis_box",
		Details: map[string]any{"align": b.align, "width": b.width},
	}
	n := node.New(str)
	if b.child != nil {
		n.AddNext(b.child.GetStructure())
	}
	return n
}

func (b *inlineAxisBox) GetHeight(provider core.Provider, cell *entity.Cell) float64 {
	if b.child == nil {
		return 0
	}
	inner := b.innerCell(cell)
	return b.child.GetHeight(provider, &inner)
}

func (b *inlineAxisBox) Render(provider core.Provider, cell *entity.Cell) {
	if b.child == nil {
		return
	}
	inner := b.innerCell(cell)
	b.child.Render(provider, &inner)
}

func (b *inlineAxisBox) innerCell(cell *entity.Cell) entity.Cell {
	inner := cell.Copy()
	if b.width <= 0 || b.width >= cell.Width {
		return inner
	}
	switch b.align {
	case flexAlignCenter:
		inner.X += (cell.Width - b.width) / 2
	case flexAlignEnd:
		inner.X += cell.Width - b.width
	case flexAlignStart:
	default:
		return inner
	}
	inner.Width = b.width
	return inner
}
