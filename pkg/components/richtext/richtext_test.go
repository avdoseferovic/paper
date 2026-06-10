package richtext_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
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

	t.Run("counts mixed inline runs as one paragraph instead of one line per run", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{
			{Text: "This paragraph combines "},
			{Text: "bold", Style: fontstyle.Bold},
			{Text: ", "},
			{Text: "italic", Style: fontstyle.Italic},
			{Text: ", "},
			{Text: "subscript", VerticalAlign: "sub"},
			{Text: ", "},
			{Text: "superscript", VerticalAlign: "super"},
			{Text: ", and text."},
		}

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(1).Once()

		cell := &entity.Cell{Width: 500, Height: 100}
		sut := richtext.New(runs)
		sut.SetConfig(defaultConfig())

		assert.Equal(t, 5.0, sut.GetHeight(provider, cell))
	})

	t.Run("nowrap height ignores automatic wrapping", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "one two three", Family: "Helvetica", Style: fontstyle.Normal, Size: 10}}

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(4).Maybe()

		cell := &entity.Cell{Width: 10, Height: 100}
		sut := richtext.New(runs, props.RichText{WhiteSpace: "nowrap"})
		sut.SetConfig(defaultConfig())

		assert.Equal(t, 5.0, sut.GetHeight(provider, cell))
	})

	t.Run("pre-line height preserves explicit line breaks", func(t *testing.T) {
		t.Parallel()
		runs := []props.RichRun{{Text: "one\ntwo", Family: "Helvetica", Style: fontstyle.Normal, Size: 10}}

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"), mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64")).Return(1).Maybe()

		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New(runs, props.RichText{WhiteSpace: "pre-line"})
		sut.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, sut.GetHeight(provider, cell))
	})

	t.Run("uses rich text measurer when provider supports exact layout", func(t *testing.T) {
		t.Parallel()
		provider := &richTextMeasureProviderFake{
			richTextProviderFake: richTextProviderFake{Provider: mocks.NewProvider(t)},
			height:               17,
		}
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(
			mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"),
			mock.AnythingOfType("float64"),
		).Return(1).Maybe()

		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New([]props.RichRun{{Text: "wide", LetterSpacing: 20}})
		sut.SetConfig(defaultConfig())

		assert.Equal(t, 17.0, sut.GetHeight(provider, cell))
		assert.Equal(t, 1, provider.calls)
	})

	t.Run("caches exact measured height per width", func(t *testing.T) {
		t.Parallel()
		provider := &richTextMeasureProviderFake{
			richTextProviderFake: richTextProviderFake{Provider: mocks.NewProvider(t)},
			heightByWidth:        map[float64]float64{50: 10, 20: 25},
		}

		sut := richtext.New([]props.RichRun{{Text: "wrap sensitive"}})
		sut.SetConfig(defaultConfig())

		assert.Equal(t, 10.0, sut.GetHeight(provider, &entity.Cell{Width: 50, Height: 100}))
		assert.Equal(t, 10.0, sut.GetHeight(provider, &entity.Cell{Width: 50, Height: 100}))
		assert.Equal(t, 25.0, sut.GetHeight(provider, &entity.Cell{Width: 20, Height: 100}))
		assert.Equal(t, 2, provider.calls)
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

	t.Run("resolves empty run font fields before rich text provider render", func(t *testing.T) {
		t.Parallel()
		provider := &richTextProviderFake{Provider: mocks.NewProvider(t)}
		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New([]props.RichRun{{Text: "hello"}})
		sut.SetConfig(defaultConfig())

		sut.Render(provider, cell)

		require.Len(t, provider.runs, 1)
		assert.Equal(t, "Helvetica", provider.runs[0].Family)
		assert.Equal(t, fontstyle.Normal, provider.runs[0].Style)
		assert.Equal(t, 10.0, provider.runs[0].Size)
	})

	t.Run("passes white-space and first-line indent props to rich text provider", func(t *testing.T) {
		t.Parallel()
		provider := &richTextProviderFake{Provider: mocks.NewProvider(t)}
		cell := &entity.Cell{Width: 50, Height: 100}
		sut := richtext.New([]props.RichRun{{Text: "hello"}}, props.RichText{
			WhiteSpace:      "pre-line",
			FirstLineIndent: 5,
		})
		sut.SetConfig(defaultConfig())

		sut.Render(provider, cell)

		require.NotNil(t, provider.prop)
		assert.Equal(t, "pre-line", provider.prop.WhiteSpace)
		assert.Equal(t, 5.0, provider.prop.FirstLineIndent)
	})
}

// Verify RichText implements core.Component at compile time.
var _ core.Component = (*richtext.RichText)(nil)

type richTextProviderFake struct {
	*mocks.Provider
	runs []props.RichRun
	prop *props.RichText
}

func (f *richTextProviderFake) MeasureString(_ string, _ *props.Text) float64 {
	return 0
}

func (f *richTextProviderFake) AddTextAt(_, _ float64, _ string, _ *props.Text) {}

func (f *richTextProviderFake) AddRichText(runs []props.RichRun, _ *entity.Cell, prop *props.RichText) {
	f.runs = runs
	f.prop = prop
}

type richTextMeasureProviderFake struct {
	richTextProviderFake
	height        float64
	heightByWidth map[float64]float64
	calls         int
}

func (f *richTextMeasureProviderFake) MeasureRichText(_ []props.RichRun, cell *entity.Cell, _ *props.RichText) float64 {
	f.calls++
	if f.heightByWidth != nil {
		return f.heightByWidth[cell.Width]
	}
	return f.height
}
