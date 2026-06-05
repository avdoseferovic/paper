package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnfercher/paper/v2/pkg/html/dom"
)

func TestVars_BodyDefinedInheritsToChildren(t *testing.T) {
	t.Parallel()
	// Define the variable on body so the walker (which starts at body) can
	// build the cascade. :root works in browsers but our Walk traverses from
	// body downwards — same semantics as far as inheritance is concerned.
	html := `
<html><head><style>
body { --accent: #ff0000 }
p { color: var(--accent) }
</style></head>
<body><p>hello</p></body></html>`
	doc, err := dom.Parse(html)
	require.NoError(t, err)
	sheet := parseStylesheet(doc.StyleText())
	var body, p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		switch n.Tag() {
		case "body":
			body = n
		case "p":
			p = n
		}
		return true
	})
	require.NotNil(t, body)
	require.NotNil(t, p)
	bodyStyle := computeNodeStyle(sheet, body, nil)
	pStyle := computeNodeStyle(sheet, p, bodyStyle)
	require.NotNil(t, pStyle.Color)
	assert.Equal(t, 255, pStyle.Color.R)
	assert.Equal(t, 0, pStyle.Color.G)
	assert.Equal(t, 0, pStyle.Color.B)
}

func TestVars_InlineStyleVarUsed(t *testing.T) {
	t.Parallel()
	html := `<div style="--bg: #00ff00"><p style="background-color: var(--bg)">hi</p></div>`
	doc, err := dom.Parse(html)
	require.NoError(t, err)
	sheet := parseStylesheet(doc.StyleText())
	var div, p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		switch n.Tag() {
		case "div":
			div = n
		case "p":
			p = n
		}
		return true
	})
	require.NotNil(t, div)
	require.NotNil(t, p)
	divStyle := computeNodeStyle(sheet, div, nil)
	pStyle := computeNodeStyle(sheet, p, divStyle)
	require.NotNil(t, pStyle.BackgroundColor)
	assert.Equal(t, 0, pStyle.BackgroundColor.R)
	assert.Equal(t, 255, pStyle.BackgroundColor.G)
	assert.Equal(t, 0, pStyle.BackgroundColor.B)
}
