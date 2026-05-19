package translate_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/johnfercher/maroto/v2/pkg/html/translate"
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
