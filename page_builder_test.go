package paper

import (
	"reflect"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/translate"
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

func TestPageBuilderSuppressesFooterReservationForCurrentPage(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(0).
		WithBottomMargin(0).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
	}))
	builder.addRow(fixedRow(95))

	assert.Len(t, builder.pages, 0, "95mm should fit when the current page suppresses the 10mm footer")
	require.Len(t, builder.rows, 1)
	assert.Equal(t, 95.0, builder.currentHeight)
}

func TestPageBuilderUsesPageControlTopMarginForCurrentPage(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		Build()
	builder := newPageBuilder(cfg, nil)
	top := 0.0

	builder.addRow(translate.NewPageControlRow(core.PageControl{TopMargin: &top}))
	builder.addRow(fixedRow(95))

	assert.Len(t, builder.pages, 0, "95mm should fit when the current page top margin is overridden to zero")
	require.Len(t, builder.rows, 1)
	assert.Equal(t, 95.0, builder.currentHeight)
}

func TestPageBuilderKeepsPageControlForSplitContinuation(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		WithPageNumber(props.PageNumber{Pattern: "{current}/{total}"}).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))
	top := 0.0
	first := fixedRow(95)
	rest := fixedRow(95)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: 150},
		first:               first,
		rest:                rest,
	}

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
		TopMargin:          &top,
	}))
	builder.addRow(tall)
	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, []bool{false, false}, builder.pageNumbered)
	assert.Equal(t, first, builder.pages[0].GetRows()[0])
	assert.Equal(t, rest, builder.pages[1].GetRows()[0])
}

func TestPageBuilderFirstPageOnlyTopMarginDoesNotCarryToSplitContinuation(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))
	top := 0.0
	first := fixedRow(95)
	rest := fixedRow(80)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: 150},
		first:               first,
		rest:                rest,
	}

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:         true,
		SuppressPageNumber:     true,
		TopMargin:              &top,
		TopMarginFirstPageOnly: true,
	}))
	builder.addRow(tall)

	require.Len(t, builder.pages, 1)
	assert.True(t, builder.currentControl.SuppressFooter)
	assert.True(t, builder.currentControl.SuppressPageNumber)
	assert.Nil(t, builder.currentControl.TopMargin)

	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, first, builder.pages[0].GetRows()[0])
	assert.Equal(t, rest, builder.pages[1].GetRows()[0])
}

func TestPageBuilderUsesExplicitContinuationTopMarginAfterSplit(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))
	firstTop := 1.0
	continuationTop := 0.0
	first := fixedRow(94)
	rest := fixedRow(95)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: 150},
		first:               first,
		rest:                rest,
	}

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:         true,
		SuppressPageNumber:     true,
		TopMargin:              &firstTop,
		TopMarginFirstPageOnly: true,
		ContinuationTopMargin:  &continuationTop,
	}))
	builder.addRow(tall)

	require.Len(t, builder.pages, 1)
	require.NotNil(t, builder.currentControl.TopMargin)
	assert.Equal(t, 0.0, *builder.currentControl.TopMargin)

	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, first, builder.pages[0].GetRows()[0])
	assert.Equal(t, rest, builder.pages[1].GetRows()[0])
}

func TestPageBuilderUsesExplicitContinuationRenderOffsetAfterSplit(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))
	firstOffsetX := 0.5
	firstOffsetY := 1.0
	continuationOffsetX := 0.0
	continuationOffsetY := 0.0
	first := fixedRow(90)
	rest := fixedRow(85)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: 150},
		first:               first,
		rest:                rest,
	}

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:            true,
		SuppressPageNumber:        true,
		RenderOffsetX:             &firstOffsetX,
		RenderOffsetY:             &firstOffsetY,
		RenderOffsetFirstPageOnly: true,
		ContinuationRenderOffsetX: &continuationOffsetX,
		ContinuationRenderOffsetY: &continuationOffsetY,
	}))
	builder.addRow(tall)

	require.Len(t, builder.pages, 1)
	require.NotNil(t, builder.currentControl.RenderOffsetX)
	require.NotNil(t, builder.currentControl.RenderOffsetY)
	assert.Equal(t, 0.0, *builder.currentControl.RenderOffsetX)
	assert.Equal(t, 0.0, *builder.currentControl.RenderOffsetY)

	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, first, builder.pages[0].GetRows()[0])
	assert.Equal(t, rest, builder.pages[1].GetRows()[0])
}

func TestPageBuilderFirstPageOnlyDecorSuppressionDoesNotCarryToSplitContinuation(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(10).
		WithBottomMargin(0).
		WithPageNumber(props.PageNumber{Pattern: "{current}/{total}"}).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))
	top := 0.0
	first := fixedRow(95)
	rest := fixedRow(80)
	tall := &splittingRow{
		atomicSplittableRow: atomicSplittableRow{height: 150},
		first:               first,
		rest:                rest,
	}

	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
		DecorFirstPageOnly: true,
		TopMargin:          &top,
	}))
	builder.addRow(tall)

	require.Len(t, builder.pages, 1)
	assert.Equal(t, []bool{false}, builder.pageNumbered)
	assert.False(t, builder.currentControl.SuppressFooter)
	assert.False(t, builder.currentControl.SuppressPageNumber)
	require.NotNil(t, builder.currentControl.TopMargin)
	assert.Equal(t, 0.0, *builder.currentControl.TopMargin)

	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, []bool{false, true}, builder.pageNumbered)
	assert.Equal(t, []bool{false, true}, builder.pageNumberCounted)
	assert.Equal(t, first, builder.pages[0].GetRows()[0])
	assert.Equal(t, rest, builder.pages[1].GetRows()[0])
}

func TestPageBuilderBlankAndSuppressedPagesAreExcludedFromPageNumberTotal(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(0).
		WithBottomMargin(0).
		WithPageNumber(props.PageNumber{Pattern: "{current}/{total}"}).
		Build()
	builder := newPageBuilder(cfg, nil)

	builder.addRow(fixedRow(10))
	builder.addRow(translate.NewBlankPageRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
	}))
	builder.addRow(translate.NewPageControlRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
	}))
	builder.addRow(fixedRow(95))
	builder.addRow(fixedRow(95))
	builder.finalize()

	require.Len(t, builder.pages, 4)
	require.Len(t, builder.pageNumbered, 4)
	assert.Equal(t, []bool{true, false, false, true}, builder.pageNumbered)
	assert.Equal(t, []bool{true, false, false, true}, builder.pageNumberCounted)
}

func TestPageBuilderBlankPageCanCountWithoutPageNumberStamp(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(0).
		WithBottomMargin(0).
		WithPageNumber(props.PageNumber{Pattern: "{current}/{total}"}).
		Build()
	builder := newPageBuilder(cfg, nil)

	builder.addRow(fixedRow(10))
	builder.addRow(translate.NewBlankPageRow(core.PageControl{
		SuppressFooter:     true,
		SuppressPageNumber: true,
		CountPageNumber:    true,
	}))
	builder.addRow(fixedRow(10))
	builder.finalize()

	require.Len(t, builder.pages, 3)
	assert.Equal(t, []bool{true, false, true}, builder.pageNumbered)
	assert.Equal(t, []bool{true, true, true}, builder.pageNumberCounted)
	assert.Equal(t, 1, builder.pages[0].GetNumber())
	assert.Equal(t, 0, builder.pages[1].GetNumber(), "count-only blank page should not receive a visible stamp")
	assert.Equal(t, 3, builder.pages[2].GetNumber())
}

func TestPageBuilderBlankPageCanKeepFooterAndPageNumber(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithDimensions(50, 100).
		WithTopMargin(0).
		WithBottomMargin(0).
		WithPageNumber(props.PageNumber{Pattern: "{current}/{total}"}).
		Build()
	builder := newPageBuilder(cfg, nil)
	require.NoError(t, builder.registerFooter(fixedRow(10)))

	builder.addRow(fixedRow(10))
	builder.addRow(translate.NewBlankPageRow(core.PageControl{}))
	builder.finalize()

	require.Len(t, builder.pages, 2)
	assert.Equal(t, []bool{true, true}, builder.pageNumbered)
	require.Len(t, builder.pages[1].GetRows(), 2, "decorated blank page should contain spacer plus footer rows")
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

func fixedRow(height float64) core.Row {
	return row.New(height).Add(col.New())
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

func (r *atomicSplittableRow) SplitAt(_ core.Provider, _, _ float64) (core.Row, core.Row, bool) {
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

func (r *splittingRow) SplitAt(_ core.Provider, _, _ float64) (core.Row, core.Row, bool) {
	return r.first, r.rest, true
}
