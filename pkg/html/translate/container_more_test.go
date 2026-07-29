package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestMarginBox_GetHeight_NilChildReturnsMargins(t *testing.T) {
	t.Parallel()
	m := &marginBox{marginTop: 3, marginBottom: 4}
	assert.Equal(t, 7.0, m.GetHeight(nil, &entity.Cell{Width: 100, Height: 100}))
}

func TestBlockContainerPadding_IncludesHorizontalVisibleBorders(t *testing.T) {
	t.Parallel()

	style := &css.ComputedStyle{
		PaddingTop:        1,
		PaddingRight:      2,
		PaddingBottom:     3,
		PaddingLeft:       4,
		BorderTopWidth:    5,
		BorderRightWidth:  6,
		BorderBottomWidth: 7,
		BorderLeftWidth:   8,
		BorderTopStyle:    "solid",
		BorderRightStyle:  "solid",
		BorderBottomStyle: "solid",
		BorderLeftStyle:   "solid",
	}

	top, right, bottom, left := blockContainerPadding(style)

	assert.Equal(t, 1.0, top)
	assert.Equal(t, 8.0, right)
	assert.Equal(t, 3.0, bottom)
	assert.Equal(t, 12.0, left)
}

func TestBlockContainerPadding_IgnoresNonVisibleHorizontalBorderStyles(t *testing.T) {
	t.Parallel()

	style := &css.ComputedStyle{
		PaddingTop:        1,
		PaddingRight:      2,
		PaddingBottom:     3,
		PaddingLeft:       4,
		BorderTopWidth:    5,
		BorderRightWidth:  6,
		BorderBottomWidth: 7,
		BorderLeftWidth:   8,
		BorderTopStyle:    "none",
		BorderRightStyle:  "hidden",
		BorderBottomStyle: "none",
		BorderLeftStyle:   "hidden",
	}

	top, right, bottom, left := blockContainerPadding(style)

	assert.Equal(t, 1.0, top)
	assert.Equal(t, 2.0, right)
	assert.Equal(t, 3.0, bottom)
	assert.Equal(t, 4.0, left)
}

func TestMarginBox_GetHeight_AddsMarginsToChildHeight(t *testing.T) {
	t.Parallel()
	m := &marginBox{
		child:        &fixedHeightComponent{height: 10},
		marginTop:    2,
		marginBottom: 3,
	}
	assert.Equal(t, 15.0, m.GetHeight(nil, &entity.Cell{Width: 100, Height: 100}))
}

func TestMarginBox_Render_NilChildIsNoOp(t *testing.T) {
	t.Parallel()
	m := &marginBox{marginTop: 1}
	m.Render(nil, &entity.Cell{Width: 10, Height: 10}) // must not panic
}

func TestMarginBox_Render_OffsetsChildByMargins(t *testing.T) {
	t.Parallel()
	child := &fixedHeightComponent{height: 5}
	m := &marginBox{
		child:        child,
		marginTop:    2,
		marginRight:  3,
		marginBottom: 4,
		marginLeft:   1,
	}

	m.Render(nil, &entity.Cell{X: 10, Y: 20, Width: 100, Height: 50})

	assert.Equal(t, 11.0, child.renderedCell.X)
	assert.Equal(t, 22.0, child.renderedCell.Y)
	assert.Equal(t, 96.0, child.renderedCell.Width)
	assert.Equal(t, 44.0, child.renderedCell.Height)
}

func TestMarginBox_InnerCell_ClampsNegativeDimensions(t *testing.T) {
	t.Parallel()
	m := &marginBox{marginTop: 20, marginBottom: 20, marginLeft: 20, marginRight: 20}
	inner := m.innerCell(&entity.Cell{Width: 10, Height: 10})
	assert.Equal(t, 0.0, inner.Width)
	assert.Equal(t, 0.0, inner.Height)
}

func TestMarginBox_SetConfig(t *testing.T) {
	t.Parallel()
	noChild := &marginBox{}
	noChild.SetConfig(&entity.Config{}) // must not panic

	withChild := &marginBox{child: &fixedHeightComponent{}}
	withChild.SetConfig(&entity.Config{MaxGridSize: 12})
}

func TestMarginBox_GetStructure_IncludesMargins(t *testing.T) {
	t.Parallel()
	m := &marginBox{child: &fixedHeightComponent{}, marginTop: 1, marginBottom: 2}
	str := m.GetStructure()
	assert.Equal(t, "margin_box", str.GetData().Type)
	assert.Equal(t, 1.0, str.GetData().Details["margin_top"])
	assert.Equal(t, 2.0, str.GetData().Details["margin_bottom"])
	assert.Len(t, str.GetNexts(), 1)
}

func TestHorizontalMarginRow_RenderOffsetsChildWidth(t *testing.T) {
	t.Parallel()
	child := &recordingRow{height: 5}
	row := &horizontalMarginRow{
		child:       child,
		marginLeft:  4,
		marginRight: 6,
	}

	row.Render(nil, entity.Cell{X: 10, Y: 20, Width: 100, Height: 50})

	require.True(t, child.rendered)
	assert.Equal(t, 14.0, child.renderedCell.X)
	assert.Equal(t, 20.0, child.renderedCell.Y)
	assert.Equal(t, 90.0, child.renderedCell.Width)
}

func TestHorizontalMarginRow_GetHeightMeasuresInnerWidth(t *testing.T) {
	t.Parallel()
	var widths []float64
	row := &horizontalMarginRow{
		child:       widthRecordingRow{widths: &widths},
		marginLeft:  4,
		marginRight: 6,
	}

	_ = row.GetHeight(nil, &entity.Cell{Width: 100, Height: 50})

	require.Len(t, widths, 1)
	assert.Equal(t, 90.0, widths[0])
}

func TestHorizontalMarginRow_SplitAtDelegatesInnerWidth(t *testing.T) {
	t.Parallel()
	var widths []float64
	rows := []core.Row{
		widthRecordingRow{widths: &widths},
		widthRecordingRow{widths: &widths},
	}
	child := newSplittableContainerRow(&blockContainer{rows: rows})
	row := &horizontalMarginRow{
		child:       child,
		marginLeft:  4,
		marginRight: 6,
	}

	_, _, didSplit := row.SplitAt(nil, 150, 100)

	require.True(t, didSplit)
	require.NotEmpty(t, widths)
	for _, w := range widths {
		assert.Equal(t, 90.0, w)
	}
}

func TestTranslate_BlockHorizontalMarginsWrapGenericRows(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="margin-left:4mm;margin-right:6mm"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var found bool
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "horizontal_margin_row" {
			found = true
			assert.InDelta(t, 4.0, s.Details["margin_left"], 0.001)
			assert.InDelta(t, 6.0, s.Details["margin_right"], 0.001)
		}
	})
	assert.True(t, found, "expected block margin-left/right to wrap generic content rows")
}

// countingRow counts GetHeight invocations to observe blockContainer caching.
type countingRow struct {
	recordingRow
	heightCalls int
}

func (r *countingRow) GetHeight(_ core.Provider, _ *entity.Cell) float64 {
	r.heightCalls++
	return r.height
}

func TestBlockContainer_GetHeight_CachesPerWidth(t *testing.T) {
	t.Parallel()
	child := &countingRow{recordingRow: recordingRow{height: 8}}
	b := &blockContainer{rows: []core.Row{child}, paddingTop: 1, paddingBottom: 1}
	cell := &entity.Cell{Width: 100, Height: 100}

	first := b.GetHeight(nil, cell)
	second := b.GetHeight(nil, cell)

	assert.Equal(t, 10.0, first)
	assert.Equal(t, first, second)
	assert.Equal(t, 1, child.heightCalls, "second call must hit the cache")

	// A different width invalidates the cache.
	_ = b.GetHeight(nil, &entity.Cell{Width: 50, Height: 100})
	assert.Equal(t, 2, child.heightCalls)
}

func TestBlockContainer_GetHeight_ClampsNegativeInnerWidth(t *testing.T) {
	t.Parallel()
	child := &countingRow{recordingRow: recordingRow{height: 5}}
	b := &blockContainer{rows: []core.Row{child}, paddingLeft: 20, paddingRight: 20}

	got := b.GetHeight(nil, &entity.Cell{Width: 10, Height: 100})

	assert.Equal(t, 5.0, got)
}

func TestBlockContainer_GetHeight_UsesExplicitHeightConstraints(t *testing.T) {
	t.Parallel()

	cell := &entity.Cell{Width: 100, Height: 100}
	assert.Equal(t, 20.0, (&blockContainer{
		rows:   []core.Row{&recordingRow{height: 5}},
		height: 20,
	}).GetHeight(&cursorProvider{}, cell))
	assert.Equal(t, 12.0, (&blockContainer{
		rows:      []core.Row{&recordingRow{height: 5}},
		minHeight: 12,
	}).GetHeight(&cursorProvider{}, cell))
	assert.Equal(t, 6.0, (&blockContainer{
		rows:      []core.Row{&recordingRow{height: 10}},
		maxHeight: 6,
	}).GetHeight(&cursorProvider{}, cell))
}

func TestSplittableContainerRow_DelegatesRowMethods(t *testing.T) {
	t.Parallel()
	container := &blockContainer{rows: []core.Row{buildFixedHeightRow(5)}}
	scr := newSplittableContainerRow(container)
	cfg := &entity.Config{MaxGridSize: 12}
	scr.SetConfig(cfg)

	cols := scr.GetColumns()
	require.Len(t, cols, 1, "inner row wraps the container in a single col")

	assert.NotNil(t, scr.WithStyle(nil))
	assert.NotNil(t, scr.Add())
	assert.NotNil(t, scr.GetStructure())

	// Render must delegate to the inner row without panicking.
	scr.Render(&cursorProvider{}, entity.Cell{Width: 100, Height: 50})
}

// noPositionProvider shadows SetCursor with an incompatible signature so the
// outer type no longer satisfies core.PositionProvider while still being a
// full core.Provider via the embedded cursorProvider.
type noPositionProvider struct{ cursorProvider }

func (*noPositionProvider) SetCursor() {}

type createColRecordingProvider struct {
	cursorProvider
	created []entity.Cell
}

func (p *createColRecordingProvider) CreateCol(width, height float64, _ *entity.Config, _ *props.Cell) {
	p.created = append(p.created, entity.Cell{Width: width, Height: height})
}

func TestBlockContainer_Render_PaintsStyleAndClampsHeight(t *testing.T) {
	t.Parallel()
	child := &recordingRow{height: 30}
	b := &blockContainer{
		rows:        []core.Row{child},
		style:       &props.Cell{},
		paddingTop:  2,
		paddingLeft: 1,
	}
	b.SetConfig(&entity.Config{MaxGridSize: 12})

	// cell.Height (10) is smaller than content height (32): clamp branch.
	b.Render(&cursorProvider{}, &entity.Cell{X: 5, Y: 5, Width: 100, Height: 10})

	assert.True(t, child.rendered)
	assert.Equal(t, 6.0, child.renderedCell.X, "padding-left offsets children")
	assert.Equal(t, 7.0, child.renderedCell.Y, "padding-top offsets children")
}

func TestBlockContainer_Render_PaintsLargerAssignedHeight(t *testing.T) {
	t.Parallel()
	child := &recordingRow{height: 5}
	b := &blockContainer{
		rows:  []core.Row{child},
		style: &props.Cell{},
	}
	b.SetConfig(&entity.Config{MaxGridSize: 12})
	provider := &createColRecordingProvider{}

	b.Render(provider, &entity.Cell{Width: 100, Height: 20})

	require.Len(t, provider.created, 1)
	assert.Equal(t, 20.0, provider.created[0].Height)
	assert.True(t, child.rendered)
}

func TestBlockContainer_Render_WithoutPositionProvider(t *testing.T) {
	t.Parallel()
	child := &recordingRow{height: 5}
	b := &blockContainer{rows: []core.Row{child}, style: &props.Cell{}}
	b.SetConfig(&entity.Config{MaxGridSize: 12})

	b.Render(&noPositionProvider{}, &entity.Cell{Width: 100, Height: 50})

	assert.True(t, child.rendered, "render must work without cursor support")
}

func TestBlockContainer_Render_ClampsNegativeInnerWidth(t *testing.T) {
	t.Parallel()
	child := &recordingRow{height: 5}
	b := &blockContainer{rows: []core.Row{child}, paddingLeft: 20, paddingRight: 20}
	b.SetConfig(&entity.Config{MaxGridSize: 12})

	b.Render(&cursorProvider{}, &entity.Cell{Width: 10, Height: 50})

	assert.True(t, child.rendered)
	assert.Equal(t, 0.0, child.renderedCell.Width)
}

func TestSplittableContainerRow_SplitAt_NilContainer(t *testing.T) {
	t.Parallel()
	scr := &splittableContainerRow{}
	first, rest, didSplit := scr.SplitAt(&cursorProvider{}, 10, 0)
	assert.Nil(t, first)
	assert.Nil(t, rest)
	assert.False(t, didSplit)
}
