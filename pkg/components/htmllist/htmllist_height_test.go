package htmllist_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestHTMLList_GetHeightMeasuresItemContent(t *testing.T) {
	t.Parallel()

	provider := mocks.NewProvider(t)
	provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
		mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(3).Maybe()

	content := text.New("a paragraph that wraps over multiple lines")
	content.SetConfig(defaultConfig())
	items := []htmllist.Item{{Content: content}}
	l := htmllist.New(items)
	l.SetConfig(defaultConfig())

	withContent := l.GetHeight(provider, &entity.Cell{Width: 100, Height: 200})

	empty := htmllist.New([]htmllist.Item{{}})
	empty.SetConfig(defaultConfig())
	singleLine := empty.GetHeight(provider, &entity.Cell{Width: 100, Height: 200})

	// Wrapped content reports more height than the single-line fallback.
	assert.Greater(t, withContent, singleLine)
}

func TestHTMLList_GetHeightIncludesSubList(t *testing.T) {
	t.Parallel()

	provider := mocks.NewProvider(t)
	provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
		mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()

	sub := htmllist.New([]htmllist.Item{{}, {}})
	sub.SetConfig(defaultConfig())
	nested := htmllist.New([]htmllist.Item{{SubList: sub}})
	nested.SetConfig(defaultConfig())

	flat := htmllist.New([]htmllist.Item{{}})
	flat.SetConfig(defaultConfig())

	cell := &entity.Cell{Width: 100, Height: 200}

	assert.Greater(t, nested.GetHeight(provider, cell), flat.GetHeight(provider, cell))
}

func TestHTMLList_GetStructureExposesStartAndReversed(t *testing.T) {
	t.Parallel()

	items := []htmllist.Item{{}}
	l := htmllist.New(items, htmllist.Prop{Style: htmllist.Decimal, Start: 4, Reversed: true})
	l.SetConfig(defaultConfig())

	details := l.GetStructure().GetData().Details

	assert.Equal(t, 4, details["start"])
	assert.Equal(t, true, details["reversed"])
}

func TestHTMLList_GutterWidth(t *testing.T) {
	t.Parallel()

	t.Run("explicit gutter width wins", func(t *testing.T) {
		t.Parallel()

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()

		l := htmllist.New([]htmllist.Item{{}}, htmllist.Prop{Style: htmllist.Decimal, GutterWidth: 42})
		l.SetConfig(defaultConfig())

		// GetHeight exercises gutterWidth; an explicit width avoids measurement.
		h := l.GetHeight(provider, &entity.Cell{Width: 100, Height: 200})

		assert.Greater(t, h, 0.0)
	})

	t.Run("style none renders without marker gutter", func(t *testing.T) {
		t.Parallel()

		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()

		l := htmllist.New([]htmllist.Item{{}}, htmllist.Prop{Style: htmllist.None})
		l.SetConfig(defaultConfig())

		h := l.GetHeight(provider, &entity.Cell{Width: 100, Height: 200})

		assert.Greater(t, h, 0.0)
	})
}
