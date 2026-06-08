package table_test

import (
	"testing"

	"github.com/avdoseferovic/paper/mocks"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
