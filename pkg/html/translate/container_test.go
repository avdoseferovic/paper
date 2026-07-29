package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

func TestBlockContainer_DivWithBackground_ProducesSingleStyledRow(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="background-color:#eaf1fb; padding:5mm"><p>A</p><p>B</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	require.Len(t, rows, 1, "div with bg+padding should collapse to one wrapper row")

	cols := rows[0].GetColumns()
	require.Len(t, cols, 1, "wrapper row should have one col")

	// Inspect the structure tree to confirm a container node with two child rows.
	str := rows[0].GetStructure()
	// Walk children: should find a container with rows=2
	var found bool
	for _, child := range str.GetNexts() {
		// Each col contains a container component
		for _, comp := range child.GetNexts() {
			d := comp.GetData()
			if d.Type == "container" {
				found = true
				if rowsCount, ok := d.Details["rows"]; ok {
					assert.Equal(t, 2, rowsCount, "container should contain two child rows (the <p>s)")
				}
			}
		}
	}
	assert.True(t, found, "expected to find a container structure node")
}

func TestBlockContainer_DivWithGradientOnly_ProducesStyledRow(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="background-image:linear-gradient(to right, red, blue)"><p>A</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)

	require.Len(t, rows, 1, "div with gradient should collapse to one wrapper row")

	var found bool
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "container" {
			found = true
		}
	})
	assert.True(t, found, "expected to find a gradient container structure node")
}

func TestBlockContainer_PlainDivStillFlattens(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div><p>a</p><p>b</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	// No styling → keep flat behaviour, 2 rows
	assert.Len(t, rows, 2)
}

func TestBlockContainer_RenderRestoresCursorForParentRowAdvance(t *testing.T) {
	t.Parallel()

	provider := &cursorProvider{}
	cfg := &entity.Config{MaxGridSize: 12}
	rows := []core.Row{
		row.New(5).Add(col.New(12)),
		row.New(7).Add(col.New(12)),
	}
	container := &blockContainer{rows: rows}
	container.SetConfig(cfg)

	cell := &entity.Cell{X: 10, Y: 20, Width: 100, Height: 12}
	container.Render(provider, cell)

	assert.Equal(t, 10.0, provider.x)
	assert.Equal(t, 20.0, provider.y)
}

func TestFlexCellContent_RenderRestoresCursorForParentRowAdvance(t *testing.T) {
	t.Parallel()

	provider := &cursorProvider{}
	cfg := &entity.Config{MaxGridSize: 12}
	content := newFlexCellContent([]core.Row{
		row.New(5).Add(col.New(12)),
		row.New(7).Add(col.New(12)),
	})
	content.SetConfig(cfg)

	cell := &entity.Cell{X: 10, Y: 20, Width: 100, Height: 12}
	content.Render(provider, cell)

	assert.Equal(t, 10.0, provider.x)
	assert.Equal(t, 20.0, provider.y)
}

func TestCSSCascade_ClassAndInline(t *testing.T) {
	t.Parallel()

	t.Run("user class rule applies background-color", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><head><style>.band{background-color:#ff0000}</style></head>
			<body><h2 class="band">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		assert.Equal(t, 255, s.BackgroundColor.R)
		assert.Equal(t, 0, s.BackgroundColor.G)
	})

	t.Run("inline style wins over class rule", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><head><style>.band{background-color:#ff0000}</style></head>
			<body><h2 class="band" style="background-color:#00ff00">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		assert.Equal(t, 0, s.BackgroundColor.R)
		assert.Equal(t, 255, s.BackgroundColor.G)
	})

	t.Run("class selector wins over tag selector", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><head><style>h2{background-color:#ff0000}.band{background-color:#0000ff}</style></head>
			<body><h2 class="band">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		assert.Equal(t, 0, s.BackgroundColor.R)
		assert.Equal(t, 255, s.BackgroundColor.B)
	})
}

func TestBuiltinCSS_ListMargins(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><h2>PAYMENT</h2><ol><li>One</li></ol></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	structure := rows[1].GetStructure()
	found := false
	walkStructure(structure, func(s core.Structure) {
		if s.Type == "margin_box" {
			found = true
			assert.Equal(t, 2.0, s.Details["margin_top"])
			assert.Equal(t, 1.0, s.Details["margin_bottom"])
		}
	})
	assert.True(t, found, "expected built-in ol margin to wrap list content")
}

func walkStructure(n *node.Node[core.Structure], fn func(core.Structure)) {
	if n == nil {
		return
	}
	fn(n.GetData())
	for _, child := range n.GetNexts() {
		walkStructure(child, fn)
	}
}

func TestShouldUseContainer(t *testing.T) {
	t.Parallel()

	t.Run("nil style", func(t *testing.T) {
		t.Parallel()
		assert.False(t, shouldUseContainer(nil))
	})

	t.Run("background-color triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="background-color:red"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("background-gradient triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="background-image:linear-gradient(to right, red, blue)"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("padding-only triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="padding:5mm"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("border triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="border:1pt solid red"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("plain div does not trigger", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.False(t, shouldUseContainer(s))
	})
}

type cursorProvider struct {
	x float64
	y float64
}

func (p *cursorProvider) SetCursor(x, y float64) {
	p.x = x
	p.y = y
}

func (p *cursorProvider) CreateRow(height float64) {
	p.y += height
}

func (p *cursorProvider) CreateCol(_, _ float64, _ *entity.Config, _ *props.Cell) {}

func (p *cursorProvider) AddLine(_ *entity.Cell, _ *props.Line) {}

func (p *cursorProvider) AddText(_ string, _ *entity.Cell, _ *props.Text) {}

func (p *cursorProvider) AddCheckbox(_ string, _ *entity.Cell, _ *props.Checkbox) {}

func (p *cursorProvider) GetFontHeight(_ *props.Font) float64 { return 1 }

func (p *cursorProvider) GetLinesQuantity(_ string, _ *props.Text, _ float64) int {
	return 1
}

func (p *cursorProvider) AddMatrixCode(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddQrCode(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddBarCode(_ string, _ *entity.Cell, _ *props.Barcode) {}

func (p *cursorProvider) GetDimensionsByMatrixCode(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByImageByte(_ []byte, _ extension.Type) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByImage(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByQrCode(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) AddImageFromFile(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddImageFromBytes(_ []byte, _ *entity.Cell, _ *props.Rect, _ extension.Type) {
}

func (p *cursorProvider) AddBackgroundImageFromBytes(_ []byte, _ *entity.Cell, _ *props.Rect, _ extension.Type) {
}

func (p *cursorProvider) GenerateBytes() ([]byte, error) { return nil, nil }

func (p *cursorProvider) SetProtection(_ *entity.Protection) {}

func (p *cursorProvider) SetCompression(_ bool) {}

func (p *cursorProvider) SetMetadata(_ *entity.Metadata) {}

// ── Splittable container ──────────────────────────────────────────────────────

func TestSplittableContainerRow_SplitAt(t *testing.T) {
	t.Parallel()
	cfg := &entity.Config{MaxGridSize: 12}

	// SplitAt records the last measured content width, so each subtest gets its
	// own container and row rather than sharing mutated state.
	newSCR := func() *splittableContainerRow {
		// Three fixed-height rows of 10mm each, so the container is 30mm tall.
		container := &blockContainer{
			rows: []core.Row{buildFixedHeightRow(10), buildFixedHeightRow(10), buildFixedHeightRow(10)},
		}
		scr := newSplittableContainerRow(container)
		scr.SetConfig(cfg)
		return scr
	}

	t.Run("container total height is 30mm", func(t *testing.T) {
		t.Parallel()
		p := &cursorProvider{}
		container := &blockContainer{
			rows: []core.Row{buildFixedHeightRow(10), buildFixedHeightRow(10), buildFixedHeightRow(10)},
		}
		assert.InDelta(t, 30.0, container.GetHeight(p, &entity.Cell{Width: 100, Height: 100}), 0.1)
	})

	t.Run("SplitAt remaining=25 splits after 2 rows (20mm) + partial", func(t *testing.T) {
		t.Parallel()
		p := &cursorProvider{}
		scr := newSCR()
		first, rest, didSplit := scr.SplitAt(p, 25, 0)
		require.True(t, didSplit, "30mm container should split when remaining=25mm")
		require.NotNil(t, first)
		require.NotNil(t, rest)
		first.SetConfig(cfg)
		// first should contain rows that fit in 25mm
		assert.LessOrEqual(t, first.GetHeight(p, &entity.Cell{Width: 100, Height: 100}), 25.0+0.01)
	})

	t.Run("SplitAt remaining=100 does not split (fits)", func(t *testing.T) {
		t.Parallel()
		p := &cursorProvider{}
		_, _, didSplit := newSCR().SplitAt(p, 100, 0)
		assert.False(t, didSplit, "container that fits should not split")
	})

	t.Run("SplitAt remaining=1 returns atomic push (nil first) when no row fits", func(t *testing.T) {
		t.Parallel()
		p := &cursorProvider{}
		// When no rows fit (remaining < smallest row), first == nil means push whole container.
		first, _, didSplit := newSCR().SplitAt(p, 0, 0)
		assert.True(t, didSplit, "split should be signaled")
		assert.Nil(t, first, "when nothing fits, first must be nil (push to next page)")
	})

	t.Run("SplitAt uses the last measured content width", func(t *testing.T) {
		t.Parallel()
		p := &cursorProvider{}
		widthAware := &widthAwareRow{}
		container := &blockContainer{rows: []core.Row{widthAware}}
		scr := newSplittableContainerRow(container)
		scr.SetConfig(cfg)

		narrowCell := &entity.Cell{Width: 100, Height: 100}
		assert.Equal(t, 30.0, scr.GetHeight(p, narrowCell))

		first, rest, didSplit := scr.SplitAt(p, 10, 0)

		assert.True(t, didSplit, "row is 30mm tall at the measured width and must split")
		assert.Nil(t, first, "nothing fits in 10mm at the measured width")
		assert.NotNil(t, rest)
	})
}

// TestSplittableContainerRow_BreakInsideAvoid verifies that a break-inside:avoid
// container is never divided: when it does not fit in the remaining space it is
// pushed whole to the next page (atomic mode, first==nil) even though some of
// its rows would otherwise fit — preventing an orphaned heading/label.
func TestSplittableContainerRow_BreakInsideAvoid(t *testing.T) {
	t.Parallel()
	p := &cursorProvider{}
	cfg := &entity.Config{MaxGridSize: 12}

	newContainer := func(breakInside bool) *splittableContainerRow {
		c := &blockContainer{
			rows:        []core.Row{buildFixedHeightRow(10), buildFixedHeightRow(10), buildFixedHeightRow(10)},
			breakInside: breakInside,
		}
		scr := newSplittableContainerRow(c)
		scr.SetConfig(cfg)
		return scr
	}

	// Sanity: without break-inside the 30mm container greedily splits at 25mm.
	first, _, didSplit := newContainer(false).SplitAt(p, 25, 0)
	require.True(t, didSplit)
	require.NotNil(t, first, "default container splits, placing the rows that fit")

	// With break-inside: avoid it must push the whole box (first == nil).
	first, rest, didSplit := newContainer(true).SplitAt(p, 25, 0)
	require.True(t, didSplit, "container that does not fit must signal a split")
	assert.Nil(t, first, "break-inside:avoid must push the whole box, not place partial rows")
	assert.NotNil(t, rest, "the whole container is carried to the next page")

	// When it fits, no split regardless of break-inside.
	_, _, didSplit = newContainer(true).SplitAt(p, 100, 0)
	assert.False(t, didSplit, "container that fits should not split")
}

// buildFixedHeightRow creates a Row with a fixed pixel height for test purposes.
func buildFixedHeightRow(heightMM float64) core.Row {
	return row.New(heightMM).Add(col.New())
}

type widthAwareRow struct{}

func (r *widthAwareRow) SetConfig(_ *entity.Config) {}

func (r *widthAwareRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "width_aware_row"})
}

func (r *widthAwareRow) Add(...core.Col) core.Row { return r }

func (r *widthAwareRow) GetHeight(_ core.Provider, cell *entity.Cell) float64 {
	if cell.Width <= 200 {
		return 30
	}
	return 5
}

func (r *widthAwareRow) GetColumns() []core.Col { return nil }

func (r *widthAwareRow) WithStyle(_ *props.Cell) core.Row { return r }

func (r *widthAwareRow) Render(core.Provider, entity.Cell) {}
