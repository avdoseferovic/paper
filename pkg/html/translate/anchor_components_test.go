package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// linkRecordingProvider extends cursorProvider with core.LinkProvider so
// anchorTarget.Render can register named destinations.
type linkRecordingProvider struct {
	cursorProvider
	nextID   int
	setLinks []struct {
		id   int
		y    float64
		page int
	}
}

func (p *linkRecordingProvider) AddLink() int {
	p.nextID++
	return p.nextID
}

func (p *linkRecordingProvider) SetLink(linkID int, y float64, page int) {
	p.setLinks = append(p.setLinks, struct {
		id   int
		y    float64
		page int
	}{linkID, y, page})
}

func (p *linkRecordingProvider) Link(_, _, _, _ float64, _ int) {}

// recordingRow records the cell and config it receives, with a fixed height.
type recordingRow struct {
	height       float64
	renderedCell entity.Cell
	config       *entity.Config
	rendered     bool
}

func (r *recordingRow) SetConfig(cfg *entity.Config) { r.config = cfg }

func (r *recordingRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "recording_row"})
}

func (r *recordingRow) GetHeight(_ core.Provider, _ *entity.Cell) float64 { return r.height }

func (r *recordingRow) Render(_ core.Provider, cell entity.Cell) {
	r.rendered = true
	r.renderedCell = cell
}

func (r *recordingRow) Add(_ ...core.Col) core.Row     { return r }
func (r *recordingRow) WithStyle(*props.Cell) core.Row { return r }
func (r *recordingRow) GetColumns() []core.Col         { return nil }

func TestCombinedRow_GetHeight_SumsChildRows(t *testing.T) {
	t.Parallel()
	c := &combinedRow{rows: []core.Row{
		&recordingRow{height: 5},
		&recordingRow{height: 7},
	}}
	got := c.GetHeight(nil, &entity.Cell{Width: 100, Height: 100})
	assert.Equal(t, 12.0, got)
}

func TestCombinedRow_Render_StacksRowsVertically(t *testing.T) {
	t.Parallel()
	first := &recordingRow{height: 5}
	second := &recordingRow{height: 7}
	c := &combinedRow{rows: []core.Row{first, second}}

	c.Render(nil, entity.Cell{X: 1, Y: 10, Width: 100, Height: 50})

	require.True(t, first.rendered)
	require.True(t, second.rendered)
	assert.Equal(t, 10.0, first.renderedCell.Y)
	assert.Equal(t, 5.0, first.renderedCell.Height)
	assert.Equal(t, 15.0, second.renderedCell.Y, "second row starts after first row's height")
	assert.Equal(t, 7.0, second.renderedCell.Height)
}

func TestCombinedRow_SetConfig_Propagates(t *testing.T) {
	t.Parallel()
	child := &recordingRow{}
	c := &combinedRow{rows: []core.Row{child}}
	cfg := &entity.Config{MaxGridSize: 12}

	c.SetConfig(cfg)

	assert.Equal(t, cfg, c.config)
	assert.Equal(t, cfg, child.config)
}

func TestCombinedRow_GetStructure_ListsChildren(t *testing.T) {
	t.Parallel()
	c := &combinedRow{rows: []core.Row{&recordingRow{}, &recordingRow{}}}
	str := c.GetStructure()
	assert.Equal(t, "combined", str.GetData().Type)
	assert.Equal(t, 2, str.GetData().Details["rows"])
	assert.Len(t, str.GetNexts(), 2)
}

func TestCombinedRow_NoopRowMethods(t *testing.T) {
	t.Parallel()
	c := &combinedRow{}
	assert.Equal(t, core.Row(c), c.Add())
	assert.Equal(t, core.Row(c), c.WithStyle(nil))
	assert.Nil(t, c.GetColumns())
}

func TestAnchorRegistry_EnsureLinkID(t *testing.T) {
	t.Parallel()

	t.Run("nil link provider returns not ok when unregistered", func(t *testing.T) {
		t.Parallel()
		reg := newAnchorRegistry()
		id, ok := reg.EnsureLinkID("missing", nil)
		assert.Equal(t, 0, id)
		assert.False(t, ok)
	})

	t.Run("registers once and reuses the same id", func(t *testing.T) {
		t.Parallel()
		reg := newAnchorRegistry()
		lp := &linkRecordingProvider{}

		first, ok := reg.EnsureLinkID("sec", lp)
		require.True(t, ok)
		second, ok := reg.EnsureLinkID("sec", lp)
		require.True(t, ok)

		assert.Equal(t, first, second)
		assert.Equal(t, 1, lp.nextID, "AddLink must be called exactly once per name")
	})

	t.Run("registered name resolves without a provider", func(t *testing.T) {
		t.Parallel()
		reg := newAnchorRegistry()
		lp := &linkRecordingProvider{}
		id, ok := reg.EnsureLinkID("sec", lp)
		require.True(t, ok)

		got, ok := reg.EnsureLinkID("sec", nil)
		assert.True(t, ok)
		assert.Equal(t, id, got)
	})
}

func TestAnchorTarget_GetHeight(t *testing.T) {
	t.Parallel()

	t.Run("nil child returns zero", func(t *testing.T) {
		t.Parallel()
		a := newAnchorTarget(nil, "x", newAnchorRegistry())
		assert.Equal(t, 0.0, a.GetHeight(nil, &entity.Cell{}))
	})

	t.Run("delegates to child", func(t *testing.T) {
		t.Parallel()
		a := newAnchorTarget(&fixedHeightComponent{height: 4}, "x", newAnchorRegistry())
		assert.Equal(t, 4.0, a.GetHeight(nil, &entity.Cell{}))
	})
}

func TestAnchorTarget_SetConfig_PropagatesToChild(t *testing.T) {
	t.Parallel()
	cfg := &entity.Config{MaxGridSize: 12}

	withChild := newAnchorTarget(&fixedHeightComponent{}, "x", newAnchorRegistry())
	withChild.SetConfig(cfg)
	assert.Equal(t, cfg, withChild.config)

	noChild := newAnchorTarget(nil, "x", newAnchorRegistry())
	noChild.SetConfig(cfg) // must not panic
	assert.Equal(t, cfg, noChild.config)
}

func TestAnchorTarget_GetStructure_IncludesName(t *testing.T) {
	t.Parallel()
	a := newAnchorTarget(&fixedHeightComponent{}, "dest", newAnchorRegistry())
	str := a.GetStructure()
	assert.Equal(t, "anchor_target", str.GetData().Type)
	assert.Equal(t, "dest", str.GetData().Details["name"])
	assert.Len(t, str.GetNexts(), 1)
}

func TestAnchorTarget_Render_SetsLinkOnLinkProvider(t *testing.T) {
	t.Parallel()
	child := &fixedHeightComponent{height: 2}
	a := newAnchorTarget(child, "dest", newAnchorRegistry())
	lp := &linkRecordingProvider{}

	a.Render(lp, &entity.Cell{X: 1, Y: 33, Width: 10, Height: 10})

	require.Len(t, lp.setLinks, 1)
	assert.Equal(t, 33.0, lp.setLinks[0].y)
	assert.Equal(t, -1, lp.setLinks[0].page)
	assert.Equal(t, 33.0, child.renderedCell.Y, "child must still render")
}

func TestAnchorTarget_Render_PlainProviderStillRendersChild(t *testing.T) {
	t.Parallel()
	child := &fixedHeightComponent{height: 2}
	a := newAnchorTarget(child, "dest", newAnchorRegistry())

	a.Render(&cursorProvider{}, &entity.Cell{Y: 5, Width: 10, Height: 10})

	assert.Equal(t, 5.0, child.renderedCell.Y)
}

func TestAnchorTarget_Render_NilChildOnlySetsLink(t *testing.T) {
	t.Parallel()
	a := newAnchorTarget(nil, "dest", newAnchorRegistry())
	lp := &linkRecordingProvider{}

	a.Render(lp, &entity.Cell{Y: 7})

	require.Len(t, lp.setLinks, 1)
	assert.Equal(t, 7.0, lp.setLinks[0].y)
}
