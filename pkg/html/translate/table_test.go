package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseTranslator is a test helper that builds a *translator from a full HTML string.
func parseTranslator(t *testing.T, htmlSrc string) (*translator, *dom.Document) {
	t.Helper()
	doc, err := dom.Parse(htmlSrc)
	require.NoError(t, err)
	return &translator{sheet: parseStylesheet(doc.StyleText())}, doc
}

// findNode walks the document and returns the first node with the given tag.
func findNode(doc *dom.Document, tag string) *dom.Node {
	var found *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == tag {
			found = n
			return false
		}
		return true
	})
	return found
}

func TestBuildCell_RowStyleBackground(t *testing.T) {
	t.Parallel()

	t.Run("tr background propagates to cell when cell has no own background", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><table>
			<tr style="background-color:#1a3e72"><td>Cell</td></tr>
		</table></body></html>`)

		trNode := findNode(doc, "tr")
		require.NotNil(t, trNode)

		rowStyle := computeNodeStyle(tr.sheet, trNode, nil)
		cells := tr.buildRow(trNode, rowStyle)

		require.Len(t, cells, 1)
		require.NotNil(t, cells[0].Style, "cell Style should be set from row background")
		require.NotNil(t, cells[0].Style.BackgroundColor)
		assert.Equal(t, 26, cells[0].Style.BackgroundColor.Red)   // 0x1a = 26
		assert.Equal(t, 62, cells[0].Style.BackgroundColor.Green) // 0x3e = 62
		assert.Equal(t, 114, cells[0].Style.BackgroundColor.Blue) // 0x72 = 114
	})

	t.Run("cell's own background wins over row background", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><table>
			<tr style="background-color:#1a3e72"><td style="background-color:#ff0000">Cell</td></tr>
		</table></body></html>`)

		trNode := findNode(doc, "tr")
		require.NotNil(t, trNode)

		rowStyle := computeNodeStyle(tr.sheet, trNode, nil)
		cells := tr.buildRow(trNode, rowStyle)

		require.Len(t, cells, 1)
		require.NotNil(t, cells[0].Style)
		require.NotNil(t, cells[0].Style.BackgroundColor)
		// Cell's own red background should win
		assert.Equal(t, 255, cells[0].Style.BackgroundColor.Red)
		assert.Equal(t, 0, cells[0].Style.BackgroundColor.Green)
	})

	t.Run("tr without background produces nil td Style", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><table>
			<tr><td>Cell</td></tr>
		</table></body></html>`)

		trNode := findNode(doc, "tr")
		require.NotNil(t, trNode)

		rowStyle := computeNodeStyle(tr.sheet, trNode, nil)
		cells := tr.buildRow(trNode, rowStyle)

		require.Len(t, cells, 1)
		assert.Nil(t, cells[0].Style)
	})

	t.Run("cell padding is carried into table cell style", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><table>
			<tr><th style="padding:2mm 3mm 4mm 5mm">Item</th></tr>
		</table></body></html>`)

		trNode := findNode(doc, "tr")
		require.NotNil(t, trNode)

		rowStyle := computeNodeStyle(tr.sheet, trNode, nil)
		cells := tr.buildRow(trNode, rowStyle)

		require.Len(t, cells, 1)
		require.NotNil(t, cells[0].Style)
		assert.Equal(t, 2.0, cells[0].Style.PaddingTop)
		assert.Equal(t, 3.0, cells[0].Style.PaddingRight)
		assert.Equal(t, 4.0, cells[0].Style.PaddingBottom)
		assert.Equal(t, 5.0, cells[0].Style.PaddingLeft)
	})

	t.Run("built-in table header padding applies when css omits padding", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><table>
			<tr><th>Item</th><td>Value</td></tr>
		</table></body></html>`)

		trNode := findNode(doc, "tr")
		require.NotNil(t, trNode)

		rowStyle := computeNodeStyle(tr.sheet, trNode, nil)
		cells := tr.buildRow(trNode, rowStyle)

		require.Len(t, cells, 2)
		require.NotNil(t, cells[0].Style)
		assert.Equal(t, 0.8, cells[0].Style.PaddingTop)
		assert.Equal(t, 1.0, cells[0].Style.PaddingRight)
		assert.Equal(t, 0.8, cells[0].Style.PaddingBottom)
		assert.Equal(t, 1.0, cells[0].Style.PaddingLeft)
		assert.Nil(t, cells[1].Style)
	})
}

func TestTranslate_TableRowStyle_Integration(t *testing.T) {
	t.Parallel()

	t.Run("table with styled tr row produces 1 row", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><table>
			<tr style="background-color:#1a3e72"><td>A</td><td>B</td></tr>
		</table></body></html>`)
		require.NoError(t, err)

		rows, err := Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}
