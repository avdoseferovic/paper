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

	t.Run("when page control overrides top margin, should render rows in shifted content cell", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		top := 0.0
		cfg := &entity.Config{Margins: &entity.Margins{Top: 10}}
		contentCell := entity.Cell{X: cell.X, Y: cell.Y - 10, Width: cell.Width, Height: cell.Height + 10}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, contentCell).Once()
		row.EXPECT().GetHeight(provider, &contentCell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.(interface{ SetPageControl(core.PageControl) }).SetPageControl(core.PageControl{TopMargin: &top})
		sut.Add(row)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})

	t.Run("when page control sets render offset, should shift rows without changing dimensions", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{}
		offsetX := 2.0
		offsetY := 3.0
		contentCell := entity.Cell{X: cell.X + offsetX, Y: cell.Y + offsetY, Width: cell.Width, Height: cell.Height}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, contentCell).Once()
		row.EXPECT().GetHeight(provider, &contentCell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.(interface{ SetPageControl(core.PageControl) }).SetPageControl(core.PageControl{
			RenderOffsetX: &offsetX,
			RenderOffsetY: &offsetY,
		})
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
	t.Run("when page dimensions are known, should draw background image across the physical page", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			Dimensions: &entity.Dimensions{
				Width:  210,
				Height: 297,
			},
			Margins: &entity.Margins{
				Left:   15,
				Top:    12,
				Right:  15,
				Bottom: 25,
			},
			BackgroundImage: &entity.Image{
				Bytes:     []byte{1, 2, 3},
				Extension: extension.Jpg,
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		backgroundCell := entity.Cell{X: -15, Y: -12, Width: 210, Height: 297}

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &backgroundCell, rectProp, cfg.BackgroundImage.Extension).Once()
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
	t.Run("when background image has a page cell, should draw it in physical page coordinates", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left: 15,
				Top:  12,
			},
			BackgroundImage: &entity.Image{
				Bytes:     []byte{1, 2, 3},
				Extension: extension.Png,
				PageCell: &entity.Cell{
					X:      20,
					Y:      30,
					Width:  40,
					Height: 2,
				},
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		backgroundCell := entity.Cell{X: 5, Y: 18, Width: 40, Height: 2}

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &backgroundCell, rectProp, cfg.BackgroundImage.Extension).Once()
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
	t.Run("when positioned background image has object fit, should pass it to provider", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left: 15,
				Top:  12,
			},
			BackgroundImage: &entity.Image{
				Bytes:     []byte{1, 2, 3},
				Extension: extension.Png,
				PageCell:  &entity.Cell{X: 20, Y: 30, Width: 40, Height: 2},
				ObjectFit: "fill",
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		rectProp.ObjectFit = "fill"
		backgroundCell := entity.Cell{X: 5, Y: 18, Width: 40, Height: 2}

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &backgroundCell, rectProp, cfg.BackgroundImage.Extension).Once()
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
	t.Run("when first page background image is set on first page, should draw it after the global background", func(t *testing.T) {
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
			FirstPageBackgroundImage: &entity.Image{
				Bytes:     []byte{4, 5, 6},
				Extension: extension.Png,
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()

		provider := mocks.NewProvider(t)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.BackgroundImage.Bytes, &cell, rectProp, cfg.BackgroundImage.Extension).Once()
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageBackgroundImage.Bytes, &cell, rectProp, cfg.FirstPageBackgroundImage.Extension).Once()
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 1)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when first page background image is set on another page, should not draw it", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			FirstPageBackgroundImage: &entity.Image{
				Bytes:     []byte{4, 5, 6},
				Extension: extension.Png,
			},
		}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 2)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when first page foreground image is set on first page, should draw it after rows", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			FirstPageForegroundImage: &entity.Image{
				Bytes:     []byte{7, 8, 9},
				Extension: extension.Png,
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageForegroundImage.Bytes, &cell, rectProp, cfg.FirstPageForegroundImage.Extension).Once()

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 1)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when first page foreground image has a page cell, should draw it in physical page coordinates", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left: 15,
				Top:  12,
			},
			FirstPageForegroundImage: &entity.Image{
				Bytes:     []byte{7, 8, 9},
				Extension: extension.Png,
				PageCell: &entity.Cell{
					X:      88,
					Y:      164,
					Width:  106,
					Height: 2,
				},
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		foregroundCell := entity.Cell{X: 73, Y: 152, Width: 106, Height: 2}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageForegroundImage.Bytes, &foregroundCell, rectProp, cfg.FirstPageForegroundImage.Extension).Once()

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 1)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when first page foreground images are set on first page, should draw each after rows", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left: 15,
				Top:  12,
			},
			FirstPageForegroundImages: []*entity.Image{
				{
					Bytes:     []byte{1, 2, 3},
					Extension: extension.Png,
					PageCell: &entity.Cell{
						X:      88,
						Y:      164,
						Width:  106,
						Height: 2,
					},
				},
				{
					Bytes:     []byte{4, 5, 6},
					Extension: extension.Jpg,
					PageCell: &entity.Cell{
						X:      90,
						Y:      170,
						Width:  100,
						Height: 3,
					},
				},
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		firstCell := entity.Cell{X: 73, Y: 152, Width: 106, Height: 2}
		secondCell := entity.Cell{X: 75, Y: 158, Width: 100, Height: 3}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageForegroundImages[0].Bytes, &firstCell, rectProp, cfg.FirstPageForegroundImages[0].Extension).Once()
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageForegroundImages[1].Bytes, &secondCell, rectProp, cfg.FirstPageForegroundImages[1].Extension).Once()

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 1)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)
	})
	t.Run("when first page final foreground image is set, should draw it after page number", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left: 15,
				Top:  12,
			},
			FirstPageFinalForegroundImages: []*entity.Image{
				{
					Bytes:     []byte{9, 8, 7},
					Extension: extension.Png,
					PageCell: &entity.Cell{
						X:      88,
						Y:      164,
						Width:  106,
						Height: 2,
					},
				},
			},
		}

		rectProp := &props.Rect{}
		rectProp.MakeValid()
		finalCell := entity.Cell{X: 73, Y: 152, Width: 106, Height: 2}
		var order []string

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Run(func(_ core.Provider, _ entity.Cell) {
			order = append(order, "row")
		}).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)
		provider.EXPECT().AddText("0 / 0", &cell, prop.GetNumberTextProp(cell.Height)).Run(func(_ string, _ *entity.Cell, _ *props.Text) {
			order = append(order, "page-number")
		}).Once()
		provider.EXPECT().AddBackgroundImageFromBytes(cfg.FirstPageFinalForegroundImages[0].Bytes, &finalCell, rectProp, cfg.FirstPageFinalForegroundImages[0].Extension).Run(func(_ []byte, _ *entity.Cell, _ *props.Rect, _ extension.Type) {
			order = append(order, "final-foreground")
		}).Once()

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 1)
		sut.SetConfig(cfg)

		// Act
		sut.Render(provider, cell)

		// Assert
		assert.Equal(t, []string{"row", "page-number", "final-foreground"}, order)
	})
	t.Run("when first page foreground image is set on another page, should not draw it", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		prop := fixture.PageProp()
		prop.Pattern = ""
		cfg := &entity.Config{
			FirstPageForegroundImage: &entity.Image{
				Bytes:     []byte{7, 8, 9},
				Extension: extension.Png,
			},
		}

		provider := mocks.NewProvider(t)
		row := mocks.NewRow(t)
		row.EXPECT().Render(provider, cell).Once()
		row.EXPECT().GetHeight(provider, &cell).Return(10.0).Once()
		row.EXPECT().SetConfig(cfg)

		sut := page.New(prop)
		sut.Add(row)
		setPageIndex(t, sut, 2)
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

func setPageIndex(t *testing.T, p core.Page, index int) {
	t.Helper()
	indexed, ok := p.(interface{ SetPageIndex(int) })
	if !ok {
		t.Fatalf("page should support SetPageIndex")
	}
	indexed.SetPageIndex(index)
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
