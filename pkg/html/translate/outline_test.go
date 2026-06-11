package translate_test

import (
	"context"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

func translateOutlineDetails(t *testing.T, html string, opts ...translate.Option) []map[string]any {
	t.Helper()
	doc, err := dom.Parse(html)
	require.NoError(t, err)
	rows, err := translate.Translate(context.Background(), doc, opts...)
	require.NoError(t, err)

	var details []map[string]any
	for _, r := range rows {
		for _, rowNode := range r.GetStructure().GetNexts() {
			for _, comp := range rowNode.GetNexts() {
				details = append(details, comp.GetData().Details)
			}
		}
	}
	return details
}

func TestTranslate_WhenOutlineFromHeadings_ShouldSetOutlineOnHeadingLevels(t *testing.T) {
	t.Parallel()

	details := translateOutlineDetails(t,
		"<h1>A</h1><h3>B</h3><p>plain</p>",
		translate.WithOutlineFromHeadings(),
	)

	levels := make([]any, 0)
	for _, d := range details {
		if lvl, ok := d["prop_outline_level"]; ok {
			levels = append(levels, lvl)
		}
	}
	assert.Len(t, levels, 2)
	assert.Equal(t, 0, levels[0])
	assert.Equal(t, 2, levels[1])
}

func TestTranslateDocument_WhenTopLevelHeader_ShouldExtractHeaderRows(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse("<header><p>band</p></header><p>content</p>")
	require.NoError(t, err)

	document, err := translate.TranslateDocument(context.Background(), doc)

	require.NoError(t, err)
	assert.Len(t, document.HeaderRows, 1)
	assert.Len(t, document.Rows, 1)
}

func TestTranslate_WhenOptionAbsent_ShouldNotSetOutline(t *testing.T) {
	t.Parallel()

	details := translateOutlineDetails(t, "<h1>A</h1><h2>B</h2>")

	for _, d := range details {
		_, has := d["prop_outline_level"]
		assert.False(t, has)
	}
}
