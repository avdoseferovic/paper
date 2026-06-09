package row_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/mocks"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("when there is no cols", func(t *testing.T) {
		t.Parallel()
		// Act
		r := row.New(10)

		// Assert
		test.New(t).Assert(r.GetStructure()).Equals("components/rows/new_empty_col.json")
	})
	t.Run("when has component, should retrieve components", func(t *testing.T) {
		t.Parallel()
		// Act
		r := row.New(12).Add(col.New(12))

		// Assert
		test.New(t).Assert(r.GetStructure()).Equals("components/rows/new_filled_col.json")
	})
	t.Run("when has prop, should apply correctly", func(t *testing.T) {
		t.Parallel()
		// Act
		prop := fixture.CellProp()
		r := row.New(12).WithStyle(&prop)

		// Assert
		test.New(t).Assert(r.GetStructure()).Equals("components/rows/new_col_with_prop.json")
	})
}

func TestRow_WithStyleCopiesCallerOwnedStyle(t *testing.T) {
	t.Parallel()

	background := &props.Color{Red: 1, Green: 2, Blue: 3}
	style := &props.Cell{BackgroundColor: background}
	sut := row.New(12).WithStyle(style)

	background.Red = 99

	assert.Equal(t, "RGB(1, 2, 3)", sut.GetStructure().GetData().Details["prop_background_color"])
}

func TestRow_GetHeight(t *testing.T) {
	t.Parallel()
	t.Run("When a row has a column with height 5, should return 5", func(t *testing.T) {
		t.Parallel()
		cell := fixture.CellEntity()

		provider := mocks.NewProvider(t)

		columns := mocks.NewCol(t)
		columns.EXPECT().GetHeight(provider, &cell).Return(5)

		// Act
		r := row.New().Add(columns)

		// Assert
		assert.Equal(t, 5.0, r.GetHeight(provider, &cell))
	})
	t.Run("when auto-height row has style margins, should measure inside margins and include vertical margins", func(t *testing.T) {
		t.Parallel()
		cell := fixture.CellEntity()
		style := &props.Cell{
			MarginTop:    2,
			MarginRight:  3,
			MarginBottom: 5,
			MarginLeft:   7,
		}
		wantCell := entity.Cell{
			X:      cell.X + 7,
			Y:      cell.Y + 2,
			Width:  cell.Width - 10,
			Height: cell.Height - 7,
		}

		provider := mocks.NewProvider(t)

		columns := mocks.NewCol(t)
		columns.EXPECT().
			GetHeight(provider, mock.MatchedBy(func(inner *entity.Cell) bool {
				return inner != nil && *inner == wantCell
			})).
			Return(5.0)

		r := row.New().Add(columns).WithStyle(style)

		assert.Equal(t, 12.0, r.GetHeight(provider, &cell))
	})
}

func TestRow_GetColumns(t *testing.T) {
	t.Parallel()
	t.Run("when GetColumns is called, should return the number of registered columns", func(t *testing.T) {
		t.Parallel()
		// Act
		newCol := []core.Col{col.New(12)}

		r := row.New(10).Add(newCol[0])

		// Assert
		assert.Equal(t, newCol, r.GetColumns())
	})
}

func TestRow_GetStructure(t *testing.T) {
	t.Parallel()
	t.Run("when there is no style, should call provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{
			MaxGridSize: 12,
		}
		cell := fixture.CellEntity()

		provider := mocks.NewProvider(t)
		provider.EXPECT().CreateRow(cell.Height).Once()

		col := mocks.NewCol(t)
		col.EXPECT().Render(provider, cell, true).Once()
		col.EXPECT().SetConfig(cfg).Once()
		col.EXPECT().GetSize().Return(12)

		sut := row.New(cell.Height).Add(col)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when there is style, should call provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{
			MaxGridSize: 12,
		}
		cell := fixture.CellEntity()
		prop := fixture.CellProp()

		provider := mocks.NewProvider(t)
		provider.EXPECT().CreateRow(cell.Height).Once()
		provider.EXPECT().CreateCol(cell.Width, cell.Height, cfg, &prop).Once()

		col := mocks.NewCol(t)
		col.EXPECT().Render(provider, cell, false).Once()
		col.EXPECT().SetConfig(cfg).Once()
		col.EXPECT().GetSize().Return(12)

		sut := row.New(cell.Height).Add(col).WithStyle(&prop)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when config max grid size is invalid, should use default grid for render widths", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{MaxGridSize: 0}
		cell := fixture.CellEntity()
		cell.Width = 120

		provider := mocks.NewProvider(t)
		provider.EXPECT().CreateRow(cell.Height).Once()

		col := mocks.NewCol(t)
		col.EXPECT().Render(provider, mock.MatchedBy(func(inner entity.Cell) bool {
			return inner.Width == 60
		}), true).Once()
		col.EXPECT().SetConfig(cfg)
		col.EXPECT().GetSize().Return(6)

		sut := row.New(cell.Height).Add(col)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
}

func TestRow_RenderManualGridWidths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		gridSize  int
		colSizes  []int
		wantWidth []float64
	}{
		{name: "default grid exact fit", gridSize: 12, colSizes: []int{6, 6}, wantWidth: []float64{60, 60}},
		{name: "custom grid exact fit", gridSize: 8, colSizes: []int{4, 4}, wantWidth: []float64{60, 60}},
		{name: "underflow is preserved", gridSize: 12, colSizes: []int{4, 4}, wantWidth: []float64{40, 40}},
		{name: "overflow is preserved", gridSize: 12, colSizes: []int{8, 8}, wantWidth: []float64{80, 80}},
		{name: "invalid explicit units render as zero width", gridSize: 12, colSizes: []int{0, -1, 6}, wantWidth: []float64{0, 0, 60}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &entity.Config{MaxGridSize: tt.gridSize}
			cell := fixture.CellEntity()
			cell.Width = 120
			cell.Height = 10
			provider := mocks.NewProvider(t)

			for _, width := range tt.wantWidth {
				provider.EXPECT().CreateCol(width, cell.Height, cfg, (*props.Cell)(nil)).Once()
			}
			provider.EXPECT().CreateRow(cell.Height).Once()

			cols := make([]core.Col, 0, len(tt.colSizes))
			for _, size := range tt.colSizes {
				cols = append(cols, col.New(size))
			}
			sut := row.New(cell.Height).Add(cols...)
			sut.SetConfig(cfg)

			sut.Render(provider, cell)
		})
	}
}

func TestRow_SetConfig(t *testing.T) {
	t.Parallel()
	t.Run("should call correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()

		sut := row.New(10)

		// Act
		sut.SetConfig(nil)
	})
}
