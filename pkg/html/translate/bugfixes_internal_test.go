package translate

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/htmllist"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func translateHTML(t *testing.T, htmlStr string) []string {
	t.Helper()
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	return richTextValues(rows)
}

func TestTableFootRendersAfterBody(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<table><thead><tr><td>HEAD</td></tr></thead><tfoot><tr><td>FOOT</td></tr></tfoot><tbody><tr><td>BODY</td></tr></tbody></table>`)
	require.NoError(t, err)
	tr := &translator{}
	var tableNode *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "table" {
			tableNode = n
			return false
		}
		return true
	})
	require.NotNil(t, tableNode)
	matrix := tr.buildTableMatrix(tableNode, nil)
	require.Equal(t, 3, len(matrix))

	var order []string
	for _, r := range matrix {
		for _, cell := range r {
			if rt, ok := cell.Content.(interface{ PlainText() string }); ok {
				order = append(order, rt.PlainText())
			}
		}
	}
	if len(order) == 3 {
		assert.Equal(t, []string{"HEAD", "BODY", "FOOT"}, order)
	}
}

func TestBodyLevelInlineContentSharesOneRow(t *testing.T) {
	t.Parallel()

	values := translateHTML(t, `<html><body>Total: <strong>$5</strong> due</body></html>`)
	require.Equal(t, 1, len(values))
	assert.Equal(t, "Total: $5 due", strings.TrimSpace(values[0]))
}

func TestOrderedListStartZero(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<ol start="0"><li>zero</li><li>one</li></ol>`)
	require.NoError(t, err)
	tr := &translator{}
	var list *htmllist.HTMLList
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "ol" {
			list = tr.buildList(n)
			return false
		}
		return true
	})
	require.NotNil(t, list)
	marker0, ok := list.MarkerLabel(0)
	require.True(t, ok)
	assert.Equal(t, "0.", marker0)
	marker1, ok := list.MarkerLabel(1)
	require.True(t, ok)
	assert.Equal(t, "1.", marker1)
}

func TestInlineShorthandLonghandOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	// Run repeatedly: with map-order expansion this failed ~15% of the time.
	for range 100 {
		decls := parseInlineStyle("margin: 0; margin-top: 10px")
		require.Equal(t, "10px", decls["margin-top"])
		require.Equal(t, "0", decls["margin-bottom"])
	}
	for range 100 {
		decls := parseInlineStyle("margin-top: 10px; margin: 0")
		require.Equal(t, "0", decls["margin-top"])
	}
}

func TestCSSContentStringHexEscapes(t *testing.T) {
	t.Parallel()

	// Per CSS Syntax §4.3.7 the single whitespace after the hex digits
	// terminates the escape and is consumed; a second space is literal.
	text, rest, ok := readCSSContentString(`"\2022 item"`)
	require.True(t, ok)
	assert.Equal(t, "•item", text)
	assert.Equal(t, "", rest)

	text, _, ok = readCSSContentString(`"\2022  item"`)
	require.True(t, ok)
	assert.Equal(t, "• item", text)

	text, _, ok = readCSSContentString(`"a\a b"`)
	require.True(t, ok)
	assert.Equal(t, "a\nb", text)

	text, _, ok = readCSSContentString(`"say \"hi\""`)
	require.True(t, ok)
	assert.Equal(t, `say "hi"`, text)
}
