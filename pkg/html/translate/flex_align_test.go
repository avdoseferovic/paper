package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type fixedHeightComponent struct {
	height       float64
	renderedCell entity.Cell
}

func (f *fixedHeightComponent) SetConfig(*entity.Config) {}

func (f *fixedHeightComponent) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "fixed_height"})
}

func (f *fixedHeightComponent) GetHeight(core.Provider, *entity.Cell) float64 {
	return f.height
}

func (f *fixedHeightComponent) Render(_ core.Provider, cell *entity.Cell) {
	f.renderedCell = cell.Copy()
}

func TestCrossAxisBoxRenderOffsetsChild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		align      string
		wantY      float64
		wantHeight float64
	}{
		{name: "flex-start", align: "flex-start", wantY: 10, wantHeight: 2},
		{name: "center", align: "center", wantY: 14, wantHeight: 2},
		{name: "flex-end", align: "flex-end", wantY: 18, wantHeight: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			child := &fixedHeightComponent{height: 2}
			box := &crossAxisBox{child: child, align: tt.align}

			box.Render(nil, &entity.Cell{X: 5, Y: 10, Width: 20, Height: 10})

			assert.Equal(t, 5.0, child.renderedCell.X)
			assert.Equal(t, 20.0, child.renderedCell.Width)
			assert.Equal(t, tt.wantY, child.renderedCell.Y)
			assert.Equal(t, tt.wantHeight, child.renderedCell.Height)
		})
	}
}
