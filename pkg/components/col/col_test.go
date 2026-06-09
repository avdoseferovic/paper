package col_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/mocks"
	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/test"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("when size is not defined, should use is as max", func(t *testing.T) {
		t.Parallel()
		// Act
		c := col.New()

		// Assert
		test.New(t).Assert(c.GetStructure()).Equals("components/cols/new_zero_size.json")
	})
	t.Run("when size is defined, should not use max", func(t *testing.T) {
		t.Parallel()
		// Act
		c := col.New(12)

		// Assert
		test.New(t).Assert(c.GetStructure()).Equals("components/cols/new_defined_size.json")
	})
	t.Run("when has component, should retrieve components", func(t *testing.T) {
		t.Parallel()
		// Act
		c := col.New(12).Add(code.NewQr("code"))

		// Assert
		test.New(t).Assert(c.GetStructure()).Equals("components/cols/new_with_components.json")
	})
	t.Run("when has component, should retrieve components", func(t *testing.T) {
		t.Parallel()
		// Act
		prop := fixture.CellProp()
		c := col.New(12).WithStyle(&prop)

		// Assert
		test.New(t).Assert(c.GetStructure()).Equals("components/cols/new_with_props.json")
	})
}

func TestCol_WithStyleCopiesCallerOwnedStyle(t *testing.T) {
	t.Parallel()

	background := &props.Color{Red: 1, Green: 2, Blue: 3}
	style := &props.Cell{BackgroundColor: background}
	sut := col.New(12).WithStyle(style)

	background.Red = 99

	assert.Equal(t, "RGB(1, 2, 3)", sut.GetStructure().GetData().Details["prop_background_color"])
}

func TestCol_GetSize(t *testing.T) {
	t.Parallel()
	t.Run("when size defined in creation, should use it", func(t *testing.T) {
		t.Parallel()
		// Arrange
		c := col.New(12)

		// Act
		size := c.GetSize()

		// Assert
		assert.Equal(t, 12, size)
	})
	t.Run("when size not defined in creation, should use config max grid size", func(t *testing.T) {
		t.Parallel()
		// Arrange
		c := col.New()
		c.SetConfig(&entity.Config{MaxGridSize: 14})

		// Act
		size := c.GetSize()

		// Assert
		assert.Equal(t, 14, size)
	})
	t.Run("when size not defined and config max grid size is invalid, should use default max grid size", func(t *testing.T) {
		t.Parallel()
		// Arrange
		c := col.New()
		c.SetConfig(&entity.Config{MaxGridSize: 0})

		// Act
		size := c.GetSize()

		// Assert
		assert.Equal(t, 12, size)
	})
}

func TestCol_Render(t *testing.T) {
	t.Parallel()
	t.Run("when not createCell, should call provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		cell := fixture.CellEntity()
		style := &props.Cell{}

		provider := mocks.NewProvider(t)

		component := mocks.NewComponent(t)
		component.EXPECT().Render(provider, &cell).Once()
		component.EXPECT().SetConfig(cfg).Once()

		sut := col.New(12).Add(component)
		sut.WithStyle(style)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell, false)
	})
	t.Run("when createCell, should call provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		cell := fixture.CellEntity()
		style := &props.Cell{}

		provider := mocks.NewProvider(t)
		provider.EXPECT().CreateCol(cell.Width, cell.Height, cfg, style).Once()

		component := mocks.NewComponent(t)
		component.EXPECT().Render(provider, &cell).Once()
		component.EXPECT().SetConfig(cfg).Once()

		sut := col.New(12).Add(component)
		sut.WithStyle(style)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell, true)
	})
	t.Run("when createCell and provider supports positioning, should draw at cell origin", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		cell := fixture.CellEntity()
		style := &props.Cell{}
		calls := make([]string, 0, 2)

		baseProvider := mocks.NewProvider(t)
		provider := &positionedProvider{
			Provider: baseProvider,
			setCursor: func(x, y float64) {
				assert.Equal(t, cell.X, x)
				assert.Equal(t, cell.Y, y)
				calls = append(calls, "set_cursor")
			},
		}
		baseProvider.EXPECT().
			CreateCol(cell.Width, cell.Height, cfg, style).
			Run(func(_ float64, _ float64, _ *entity.Config, _ *props.Cell) {
				calls = append(calls, "create_col")
			}).
			Once()

		component := mocks.NewComponent(t)
		component.EXPECT().Render(provider, &cell).Once()
		component.EXPECT().SetConfig(cfg).Once()

		sut := col.New(12).Add(component)
		sut.WithStyle(style)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell, true)

		// Assert
		assert.Equal(t, []string{"set_cursor", "create_col"}, calls)
	})
	t.Run("when createCell and style has margins, should draw and render inside margin box", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := &entity.Config{}
		cell := fixture.CellEntity()
		style := &props.Cell{
			MarginTop:    3,
			MarginRight:  5,
			MarginBottom: 7,
			MarginLeft:   2,
		}
		wantCell := entity.Cell{
			X:      cell.X + 2,
			Y:      cell.Y + 3,
			Width:  cell.Width - 7,
			Height: cell.Height - 10,
		}

		provider := mocks.NewProvider(t)
		provider.EXPECT().CreateCol(wantCell.Width, wantCell.Height, cfg, style).Once()

		component := mocks.NewComponent(t)
		component.EXPECT().
			Render(provider, mock.MatchedBy(func(got *entity.Cell) bool {
				return got != nil && *got == wantCell
			})).
			Once()
		component.EXPECT().SetConfig(cfg).Once()

		sut := col.New(12).Add(component)
		sut.WithStyle(style)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell, true)
	})
}

type positionedProvider struct {
	*mocks.Provider
	setCursor func(x, y float64)
}

func (p *positionedProvider) SetCursor(x, y float64) {
	p.setCursor(x, y)
}

func TestCol_GetHeight(t *testing.T) {
	t.Parallel()
	t.Run("when column has two components, should return the largest", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		cfg := &entity.Config{MaxGridSize: 12}

		provider := mocks.NewProvider(t)

		component := mocks.NewComponent(t)
		component.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		component.EXPECT().SetConfig(cfg)

		component2 := mocks.NewComponent(t)
		component2.EXPECT().GetHeight(provider, &cell).Return(15.0)
		component2.EXPECT().SetConfig(cfg)

		sut := col.New(12).Add(component, component2)
		sut.SetConfig(cfg)

		// Act
		height := sut.GetHeight(provider, &cell)

		// Assert
		assert.Equal(t, 15.0, height)
	})
	t.Run("when config max grid size is invalid, should use default grid for height measurement", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		cell.Width = 120
		cfg := &entity.Config{MaxGridSize: 0}

		provider := mocks.NewProvider(t)

		component := mocks.NewComponent(t)
		component.EXPECT().
			GetHeight(provider, mock.MatchedBy(func(inner *entity.Cell) bool {
				return inner != nil && inner.Width == 60
			})).
			Return(10.0)
		component.EXPECT().SetConfig(cfg)

		sut := col.New(6).Add(component)
		sut.SetConfig(cfg)

		// Act
		height := sut.GetHeight(provider, &cell)

		// Assert
		assert.Equal(t, 10.0, height)
	})
	t.Run("when styled column has margins, should measure inside margins and include vertical margins", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		cell.Width = 120
		cfg := &entity.Config{MaxGridSize: 12}
		style := &props.Cell{
			MarginTop:    2,
			MarginRight:  3,
			MarginBottom: 4,
			MarginLeft:   5,
		}
		wantCell := entity.Cell{
			X:      cell.X + 5,
			Y:      cell.Y + 2,
			Width:  60 - 8,
			Height: cell.Height - 6,
		}

		provider := mocks.NewProvider(t)

		component := mocks.NewComponent(t)
		component.EXPECT().
			GetHeight(provider, mock.MatchedBy(func(inner *entity.Cell) bool {
				return inner != nil && *inner == wantCell
			})).
			Return(10.0).
			Once()
		component.EXPECT().SetConfig(cfg)

		sut := col.New(6).Add(component).WithStyle(style)
		sut.SetConfig(cfg)

		// Act
		height := sut.GetHeight(provider, &cell)

		// Assert
		assert.Equal(t, 16.0, height)
	})
}
