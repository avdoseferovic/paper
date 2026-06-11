package dom_test

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"golang.org/x/net/html"
)

// findNode walks the document and returns the first node matching tag.
func findNode(t *testing.T, doc *dom.Document, tag string) *dom.Node {
	t.Helper()
	var found *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == tag {
			found = n
			return false
		}
		return true
	})
	require.NotNil(t, found, "expected to find <%s>", tag)
	return found
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	t.Run("deeply nested elements wraps ErrDOMTooDeep", func(t *testing.T) {
		t.Parallel()
		src := strings.Repeat("<div>", 600)
		doc, err := dom.Parse(src)
		require.Error(t, err)
		assert.ErrorIs(t, err, htmllimits.ErrDOMTooDeep)
		assert.Nil(t, doc)
	})

	t.Run("empty input parses to a document", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse("")
		require.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "", doc.StyleText())
	})
}

func TestStyleSources(t *testing.T) {
	t.Parallel()

	t.Run("returns inline style text and stylesheet links in DOM order", func(t *testing.T) {
		t.Parallel()
		src := `<html><head>
			<style>p{color:red}</style>
			<link rel="stylesheet" href="a.css">
			<link rel="STYLESHEET" href="b.css">
		</head><body><link rel="stylesheet" href="c.css"></body></html>`
		doc, err := dom.Parse(src)
		require.NoError(t, err)
		text, links := doc.StyleSources()
		assert.Contains(t, text, "p{color:red}")
		require.Len(t, links, 3)
		assert.Equal(t, "a.css", links[0])
		assert.Equal(t, "b.css", links[1])
		assert.Equal(t, "c.css", links[2])
	})

	t.Run("ignores non-stylesheet links and links without href", func(t *testing.T) {
		t.Parallel()
		src := `<html><head>
			<link rel="icon" href="favicon.ico">
			<link rel="stylesheet">
			<link href="orphan.css">
		</head><body></body></html>`
		doc, err := dom.Parse(src)
		require.NoError(t, err)
		text, links := doc.StyleSources()
		assert.Equal(t, "", text)
		assert.Empty(t, links)
	})

	t.Run("concatenates multiple style blocks", func(t *testing.T) {
		t.Parallel()
		src := `<html><head><style>a{}</style><style>b{}</style></head><body></body></html>`
		doc, err := dom.Parse(src)
		require.NoError(t, err)
		text, links := doc.StyleSources()
		assert.Contains(t, text, "a{}")
		assert.Contains(t, text, "b{}")
		assert.Empty(t, links)
	})
}

func TestValidateLimits(t *testing.T) {
	t.Parallel()

	t.Run("passes within limits", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p>hi</p></body></html>`)
		require.NoError(t, err)
		assert.NoError(t, doc.ValidateLimits(htmllimits.Default()))
	})

	t.Run("returns ErrDOMTooDeep when depth limit exceeded", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div><div><div><p>deep</p></div></div></div></body></html>`)
		require.NoError(t, err)
		limits := htmllimits.Limits{MaxDOMDepth: 2}
		err = doc.ValidateLimits(limits)
		require.Error(t, err)
		assert.ErrorIs(t, err, htmllimits.ErrDOMTooDeep)
	})

	t.Run("returns ErrDOMTooLarge when node limit exceeded", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p>a</p><p>b</p><p>c</p></body></html>`)
		require.NoError(t, err)
		limits := htmllimits.Limits{MaxDOMNodes: 2}
		err = doc.ValidateLimits(limits)
		require.Error(t, err)
		assert.ErrorIs(t, err, htmllimits.ErrDOMTooLarge)
	})
}

func TestWalkWithLimits(t *testing.T) {
	t.Parallel()

	t.Run("stops traversal with error when node limit exceeded", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p>a</p><p>b</p><p>c</p></body></html>`)
		require.NoError(t, err)
		var visited int
		err = doc.WalkWithLimits(htmllimits.Limits{MaxDOMNodes: 2}, func(n *dom.Node) bool {
			visited++
			return true
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, htmllimits.ErrDOMTooLarge)
		assert.Less(t, visited, 4)
	})

	t.Run("visits all elements when limits disabled", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div><p>a</p><span>b</span></div></body></html>`)
		require.NoError(t, err)
		var tags []string
		err = doc.WalkWithLimits(htmllimits.NoLimits(), func(n *dom.Node) bool {
			tags = append(tags, n.Tag())
			return true
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"body", "div", "p", "span"}, tags)
	})

	t.Run("callback returning false stops traversal without error", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div><p>a</p></div><span>b</span></body></html>`)
		require.NoError(t, err)
		var tags []string
		err = doc.WalkWithLimits(htmllimits.NoLimits(), func(n *dom.Node) bool {
			tags = append(tags, n.Tag())
			return n.Tag() != "div"
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"body", "div", "span"}, tags)
	})
}

func TestHTMLElement(t *testing.T) {
	t.Parallel()

	t.Run("returns html element node", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html lang="en"><body></body></html>`)
		require.NoError(t, err)
		h := doc.HTMLElement()
		require.NotNil(t, h)
		assert.Equal(t, "html", h.Tag())
		assert.Equal(t, "en", h.Attr("lang"))
	})
}

func TestRootAndRawNode(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><p>hi</p></body></html>`)
	require.NoError(t, err)

	root := doc.Root()
	require.NotNil(t, root)
	assert.Equal(t, html.DocumentNode, root.Type)

	p := findNode(t, doc, "p")
	raw := p.RawNode()
	require.NotNil(t, raw)
	assert.Equal(t, html.ElementNode, raw.Type)
	assert.Equal(t, "p", raw.Data)
}

func TestNodeAccessors(t *testing.T) {
	t.Parallel()

	t.Run("Attr returns empty string for absent attribute", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p id="x">hi</p></body></html>`)
		require.NoError(t, err)
		p := findNode(t, doc, "p")
		assert.Equal(t, "x", p.Attr("id"))
		assert.Equal(t, "", p.Attr("class"))
	})

	t.Run("Tag returns empty string for text nodes", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p>hi</p></body></html>`)
		require.NoError(t, err)
		p := findNode(t, doc, "p")
		children := p.Children()
		require.Len(t, children, 1)
		assert.Equal(t, "", children[0].Tag())
	})

	t.Run("IsBlock and IsInline classify elements", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div><span>hi</span></div></body></html>`)
		require.NoError(t, err)
		div := findNode(t, doc, "div")
		span := findNode(t, doc, "span")
		assert.True(t, div.IsBlock())
		assert.False(t, div.IsInline())
		assert.False(t, span.IsBlock())
		assert.True(t, span.IsInline())
	})

	t.Run("IsInline is false for text nodes", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><p>hi</p></body></html>`)
		require.NoError(t, err)
		p := findNode(t, doc, "p")
		text := p.Children()[0]
		assert.False(t, text.IsBlock())
		assert.False(t, text.IsInline())
	})
}

func TestChildren(t *testing.T) {
	t.Parallel()

	t.Run("returns element and text children, skipping comments", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div>before<!-- comment --><span>mid</span>after</div></body></html>`)
		require.NoError(t, err)
		div := findNode(t, doc, "div")
		children := div.Children()
		require.Len(t, children, 3)
		assert.Equal(t, "", children[0].Tag())
		assert.Equal(t, "span", children[1].Tag())
		assert.Equal(t, "", children[2].Tag())
	})

	t.Run("returns nil for empty element", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse(`<html><body><div></div></body></html>`)
		require.NoError(t, err)
		div := findNode(t, doc, "div")
		assert.Empty(t, div.Children())
	})

	t.Run("children of pre preserve whitespace in text content", func(t *testing.T) {
		t.Parallel()
		doc, err := dom.Parse("<html><body><pre><span>a   b</span></pre></body></html>")
		require.NoError(t, err)
		pre := findNode(t, doc, "pre")
		children := pre.Children()
		require.Len(t, children, 1)
		assert.Equal(t, "a   b", children[0].TextContent())
	})
}

func TestTextContentWhitespace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		tag  string
		want string
	}{
		{"empty element", `<div></div>`, "div", ""},
		{"whitespace only collapses to single space", `<div>   </div>`, "div", " "},
		{"leading and trailing space preserved as single space", `<div>  hello  world  </div>`, "div", " hello world "},
		{"tabs and newlines collapse", "<div>a\t\n b</div>", "div", "a b"},
		{"code preserves whitespace", `<code>x   y</code>`, "code", "x   y"},
		// The final collapse pass applies because the queried node itself is
		// not preformatted, so nested <pre> whitespace is still collapsed.
		{"nested pre inside non-pre ancestor collapses", `<div><pre>a   b</pre></div>`, "div", "a b"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := dom.Parse(`<html><body>` + tc.src + `</body></html>`)
			require.NoError(t, err)
			n := findNode(t, doc, tc.tag)
			assert.Equal(t, tc.want, n.TextContent())
		})
	}
}
