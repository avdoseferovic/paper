package page_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/internal/test"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("when prop is not sent, should use default", func(t *testing.T) {
		t.Parallel()
		// Act
		sut := page.New()

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("components/lines/new_page_default_prop.json")
	})
	t.Run("when prop is sent, should use the provided", func(t *testing.T) {
		t.Parallel()
		// Act
		sut := page.New(fixture.PageProp())

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("components/lines/new_page_custom_prop.json")
	})
	t.Run("when prop is sent and there is rows, should use the provided", func(t *testing.T) {
		t.Parallel()
		// Act
		sut := page.New(fixture.PageProp())

		row := image.NewFromFileRow(10, "path")
		sut.Add(row)

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("components/lines/new_page_custom_prop_and_with_rows.json")
	})
}

func TestPage_Render(t *testing.T) {
	t.Parallel()
	t.Run("when there is no background image and there is no page pattern, should call row render correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})

	t.Run("when there is background image and there is no page pattern, should call row render and provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			BackgroundImage: &entity.Image{
				Bytes:     []byte{1, 2, 3},
				Extension: extension.Jpg,
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &cell, rectProp, cfg.BackgroundImage.Extension).Once()
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when there is background image and there is page pattern, should call row render and provider correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		cfg := &entity.Config{
			BackgroundImage: &entity.Image{
				Bytes:     []byte{1, 2, 3},
				Extension: extension.Jpg,
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &cell, rectProp, cfg.BackgroundImage.Extension).Once()
		provider.EXPECT().AddText("0 / 0", &cell, prop.GetNumberTextProp(cell.Height)).Once()
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
}

func TestPage_SetNumber(t *testing.T) {
	t.Parallel()
	t.Run("when called set number, should set correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := page.New()

		// Act
		sut.SetNumber(1, 2)

		// Assert
		assert.Equal(t, 1, sut.GetNumber())
	})
}

func TestPage_GetRows(t *testing.T) {
	t.Parallel()
	t.Run("when called get rows, should return rows correctly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		row := mocks.NewRow(t)

		sut := page.New()
		sut.Add(row)

		// Act
		rows := sut.GetRows()

		// Assert
		assert.Equal(t, []core.Row{row}, rows)
	})
}
