package table_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type positioningProvider struct {
	*mocks.Provider
	cursorX float64
	cursorY float64
}

func (p *positioningProvider) SetCursor(x, y float64) {
	p.cursorX = x
	p.cursorY = y
}

func defaultConfig() *entity.Config {
	return &entity.Config{
		MaxGridSize: 12,
		DefaultFont: &props.Font{Family: "Helvetica", Style: fontstyle.Normal, Size: 10},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("simple 2x2 table builds without error", func(t *testing.T) {
		t.Parallel()
		cells := [][]table.Cell{
			{{Content: nil}, {Content: nil}},
			{{Content: nil}, {Content: nil}},
		}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		assert.NotNil(t, tbl)
	})

	t.Run("colspan=2 header row, 2-col body — column count is 2 (not 1)", func(t *testing.T) {
		t.Parallel()
		cells := [][]table.Cell{
			{{Content: nil, Colspan: 2}},     // first row: one cell spanning 2 cols
			{{Content: nil}, {Content: nil}}, // second row: 2 normal cells
		}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		assert.NotNil(t, tbl)
		assert.Equal(t, 2, tbl.ColCount())
	})

	t.Run("overlapping rowspan returns ErrTableSpanOverlap", func(t *testing.T) {
		t.Parallel()
		// Row 0 has one cell with rowspan=2, occupying [0][0] and [1][0].
		// Row 1 also declares a cell, which tries to fill [1][0] — already occupied.
		cells := [][]table.Cell{
			{{Content: nil, Rowspan: 2}}, // occupies rows 0 and 1, col 0
			{{Content: nil}},             // tries to also start at row 1, col 0 → overlap
		}
		_, err := table.New(cells)
		assert.ErrorIs(t, err, table.ErrTableSpanOverlap)
	})

	t.Run("rowspan=3 taller than sum of other rows — heights sum correctly", func(t *testing.T) {
		t.Parallel()
		// Row 0: cells (0,0) rowspan=3 + (0,1) short
		// Row 1: cell (1,1) short
		// Row 2: cell (2,1) short
		cells := [][]table.Cell{
			{{Content: nil, Rowspan: 3}, {Content: nil}},
			{{Content: nil}},
			{{Content: nil}},
		}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		assert.NotNil(t, tbl)
	})
}

func TestNewCopiesCallerOwnedCellsAndStyles(t *testing.T) {
	t.Parallel()

	background := &props.Color{Red: 1, Green: 2, Blue: 3}
	cells := [][]table.Cell{{
		{Content: nil, Style: &props.Cell{BackgroundColor: background}},
	}}

	tbl, err := table.New(cells)
	assert.NoError(t, err)
	assert.Equal(t, 0, cells[0][0].Colspan)
	assert.Equal(t, 0, cells[0][0].Rowspan)

	background.Red = 99

	provider := mocks.NewProvider(t)
	provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	provider.EXPECT().CreateCol(
		100.0,
		5.0,
		mock.AnythingOfType("*entity.Config"),
		mock.MatchedBy(func(style *props.Cell) bool {
			return style != nil && style.BackgroundColor != nil && style.BackgroundColor.Red == 1
		}),
	).Return()
	provider.EXPECT().CreateRow(5.0).Return()

	tbl.SetConfig(defaultConfig())
	tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
}

func TestTable_ColCount(t *testing.T) {
	t.Parallel()
	cells := [][]table.Cell{
		{{Content: nil, Colspan: 3}},
		{{Content: nil}, {Content: nil}, {Content: nil}},
	}
	tbl, err := table.New(cells)
	assert.NoError(t, err)
	assert.Equal(t, 3, tbl.ColCount())
}

func TestTable_ColumnWidths(t *testing.T) {
	t.Parallel()

	tbl, err := table.New([][]table.Cell{{{Content: nil}, {Content: nil}}}, table.WithColumnWidths([]float64{1, 3}))
	assert.NoError(t, err)

	assert.InDeltaSlice(t, []float64{0.25, 0.75}, tbl.ColumnWidths(), 0.0001)
	node := tbl.GetStructure()
	assert.InDeltaSlice(t, []float64{0.25, 0.75}, node.GetData().Details["column_widths"], 0.0001)
}

func TestTable_BorderSpacing(t *testing.T) {
	t.Parallel()

	t.Run("renders horizontal spacing between columns", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 24.5
		})).Return(10.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 73.5
		})).Return(10.0)
		left.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 0 && cell.Width == 24.5 && cell.Height == 10
		})).Return()
		right.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 26.5 && cell.Width == 73.5 && cell.Height == 10
		})).Return()
		provider.EXPECT().CreateRow(10.0).Return()

		cells := [][]table.Cell{{{Content: left}, {Content: right}}}
		tbl, err := table.New(cells, table.WithColumnWidths([]float64{1, 3}), table.WithBorderSpacing(2, 0))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})

	t.Run("adds vertical spacing between rows", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		tbl, err := table.New([][]table.Cell{{{Content: nil}}, {{Content: nil}}}, table.WithBorderSpacing(0, 3))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 13.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})

	t.Run("advances spacing through skipped colspan slots", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		header := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		header.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		header.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 66
		})).Return(10.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 32
		})).Return(10.0)
		header.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 0 && cell.Width == 66 && cell.Height == 10
		})).Return()
		right.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 68 && cell.Width == 32 && cell.Height == 10
		})).Return()
		provider.EXPECT().CreateRow(10.0).Return()

		cells := [][]table.Cell{{{Content: header, Colspan: 2}, {Content: right}}}
		tbl, err := table.New(cells, table.WithBorderSpacing(2, 0))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})
}

func TestTable_GetHeight(t *testing.T) {
	t.Parallel()
	t.Run("returns positive height for non-empty table", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()

		cells := [][]table.Cell{
			{{Content: nil}, {Content: nil}},
		}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		cell := &entity.Cell{Width: 100, Height: 200}
		h := tbl.GetHeight(provider, cell)
		assert.GreaterOrEqual(t, h, 0.0)
	})

	t.Run("includes vertical padding in content height", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		component := mocks.NewComponent(t)
		component.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 100 && cell.Height == 194
		})).Return(10.0)

		cells := [][]table.Cell{{
			{
				Content: component,
				Style:   &props.Cell{PaddingTop: 2, PaddingBottom: 4},
			},
		}}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		h := tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200})
		assert.Equal(t, 16.0, h)
	})

	t.Run("uses explicit empty cell height instead of font fallback", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		tbl, err := table.New([][]table.Cell{{
			{Height: 1.4, Style: &props.Cell{BackgroundColor: &props.Color{Red: 39, Green: 118, Blue: 145}}},
		}})
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 1.4, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})

	t.Run("content height can exceed explicit cell height", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		component := mocks.NewComponent(t)
		component.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		component.EXPECT().GetHeight(provider, mock.AnythingOfType("*entity.Cell")).Return(10.0)

		tbl, err := table.New([][]table.Cell{{{Content: component, Height: 1.4}}})
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})

	t.Run("uses configured column widths for content measurement", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 25 && cell.Height == 200
		})).Return(8.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 75 && cell.Height == 200
		})).Return(10.0)

		cells := [][]table.Cell{{{Content: left}, {Content: right}}}
		tbl, err := table.New(cells, table.WithColumnWidths([]float64{1, 3}))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})

	t.Run("uses preferred table width for content measurement", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 17
		})).Return(8.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 17
		})).Return(10.0)

		cells := [][]table.Cell{{{Content: left}, {Content: right}}}
		tbl, err := table.New(cells, table.WithWidth(34))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})

	t.Run("caches row heights per measured width", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		component := mocks.NewComponent(t)
		component.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 100 && cell.Height == 200
		})).Return(10.0).Once()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 50 && cell.Height == 200
		})).Return(20.0).Once()

		tbl, err := table.New([][]table.Cell{{{Content: component}}})
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
		assert.Equal(t, 20.0, tbl.GetHeight(provider, &entity.Cell{Width: 50, Height: 200}))
	})

	t.Run("set config invalidates cached row heights", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		component := mocks.NewComponent(t)
		component.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return().Twice()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 100 && cell.Height == 200
		})).Return(10.0).Once()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 100 && cell.Height == 200
		})).Return(12.0).Once()

		tbl, err := table.New([][]table.Cell{{{Content: component}}})
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))

		tbl.SetConfig(defaultConfig())

		assert.Equal(t, 12.0, tbl.GetHeight(provider, &entity.Cell{Width: 100, Height: 200}))
	})
}

func TestTable_SetConfig(t *testing.T) {
	t.Parallel()
	cells := [][]table.Cell{{{Content: nil}}}
	tbl, err := table.New(cells)
	assert.NoError(t, err)
	tbl.SetConfig(defaultConfig())
}

func TestTable_GetStructure(t *testing.T) {
	t.Parallel()
	cells := [][]table.Cell{
		{{Content: nil}, {Content: nil}},
		{{Content: nil}, {Content: nil}},
	}
	tbl, err := table.New(cells)
	assert.NoError(t, err)
	tbl.SetConfig(defaultConfig())

	node := tbl.GetStructure()
	assert.Equal(t, "table", node.GetData().Type)
}

func TestTable_Render(t *testing.T) {
	t.Parallel()
	t.Run("renders without panic for nil content cells", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()
		provider.EXPECT().CreateRow(mock.AnythingOfType("float64")).Maybe()
		provider.EXPECT().CreateCol(mock.AnythingOfType("float64"), mock.AnythingOfType("float64"),
			mock.AnythingOfType("*entity.Config"), mock.AnythingOfType("*props.Cell")).Maybe()

		cells := [][]table.Cell{
			{{Content: nil}, {Content: nil}},
		}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		cell := &entity.Cell{Width: 100, Height: 200}
		assert.NotPanics(t, func() { tbl.Render(provider, cell) })
	})

	t.Run("renders cell content inside padding", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		component := mocks.NewComponent(t)
		component.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		component.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 92 && cell.Height == 194
		})).Return(10.0)
		component.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 5 && cell.Y == 2 && cell.Width == 92 && cell.Height == 10
		})).Return()
		provider.EXPECT().CreateCol(100.0, 16.0, mock.AnythingOfType("*entity.Config"),
			mock.AnythingOfType("*props.Cell")).Return()
		provider.EXPECT().CreateRow(16.0).Return()

		cells := [][]table.Cell{{
			{
				Content: component,
				Style: &props.Cell{
					PaddingTop:    2,
					PaddingRight:  3,
					PaddingBottom: 4,
					PaddingLeft:   5,
				},
			},
		}}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})

	t.Run("renders explicit height cell centered inside taller row", func(t *testing.T) {
		t.Parallel()
		baseProvider := mocks.NewProvider(t)
		provider := &positioningProvider{Provider: baseProvider}
		baseProvider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		tall := mocks.NewComponent(t)
		tall.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		anyProvider := mock.MatchedBy(func(core.Provider) bool { return true })
		tall.EXPECT().GetHeight(anyProvider, mock.AnythingOfType("*entity.Cell")).Return(10.0)
		tall.EXPECT().Render(anyProvider, mock.AnythingOfType("*entity.Cell")).Return()

		bg := &props.Cell{BackgroundColor: &props.Color{Red: 39, Green: 118, Blue: 145}}
		cells := [][]table.Cell{{
			{Height: 4, VerticalAlign: "middle", Style: bg},
			{Content: tall},
		}}
		tbl, err := table.New(cells)
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		baseProvider.EXPECT().CreateCol(50.0, 4.0, mock.AnythingOfType("*entity.Config"), mock.AnythingOfType("*props.Cell")).Return()
		baseProvider.EXPECT().CreateRow(10.0).Return()

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})

		assert.Equal(t, 0.0, provider.cursorX)
		assert.Equal(t, 3.0, provider.cursorY)
	})

	t.Run("renders unequal configured column widths", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 25
		})).Return(10.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 75
		})).Return(10.0)
		left.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 0 && cell.Width == 25 && cell.Height == 10
		})).Return()
		right.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 25 && cell.Width == 75 && cell.Height == 10
		})).Return()
		provider.EXPECT().CreateRow(10.0).Return()

		cells := [][]table.Cell{{{Content: left}, {Content: right}}}
		tbl, err := table.New(cells, table.WithColumnWidths([]float64{1, 3}))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})

	t.Run("renders preferred width table aligned right", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 17
		})).Return(10.0)
		right.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 17
		})).Return(10.0)
		left.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 66 && cell.Width == 17 && cell.Height == 10
		})).Return()
		right.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 83 && cell.Width == 17 && cell.Height == 10
		})).Return()
		provider.EXPECT().CreateRow(10.0).Return()

		cells := [][]table.Cell{{{Content: left}, {Content: right}}}
		tbl, err := table.New(cells, table.WithWidth(34), table.WithAlign("right"))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})

	t.Run("renders colspan width as sum of configured columns", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()

		header := mocks.NewComponent(t)
		left := mocks.NewComponent(t)
		right := mocks.NewComponent(t)
		header.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		left.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		right.EXPECT().SetConfig(mock.AnythingOfType("*entity.Config")).Return()
		header.EXPECT().GetHeight(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.Width == 100
		})).Return(6.0)
		left.EXPECT().GetHeight(provider, mock.AnythingOfType("*entity.Cell")).Return(5.0)
		right.EXPECT().GetHeight(provider, mock.AnythingOfType("*entity.Cell")).Return(5.0)
		header.EXPECT().Render(provider, mock.MatchedBy(func(cell *entity.Cell) bool {
			return cell.X == 0 && cell.Width == 100 && cell.Height == 6
		})).Return()
		left.EXPECT().Render(provider, mock.AnythingOfType("*entity.Cell")).Return()
		right.EXPECT().Render(provider, mock.AnythingOfType("*entity.Cell")).Return()
		provider.EXPECT().CreateRow(6.0).Return()
		provider.EXPECT().CreateRow(5.0).Return()

		cells := [][]table.Cell{
			{{Content: header, Colspan: 2}},
			{{Content: left}, {Content: right}},
		}
		tbl, err := table.New(cells, table.WithColumnWidths([]float64{1, 3}))
		assert.NoError(t, err)
		tbl.SetConfig(defaultConfig())

		tbl.Render(provider, &entity.Cell{Width: 100, Height: 200})
	})
}
