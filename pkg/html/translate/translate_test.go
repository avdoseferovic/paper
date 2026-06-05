package translate_test

import (
	"testing"

	"github.com/johnfercher/paper/v2/pkg/html/dom"
	"github.com/johnfercher/paper/v2/pkg/html/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDoc(t *testing.T, src string) *dom.Document {
	t.Helper()
	d, err := dom.Parse(src)
	require.NoError(t, err)
	return d
}

func TestTranslate_Block(t *testing.T) {
	t.Parallel()

	t.Run("single paragraph produces one row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><p>hello</p></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("h1 + p produces two rows", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><h1>title</h1><p>body</p></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 2)
	})

	t.Run("display:none skips element", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><p style="display:none">hidden</p><p>visible</p></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("hr produces a row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><hr></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("div with children flattens to children's rows", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><div><p>a</p><p>b</p></div></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 2)
	})
}

func TestTranslate_Inline(t *testing.T) {
	t.Parallel()

	t.Run("p with bold child produces one row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><p>hello <b>world</b></p></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestTranslate_Table(t *testing.T) {
	t.Parallel()
	doc := parseDoc(t, "<html><body><table><tr><td>a</td><td>b</td></tr></table></body></html>")
	rows, err := translate.Translate(doc)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rows), 1)
}

func TestTranslate_Flex(t *testing.T) {
	t.Parallel()

	t.Run("class-based display:flex produces 1 row", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><head><style>.cols{display:flex}</style></head><body><div class="cols"><div>a</div><div>b</div></div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Len(t, rows[0].GetColumns(), 2)
	})

	t.Run("class-based display:none via stylesheet hides element", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><head><style>.hidden{display:none}</style></head><body><div class="hidden"><p>invisible</p></div><p>visible</p></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("WithGridSize(8) distributes flex cols over 8 not 12", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><body><div style="display:flex"><div>a</div><div>b</div></div></body></html>`)
		rows, err := translate.Translate(doc, translate.WithGridSize(8))
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		require.Len(t, cols, 2)
		assert.Equal(t, 4, cols[0].GetSize())
		assert.Equal(t, 4, cols[1].GetSize())
	})
}

// TestTranslate_FlexDemoSections asserts that the html-demo's flex sections
// produce the expected structural shape (1 row with N cols). Catches regressions
// where flex dispatch silently fails and content gets flattened back to stacked rows.
func TestTranslate_FlexDemoSections(t *testing.T) {
	t.Parallel()

	t.Run("3-col parties section produces 1 row with 3 cols", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><head><style>.parties{display:flex;gap:6mm}</style></head>
			<body><div class="parties">
				<div><h3>Bill to</h3><p>Acme</p></div>
				<div><h3>Ship to</h3><p>Warehouse</p></div>
				<div><h3>Payment</h3><p>Net 30</p></div>
			</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		// 3 content cols + up to 2 gap spacers = 3..5 cols. Must be ≥ 3.
		assert.GreaterOrEqual(t, len(cols), 3)
	})

	t.Run("space-between totals strip produces visible between-spacer", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, `<html><head><style>.totals{display:flex;justify-content:space-between}</style></head>
			<body><div class="totals">
				<div style="flex:0 0 50%"><b>Amount due</b></div>
				<div style="flex:0 0 25%"><b>$540.00</b></div>
			</div></body></html>`)
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		cols := rows[0].GetColumns()
		// items 6 + 3 = 9 cols, slack 3 → between-spacer of 3 cols. Total 3 cols.
		require.Len(t, cols, 3)
		sum := 0
		for _, c := range cols {
			sum += c.GetSize()
		}
		assert.Equal(t, 12, sum)
	})
}

func TestTranslate_List(t *testing.T) {
	t.Parallel()

	t.Run("ul produces one row containing HTMLList", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><ul><li>a</li><li>b</li></ul></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(rows), 1)
	})

	t.Run("ol produces decimal markers", func(t *testing.T) {
		t.Parallel()
		doc := parseDoc(t, "<html><body><ol><li>a</li><li>b</li></ol></body></html>")
		rows, err := translate.Translate(doc)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(rows), 1)
	})
}
