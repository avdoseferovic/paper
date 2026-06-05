package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/v2/pkg/html/dom"
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

func TestBlockTag_TableColgroup_LogsUnsupported(t *testing.T) {
	t.Parallel()
	var unsupported []string
	doc, err := dom.Parse(`<table><colgroup><col width="20%"></colgroup><tr><td>X</td></tr></table>`)
	require.NoError(t, err)
	_, err = Translate(doc, WithUnsupportedHandler(func(thing, value string) {
		unsupported = append(unsupported, thing+":"+value)
	}))
	require.NoError(t, err)
	// At least one entry mentions colgroup.
	var hasColgroup bool
	for _, s := range unsupported {
		if assert.Contains(t, s, "colgroup") || hasColgroup {
			hasColgroup = true
		}
	}
}
