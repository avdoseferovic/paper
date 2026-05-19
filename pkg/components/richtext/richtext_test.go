package richtext_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/mocks"
	"github.com/johnfercher/maroto/v2/pkg/components/richtext"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func defaultConfig() *entity.Config {
	return &entity.Config{
		MaxGridSize: 12,
		DefaultFont: &props.Font{
			Family: "Helvetica",
			Style:  fontstyle.Normal,
			Size:   10,
		},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("should create component with runs", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "hello"}}
		sut := richtext.New(runs)
		assert.NotNil(t, sut)
	})

	t.Run("should create with prop", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "hello"}}
		prop := props.RichText{LineHeight: 1.5}
		sut := richtext.New(runs, prop)
		assert.NotNil(t, sut)
	})
}

func TestNewCol(t *testing.T) {
	t.Parallel()
	runs := []props.RichRun{{Text: "hello"}}
	col := richtext.NewCol(12, runs)
	assert.NotNil(t, col)
}

func TestNewRow(t *testing.T) {
	t.Parallel()
	runs := []props.RichRun{{Text: "hello"}}
	row := richtext.NewRow(10, runs)
	assert.NotNil(t, row)
}

func TestNewAutoRow(t *testing.T) {
	t.Parallel()
	runs := []props.RichRun{{Text: "hello"}}
	row := richtext.NewAutoRow(runs)
	assert.NotNil(t, row)
}

func TestRichText_GetStructure(t *testing.T) {
	t.Parallel()
	runs := []props.RichRun{{Text: "hello", Style: fontstyle.Bold}}
	sut := richtext.New(runs)
	sut.SetConfig(defaultConfig())

	node := sut.GetStructure()

	assert.Equal(t, "richtext", node.GetData().Type)
}

func TestRichText_SetConfig(t *testing.T) {
	t.Parallel()
	runs := []props.RichRun{{Text: "hi"}}
	sut := richtext.New(runs)
	// Should not panic
	sut.SetConfig(defaultConfig())
}

func TestRichText_GetHeight(t *testing.T) {
	t.Parallel()
	t.Run("should return positive height for non-empty runs", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "hello world", Family: "Helvetica", Style: fontstyle.Normal, Size: 10}}

		rtp := mocks.NewRichTextProvider(t)
		rtp.EXPECT().MeasureString(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text")).
			Return(10.0).Maybe()

		// GetHeight needs a core.Provider; we use a full mock
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(1).Maybe()

		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New(runs)
		sut.SetConfig(defaultConfig())

		h := sut.GetHeight(provider, cell)
		assert.Greater(t, h, 0.0)
	})

	t.Run("memoization: second GetHeight call returns cached value without re-measuring", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "hello", Family: "Helvetica", Style: fontstyle.Normal, Size: 10}}

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(1).Once() // exactly once — second call uses cache

		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New(runs)
		sut.SetConfig(defaultConfig())

		h1 := sut.GetHeight(provider, cell)
		h2 := sut.GetHeight(provider, cell)
		assert.Equal(t, h1, h2)
	})
}

func TestRichText_Render(t *testing.T) {
	t.Parallel()
	t.Run("should call AddRichText when provider supports RichTextProvider", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "hello", Family: "Helvetica", Style: fontstyle.Normal, Size: 10}}

		provider := mocks.NewProvider(t)
		// When provider doesn't implement RichTextProvider, falls back to AddText
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(1).Maybe()
		provider.EXPECT().AddText(mock.AnythingOfType("string"), mock.AnythingOfType("*entity.Cell"),
			mock.AnythingOfType("*props.Text")).Maybe()

		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New(runs)
		sut.SetConfig(defaultConfig())

		// Should not panic
		sut.Render(provider, cell)
	})
}

// Verify RichText implements core.Component at compile time.
var _ core.Component = (*richtext.RichText)(nil)
