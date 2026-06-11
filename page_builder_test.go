package paper

import (
	"reflect"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

func TestPaperDelegatesPaginationStateToPageBuilder(t *testing.T) {
	t.Parallel()

	paperType := reflect.TypeFor[Paper]()
	_, ok := paperType.FieldByName("pageBuilder")
	require.True(t, ok)

	for _, field := range []string{"cell", "pages", "rows", "header", "footer", "currentHeight"} {
		_, ok := paperType.FieldByName(field)
		assert.False(t, ok, "Paper should delegate %s to pageBuilder", field)
	}
}

func TestPageBuilderPlacesAtomicSplittableRowOnFreshPage(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	builder := newPageBuilder(cfg, nil)
	oversized := &atomicSplittableRow{height: builder.cell.Height + 1, didSplit: true}

	builder.addRow(oversized)

	require.Len(t, builder.pages, 0)
	require.Len(t, builder.rows, 1)
	assert.Equal(t, oversized, builder.rows[0])
}

func TestPageBuilderPushesAtomicSplittableRowOnce(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	builder := newPageBuilder(cfg, nil)
	builder.addRow(&atomicSplittableRow{height: 1, didSplit: false})
	oversized := &atomicSplittableRow{height: builder.cell.Height + 1, didSplit: true}

	builder.addRow(oversized)

	require.Len(t, builder.pages, 1)
	require.Len(t, builder.rows, 1)
	assert.Equal(t, oversized, builder.rows[0])
}

func TestPageBuilderSplitsRowAcrossPages(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	builder := newPageBuilder(cfg, nil)
	first := &atomicSplittableRow{height: 5}
	rest := &atomicSplittableRow{height: 10}
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: builder.cell.Height + 1},
		first:               first,
		rest:                rest,
	}

	builder.addRow(tall)

	// first lands on the committed page, rest starts the next page.
	require.Len(t, builder.pages, 1)
	require.Len(t, builder.rows, 1)
	assert.Equal(t, core.Row(rest), builder.rows[0])
}

func TestPageBuilderSplitsRowWithoutRemainder(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	builder := newPageBuilder(cfg, nil)
	first := &atomicSplittableRow{height: 5}
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: builder.cell.Height + 1},
		first:               first,
	}

	builder.addRow(tall)

	// first fills the committed page; nothing carries over.
	require.Len(t, builder.pages, 1)
	assert.Len(t, builder.rows, 0)
}

func TestPageBuilderTreatsNilSplitResultAsAtomicRow(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	builder := newPageBuilder(cfg, nil)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: builder.cell.Height + 1},
	}

	builder.addRow(tall)

	// first==nil and rest==nil falls back to pushing the original row.
	require.Len(t, builder.pages, 0)
	require.Len(t, builder.rows, 1)
	assert.Equal(t, core.Row(tall), builder.rows[0])
}

type atomicSplittableRow struct {
	height   float64
	cols     []core.Col
	didSplit bool
}

func (r *atomicSplittableRow) SetConfig(_ *entity.Config) {}

func (r *atomicSplittableRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "atomic_splittable_row"})
}

func (r *atomicSplittableRow) Add(cols ...core.Col) core.Row {
	r.cols = append(r.cols, cols...)
	return r
}

func (r *atomicSplittableRow) GetHeight(_ core.Provider, _ *entity.Cell) float64 {
	return r.height
}

func (r *atomicSplittableRow) GetColumns() []core.Col {
	return r.cols
}

func (r *atomicSplittableRow) WithStyle(_ *props.Cell) core.Row {
	return r
}

func (r *atomicSplittableRow) Render(_ core.Provider, _ entity.Cell) {}

func (r *atomicSplittableRow) SplitAt(_ core.Provider, _ float64) (core.Row, core.Row, bool) {
	if !r.didSplit {
		return nil, nil, false
	}
	return nil, r, true
}

// splittingRow always reports a split and returns the configured first/rest
// rows, exercising the non-atomic branches of addSplittableRow.
type splittingRow struct {
	atomicSplittableRow
	first core.Row
	rest  core.Row
}

func (r *splittingRow) SplitAt(_ core.Provider, _ float64) (core.Row, core.Row, bool) {
	return r.first, r.rest, true
}
