package html_test

import (
	"encoding/json"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestDocumentFromString_WhenNoPageRule_ShouldMatchFromStringRows(t *testing.T) {
	t.Parallel()

	input := "<h1>Title</h1><p>Some paragraph</p>"

	rows, err := html.FromString(input)
	require.NoError(t, err)
	doc, err := html.DocumentFromString(input)
	require.NoError(t, err)

	assert.Nil(t, doc.Page)
	require.Len(t, doc.Rows, len(rows))
	for i := range rows {
		expected, jsonErr := json.Marshal(rows[i].GetStructure().GetData())
		require.NoError(t, jsonErr)
		actual, jsonErr := json.Marshal(doc.Rows[i].GetStructure().GetData())
		require.NoError(t, jsonErr)
		assert.Equal(t, string(expected), string(actual))
	}
}

func TestDocumentFromString_WhenPageRule_ShouldExposePageOptions(t *testing.T) {
	t.Parallel()

	doc, err := html.DocumentFromString(`<style>@page { size: A5; margin: 12mm }</style><p>x</p>`)

	require.NoError(t, err)
	require.NotNil(t, doc.Page)
	assert.Equal(t, "a5", doc.Page.PageSize)
	assert.Equal(t, 12.0, doc.Page.MarginTop)
}

func TestDocumentFromString_WhenPseudoPageRule_ShouldNotifyUnsupportedHandler(t *testing.T) {
	t.Parallel()

	var got []string
	_, err := html.DocumentFromString(
		`<style>@page :first { margin: 0 }</style><p>x</p>`,
		html.WithUnsupportedHandler(func(thing, value string) {
			got = append(got, thing+"="+value)
		}),
	)

	require.NoError(t, err)
	assert.Contains(t, got, "page-rule.skipped=@page :first")
}
