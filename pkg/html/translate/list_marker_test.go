package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

type markerTextCall struct {
	text string
	prop *props.Text
}

type markerRecordingProvider struct {
	cursorProvider
	texts []markerTextCall
}

func (p *markerRecordingProvider) AddText(text string, _ *entity.Cell, prop *props.Text) {
	p.texts = append(p.texts, markerTextCall{text: text, prop: prop})
}

func TestTranslate_ListMarkerPseudoColorAndFontSize(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
li { color: #000000 }
li::marker { color: #00ff00; font-size: 20pt }
</style></head><body><ul><li>Item</li></ul></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	setMarkerTestConfig(rows[0])

	provider := &markerRecordingProvider{}
	rows[0].Render(provider, entity.Cell{Width: 100, Height: 100})

	require.NotEmpty(t, provider.texts)
	marker := provider.texts[0]
	assert.Equal(t, "•", marker.text)
	require.NotNil(t, marker.prop.Color)
	assert.Equal(t, 0, marker.prop.Color.Red)
	assert.Equal(t, 255, marker.prop.Color.Green)
	assert.Equal(t, 0, marker.prop.Color.Blue)
	assert.Equal(t, 20.0, marker.prop.Size)
}

func TestTranslate_ListMarkerPseudoContentLiteral(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
ul { list-style: none }
li::marker { content: "→ " }
</style></head><body><ul><li>Item</li></ul></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	setMarkerTestConfig(rows[0])

	provider := &markerRecordingProvider{}
	rows[0].Render(provider, entity.Cell{Width: 100, Height: 100})

	require.NotEmpty(t, provider.texts)
	assert.Equal(t, "→ ", provider.texts[0].text)
}

func TestTranslate_ListMarkerPseudoContentNoneSuppressesMarker(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
li::marker { content: none }
</style></head><body><ol><li>Item</li></ol></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	setMarkerTestConfig(rows[0])

	provider := &markerRecordingProvider{}
	rows[0].Render(provider, entity.Cell{Width: 100, Height: 100})

	for _, call := range provider.texts {
		assert.NotEqual(t, "1.", call.text)
	}
}

func TestTranslate_ListMarkerPseudoContentCounterStyle(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
ol { counter-reset: c; list-style: none }
li { counter-increment: c }
li::marker { content: counter(c, upper-roman) ". " }
</style></head><body><ol><li>One</li><li>Two</li></ol></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	setMarkerTestConfig(rows[0])

	provider := &markerRecordingProvider{}
	rows[0].Render(provider, entity.Cell{Width: 100, Height: 100})

	require.Len(t, provider.texts, 4)
	assert.Equal(t, "I. ", provider.texts[0].text)
	assert.Equal(t, "II. ", provider.texts[2].text)
}

func TestTranslate_ListMarkerPseudoContentNestedCounters(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
ol { counter-reset: item; list-style: none }
li { counter-increment: item }
li::marker { content: counters(item, ".") ". " }
</style></head><body><ol><li>Alpha<ol><li>Beta</li></ol></li><li>Delta</li></ol></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	setMarkerTestConfig(rows[0])

	provider := &markerRecordingProvider{}
	rows[0].Render(provider, entity.Cell{Width: 100, Height: 100})

	texts := recordedMarkerTexts(provider)
	assert.Contains(t, texts, "1. ")
	assert.Contains(t, texts, "1.1. ")
	assert.Contains(t, texts, "2. ")
}

func setMarkerTestConfig(row core.Row) {
	row.SetConfig(&entity.Config{
		MaxGridSize: 12,
		DefaultFont: &props.Font{
			Family: "Helvetica",
			Style:  fontstyle.Normal,
			Size:   10,
		},
	})
}

func recordedMarkerTexts(provider *markerRecordingProvider) []string {
	out := make([]string, len(provider.texts))
	for i, call := range provider.texts {
		out[i] = call.text
	}
	return out
}
