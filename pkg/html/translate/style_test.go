package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findFirstNode(t *testing.T, doc *dom.Document, tag string) *dom.Node {
	t.Helper()
	var found *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == tag {
			found = n
			return false
		}
		return true
	})
	require.NotNil(t, found, "node %q not found", tag)
	return found
}

func TestComputeNodeStyle_InlineColorApplied(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><p style="color:red">hi</p></body></html>`)
	require.NoError(t, err)

	style := computeNodeStyle(nil, findFirstNode(t, doc, "p"), nil)
	require.NotNil(t, style.Color)
	assert.Equal(t, 255, style.Color.R)
	assert.Equal(t, 0, style.Color.G)
}

func TestComputeNodeStyle_FontSize(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><p style="font-size:12pt">hi</p></body></html>`)
	require.NoError(t, err)

	style := computeNodeStyle(nil, findFirstNode(t, doc, "p"), nil)
	assert.InDelta(t, 12*0.352778, style.FontSize, 0.01)
}

func TestStylesheet_RulesAppliedToMatchingNodes(t *testing.T) {
	t.Parallel()
	html := `<html><head><style>p { color: #00ff00 } .x { font-size: 16pt }</style></head>` +
		`<body><p class="x">text</p></body></html>`

	doc, err := dom.Parse(html)
	require.NoError(t, err)

	sheet := parseStylesheet(doc.StyleText())
	style := computeNodeStyle(sheet, findFirstNode(t, doc, "p"), nil)
	require.NotNil(t, style.Color, "p { color } should apply via <style> block")
	assert.Equal(t, 0, style.Color.R)
	assert.Equal(t, 255, style.Color.G)
	assert.InDelta(t, 16*0.352778, style.FontSize, 0.01, ".x { font-size } should apply")
}

func TestStylesheet_InlineBeatsBlockRule(t *testing.T) {
	t.Parallel()
	html := `<html><head><style>p { color: red }</style></head>` +
		`<body><p style="color:#0000ff">text</p></body></html>`

	doc, err := dom.Parse(html)
	require.NoError(t, err)

	sheet := parseStylesheet(doc.StyleText())
	style := computeNodeStyle(sheet, findFirstNode(t, doc, "p"), nil)
	require.NotNil(t, style.Color)
	assert.Equal(t, 0, style.Color.R, "inline color should win over <style> block")
	assert.Equal(t, 255, style.Color.B)
}

func TestComputeNodeStyle_ShorthandBorder(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="border: 1px solid red"></div></body></html>`)
	require.NoError(t, err)

	style := computeNodeStyle(nil, findFirstNode(t, doc, "div"), nil)
	assert.Greater(t, style.BorderTopWidth, 0.0)
	assert.NotNil(t, style.BorderTopColor)
	assert.Equal(t, 255, style.BorderTopColor.R)
}
