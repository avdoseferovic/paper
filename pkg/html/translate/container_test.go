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
