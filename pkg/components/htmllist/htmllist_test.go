package htmllist_test

import (
	"testing"

	"github.com/avdoseferovic/paper/mocks"
	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func defaultConfig() *entity.Config {
	return &entity.Config{
		MaxGridSize: 12,
		DefaultFont: &props.Font{Family: "Helvetica", Style: fontstyle.Normal, Size: 10},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("creates bullet list", func(t *testing.T) {
		t.Parallel()
		items := []htmllist.Item{{Content: nil}}
		l := htmllist.New(items)
		assert.NotNil(t, l)
	})

	t.Run("creates with prop", func(t *testing.T) {
		t.Parallel()
		items := []htmllist.Item{{Content: nil}}
		prop := htmllist.Prop{Style: htmllist.Decimal}
		l := htmllist.New(items, prop)
		assert.NotNil(t, l)
	})
}

func TestHTMLList_GetStructure(t *testing.T) {
	t.Parallel()
	items := []htmllist.Item{{Content: nil}}
	l := htmllist.New(items)
	l.SetConfig(defaultConfig())
	node := l.GetStructure()
	assert.Equal(t, "htmllist", node.GetData().Type)
}

func TestHTMLList_SetConfig(t *testing.T) {
	t.Parallel()
	items := []htmllist.Item{{Content: nil}}
	l := htmllist.New(items)
	l.SetConfig(defaultConfig())
}

func TestHTMLList_GetHeight(t *testing.T) {
	t.Parallel()
	t.Run("returns positive height for non-empty items", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()

		items := []htmllist.Item{{Content: nil}, {Content: nil}}
		l := htmllist.New(items)
		l.SetConfig(defaultConfig())

		cell := &entity.Cell{Width: 100, Height: 200}
		h := l.GetHeight(provider, cell)
		assert.Greater(t, h, 0.0)
	})
}

func TestHTMLList_Render(t *testing.T) {
	t.Parallel()
	t.Run("renders without panic", func(t *testing.T) {
		t.Parallel()
		provider := mocks.NewProvider(t)
		provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
		provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
			mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()
		provider.EXPECT().AddText(mock.AnythingOfType("string"), mock.AnythingOfType("*entity.Cell"),
			mock.AnythingOfType("*props.Text")).Maybe()

		items := []htmllist.Item{{Content: nil}, {Content: nil}}
		l := htmllist.New(items)
		l.SetConfig(defaultConfig())

		cell := &entity.Cell{Width: 100, Height: 200}
		assert.NotPanics(t, func() { l.Render(provider, cell) })
	})
}

func TestMarkerFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		style    htmllist.StyleType
		idx      int
		expected string
	}{
		{htmllist.Bullet, 0, "•"},
		{htmllist.Decimal, 0, "1."},
		{htmllist.Decimal, 9, "10."},
		{htmllist.DecimalCircle, 0, "1"},
		{htmllist.DecimalCircle, 9, "10"},
		{htmllist.LowerAlpha, 0, "a."},
		{htmllist.UpperAlpha, 0, "A."},
		{htmllist.LowerRoman, 0, "i."},
		{htmllist.LowerRoman, 3, "iv."},
		{htmllist.UpperRoman, 0, "I."},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			got := htmllist.FormatMarker(tc.style, tc.idx)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// Verify HTMLList implements core.Component at compile time.
var _ core.Component = (*htmllist.HTMLList)(nil)

// shapeProviderFake embeds mocks.Provider and adds a recording DrawFilledCircle.
type shapeProviderFake struct {
	*mocks.Provider
	drawCalls  int
	lastCircle *entity.Cell
}

func (s *shapeProviderFake) DrawFilledCircle(cell *entity.Cell, _ *props.Color) {
	s.drawCalls++
	copied := cell.Copy()
	s.lastCircle = &copied
}

func TestHTMLList_Render_DecimalCircle_CallsDrawFilledCircle(t *testing.T) {
	t.Parallel()
	mockProv := mocks.NewProvider(t)
	mockProv.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	mockProv.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
		mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()
	mockProv.EXPECT().AddText(mock.AnythingOfType("string"), mock.AnythingOfType("*entity.Cell"),
		mock.AnythingOfType("*props.Text")).Maybe()

	provider := &shapeProviderFake{Provider: mockProv}

	items := []htmllist.Item{{Content: nil}, {Content: nil}}
	l := htmllist.New(items, htmllist.Prop{Style: htmllist.DecimalCircle})
	l.SetConfig(defaultConfig())

	cell := &entity.Cell{Width: 100, Height: 200}
	l.Render(provider, cell)

	assert.Equal(t, 2, provider.drawCalls, "DrawFilledCircle should be called once per list item")
}

func TestHTMLList_Render_DecimalCircle_UsesReadableMarkerBounds(t *testing.T) {
	t.Parallel()
	mockProv := mocks.NewProvider(t)
	mockProv.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	mockProv.EXPECT().AddText("1", mock.AnythingOfType("*entity.Cell"),
		mock.MatchedBy(func(tp *props.Text) bool {
			return tp.Size > 7.1 && tp.Size < 7.3 && tp.Top > 0
		})).Return()

	provider := &shapeProviderFake{Provider: mockProv}

	l := htmllist.New([]htmllist.Item{{Content: nil}}, htmllist.Prop{Style: htmllist.DecimalCircle})
	l.SetConfig(defaultConfig())
	l.Render(provider, &entity.Cell{Width: 100, Height: 200})

	require.NotNil(t, provider.lastCircle)
	assert.InDelta(t, 6.4, provider.lastCircle.Width, 0.001)
	assert.Equal(t, provider.lastCircle.Width, provider.lastCircle.Height)
}

func TestHTMLList_Render_DecimalCircle_FallsBackToTextWhenNoShapeProvider(t *testing.T) {
	t.Parallel()
	provider := mocks.NewProvider(t)
	provider.EXPECT().GetFontHeight(mock.AnythingOfType("*props.Font")).Return(5.0).Maybe()
	provider.EXPECT().GetLinesQuantity(mock.AnythingOfType("string"),
		mock.AnythingOfType("*props.Text"), mock.AnythingOfType("float64")).Return(1).Maybe()
	provider.EXPECT().AddText(mock.AnythingOfType("string"), mock.AnythingOfType("*entity.Cell"),
		mock.AnythingOfType("*props.Text")).Maybe()

	items := []htmllist.Item{{Content: nil}}
	l := htmllist.New(items, htmllist.Prop{Style: htmllist.DecimalCircle})
	l.SetConfig(defaultConfig())

	cell := &entity.Cell{Width: 100, Height: 200}
	assert.NotPanics(t, func() { l.Render(provider, cell) })
}

func TestHTMLList_GetStructure_DecimalCircle_ExposesMarkerStyle(t *testing.T) {
	t.Parallel()
	items := []htmllist.Item{{Content: nil}}
	l := htmllist.New(items, htmllist.Prop{Style: htmllist.DecimalCircle})
	str := l.GetStructure().GetData()
	assert.Equal(t, "decimal-circle", str.Details["marker_style"])
}
