package translate

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeNodeStyle_InlineColorApplied(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><p style="color:red">hi</p></body></html>`)
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

	style := computeNodeStyle(pNode, nil)
	require.NotNil(t, style.Color)
	assert.Equal(t, 255, style.Color.R)
	assert.Equal(t, 0, style.Color.G)
}

func TestComputeNodeStyle_FontSize(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><p style="font-size:12pt">hi</p></body></html>`)
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

	style := computeNodeStyle(pNode, nil)
	assert.InDelta(t, 12*0.352778, style.FontSize, 0.01)
}

func TestComputeNodeStyle_ShorthandBorder(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="border: 1px solid red"></div></body></html>`)
	require.NoError(t, err)

	var divNode *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "div" {
			divNode = n
			return false
		}
		return true
	})
	require.NotNil(t, divNode)

	style := computeNodeStyle(divNode, nil)
	assert.Greater(t, style.BorderTopWidth, 0.0)
	assert.NotNil(t, style.BorderTopColor)
	assert.Equal(t, 255, style.BorderTopColor.R)
}
