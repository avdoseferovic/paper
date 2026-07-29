package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// widthRecordingRow records the cell width it is measured at, so a test can
// assert which width SplitAt uses when measuring children.
type widthRecordingRow struct{ widths *[]float64 }

func (r widthRecordingRow) SetConfig(*entity.Config) {}
func (r widthRecordingRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "fake"})
}
func (r widthRecordingRow) Add(...core.Col) core.Row          { return r }
func (r widthRecordingRow) GetColumns() []core.Col            { return nil }
func (r widthRecordingRow) WithStyle(*props.Cell) core.Row    { return r }
func (r widthRecordingRow) Render(core.Provider, entity.Cell) {}
func (r widthRecordingRow) GetHeight(_ core.Provider, cell *entity.Cell) float64 {
	*r.widths = append(*r.widths, cell.Width)
	return 100
}

// TestSplitAtMeasuresChildrenAtRenderWidth guards the pagination regression: a
// container's children must be measured during SplitAt at the real render width
// handed in by the page builder, NOT at a stale cached width (which previously
// could be 0 → fall back to 10000, packing an over-tall chunk that overflowed).
func TestSplitAtMeasuresChildrenAtRenderWidth(t *testing.T) {
	t.Parallel()

	var widths []float64
	rows := []core.Row{
		widthRecordingRow{&widths},
		widthRecordingRow{&widths},
		widthRecordingRow{&widths},
	}
	// Prime the cache with a deliberately wrong (huge) width, as a prior
	// full-width GetHeight pass would.
	c := &blockContainer{rows: rows, cachedWidth: 10000, cachedHeight: 300}
	scr := newSplittableContainerRow(c)

	const renderWidth = 172.0
	// 3 rows × 100mm = 300mm total; only ~1 fits in 150mm remaining → it splits.
	scr.SplitAt(nil, 150, renderWidth)

	if len(widths) == 0 {
		t.Fatal("SplitAt did not measure any child rows")
	}
	for _, w := range widths {
		if w != renderWidth {
			t.Fatalf("child measured at width %v during SplitAt, want render width %v (stale-cache leak)", w, renderWidth)
		}
	}
}
