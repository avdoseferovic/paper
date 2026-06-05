package dom_test

import (
	"testing"

	"github.com/johnfercher/paper/v2/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("parses minimal html and returns document", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse("<html><body><p>hello</p></body></html>")
		require.NoError(t, err)
		assert.NotNil(t, doc)
	})

	t.Run("extracts style block text from head", func(t *testing.T) {
		t.Parallel()
		html := `<html><head><style>p{color:red}</style></head><body><p>hi</p></body></html>`
		doc, err := dom.Parse(html)
		require.NoError(t, err)
		assert.Contains(t, doc.StyleText(), "p{color:red}")
	})

	t.Run("extracts inline style from body style tag", func(t *testing.T) {
		t.Parallel()
		html := `<html><body><style>.x{font-size:12pt}</style><p>hi</p></body></html>`
		doc, err := dom.Parse(html)
		require.NoError(t, err)
		assert.Contains(t, doc.StyleText(), ".x{font-size:12pt}")
	})

	t.Run("parses element inline style attribute", func(t *testing.T) {
		t.Parallel()
		html := `<html><body><p style="font-weight:bold">hi</p></body></html>`
		doc, err := dom.Parse(html)
		require.NoError(t, err)

		var pNode *dom.Node
		doc.Walk(func(n *dom.Node) bool {
			if n.Tag() == "p" {
				pNode = n
				return false
			}
			return true
		})
		require.NotNil(t, pNode)
		assert.Equal(t, "font-weight:bold", pNode.InlineStyle())
		assert.Equal(t, "hi", pNode.TextContent())
	})

	t.Run("whitespace collapses between inline nodes", func(t *testing.T) {
		t.Parallel()
		html := `<html><body><p>hello   world</p></body></html>`
		doc, err := dom.Parse(html)
		require.NoError(t, err)

		var pNode *dom.Node
		doc.Walk(func(n *dom.Node) bool {
			if n.Tag() == "p" {
				pNode = n
				return false
			}
			return true
		})
		require.NotNil(t, pNode)
		assert.Equal(t, "hello world", pNode.TextContent())
	})

	t.Run("pre element preserves whitespace", func(t *testing.T) {
		t.Parallel()
		html := `<html><body><pre>hello   world</pre></body></html>`
		doc, err := dom.Parse(html)
		require.NoError(t, err)

		var preNode *dom.Node
		doc.Walk(func(n *dom.Node) bool {
			if n.Tag() == "pre" {
				preNode = n
				return false
			}
			return true
		})
		require.NotNil(t, preNode)
		assert.Equal(t, "hello   world", preNode.TextContent())
	})
}

func TestNodeClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag     string
		isBlock bool
	}{
		{"div", true},
		{"p", true},
		{"h1", true},
		{"table", true},
		{"ul", true},
		{"span", false},
		{"a", false},
		{"strong", false},
		{"em", false},
		{"img", false},
	}

	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			assert.Equal(t, tc.isBlock, dom.IsBlockTag(tc.tag))
		})
	}
}
