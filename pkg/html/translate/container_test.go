package translate

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockContainer_DivWithBackground_ProducesSingleStyledRow(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="background-color:#eaf1fb; padding:5mm"><p>A</p><p>B</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)

	require.Len(t, rows, 1, "div with bg+padding should collapse to one wrapper row")

	cols := rows[0].GetColumns()
	require.Len(t, cols, 1, "wrapper row should have one col")

	// Inspect the structure tree to confirm a container node with two child rows.
	str := rows[0].GetStructure()
	// Walk children: should find a container with rows=2
	var found bool
	for _, child := range str.GetNexts() {
		// Each col contains a container component
		for _, comp := range child.GetNexts() {
			d := comp.GetData()
			if d.Type == "container" {
				found = true
				if rowsCount, ok := d.Details["rows"]; ok {
					assert.Equal(t, 2, rowsCount, "container should contain two child rows (the <p>s)")
				}
			}
		}
	}
	assert.True(t, found, "expected to find a container structure node")
}

func TestBlockContainer_PlainDivStillFlattens(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div><p>a</p><p>b</p></div></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(doc)
	require.NoError(t, err)
	// No styling → keep flat behaviour, 2 rows
	assert.Len(t, rows, 2)
}

func TestBuiltinCSS_TitleBand(t *testing.T) {
	t.Parallel()

	t.Run(".title-band resolves without user <style>", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><body><h2 class="title-band">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		// #1a3e72 → R=26, G=62, B=114
		assert.Equal(t, 26, s.BackgroundColor.R)
		assert.Equal(t, 62, s.BackgroundColor.G)
		assert.Equal(t, 114, s.BackgroundColor.B)
	})

	t.Run("user CSS overrides built-in .title-band", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><head><style>.title-band{background-color:#ff0000}</style></head>
			<body><h2 class="title-band">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		assert.Equal(t, 255, s.BackgroundColor.R)
		assert.Equal(t, 0, s.BackgroundColor.G)
	})

	t.Run("inline style wins over both", func(t *testing.T) {
		t.Parallel()
		tr, doc := parseTranslator(t, `<html><head><style>.title-band{background-color:#ff0000}</style></head>
			<body><h2 class="title-band" style="background-color:#00ff00">SUMMARY</h2></body></html>`)
		h2 := findNode(doc, "h2")
		require.NotNil(t, h2)
		s := computeNodeStyle(tr.sheet, h2, nil)
		require.NotNil(t, s.BackgroundColor)
		assert.Equal(t, 0, s.BackgroundColor.R)
		assert.Equal(t, 255, s.BackgroundColor.G)
	})
}

func TestShouldUseContainer(t *testing.T) {
	t.Parallel()

	t.Run("nil style", func(t *testing.T) {
		t.Parallel()
		assert.False(t, shouldUseContainer(nil))
	})

	t.Run("background-color triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="background-color:red"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("padding-only triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="padding:5mm"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("border triggers", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div style="border:1pt solid red"></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.True(t, shouldUseContainer(s))
	})

	t.Run("plain div does not trigger", func(t *testing.T) {
		t.Parallel()
		doc, _ := dom.Parse(`<html><body><div></div></body></html>`)
		div := findNode(doc, "div")
		require.NotNil(t, div)
		tr := &translator{}
		s := computeNodeStyle(tr.sheet, div, nil)
		assert.False(t, shouldUseContainer(s))
	})
}
