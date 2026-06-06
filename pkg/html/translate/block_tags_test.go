package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func translateRows(t *testing.T, htmlStr string) int {
	t.Helper()
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	rows, err := Translate(doc)
	require.NoError(t, err)
	return len(rows)
}

func TestBlockTag_DefinitionList(t *testing.T) {
	t.Parallel()
	// dt + dd produce two rows.
	got := translateRows(t, `<dl><dt>Term</dt><dd>Definition</dd></dl>`)
	assert.Equal(t, 2, got)
}

func TestBlockTag_Details(t *testing.T) {
	t.Parallel()
	// summary + body paragraphs produce N rows (here: summary + 1 paragraph).
	got := translateRows(t, `<details><summary>Title</summary><p>Body</p></details>`)
	assert.Equal(t, 2, got)
}

func TestBlockTag_StyledHr(t *testing.T) {
	t.Parallel()
	got := translateRows(t, `<hr style="border-top:2pt solid #888">`)
	assert.Equal(t, 1, got)
}

func TestBlockTag_TableCaption(t *testing.T) {
	t.Parallel()
	// caption + table = 2 rows.
	got := translateRows(t, `<table><caption>Title</caption><tr><td>X</td></tr></table>`)
	assert.Equal(t, 2, got)
}

func TestBlockTag_TableColgroup_DoesNotLogUnsupported(t *testing.T) {
	t.Parallel()
	var unsupported []string
	doc, err := dom.Parse(`<table><colgroup><col width="20%"></colgroup><tr><td>X</td></tr></table>`)
	require.NoError(t, err)
	_, err = Translate(doc, WithUnsupportedHandler(func(thing, value string) {
		unsupported = append(unsupported, thing+":"+value)
	}))
	require.NoError(t, err)
	for _, s := range unsupported {
		assert.NotContains(t, s, "colgroup")
	}
}

func TestBlockTag_ListStyleTypeNone(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>ul.clean{list-style-type:none}</style></head><body><ul class="clean"><li>A</li></ul></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var details map[string]any
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "htmllist" {
			details = s.Details
		}
	})
	require.NotNil(t, details)
	assert.Equal(t, string(htmllist.None), details["style"])
}

func TestBlockTag_OrderedListStartAndReversed(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<ol start="7" reversed><li>A</li><li>B</li></ol>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	var details map[string]any
	walkStructure(rows[0].GetStructure(), func(s core.Structure) {
		if s.Type == "htmllist" {
			details = s.Details
		}
	})
	require.NotNil(t, details)
	assert.Equal(t, string(htmllist.Decimal), details["style"])
	assert.Equal(t, 7, details["start"])
	assert.Equal(t, true, details["reversed"])
}
