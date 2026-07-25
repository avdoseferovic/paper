package translate

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"

	"github.com/avdoseferovic/paper/pkg/html/dom"
)

// applyStyleAndGetBg parses the HTML, applies the user CSS via the
// translator's stylesheet pipeline, then returns the computed background
// colour of the first element matching findTag (or nil when unmatched / no bg).
func styleNthChild(t *testing.T, htmlStr, findTag string, nth int) string {
	t.Helper()
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	sheet := parseStylesheet(doc.StyleText())
	var found []*dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == findTag {
			found = append(found, n)
		}
		return true
	})
	require.GreaterOrEqual(t, len(found), nth, "expected at least %d <%s>", nth, findTag)
	style := computeNodeStyle(sheet, found[nth-1], nil)
	if style.BackgroundColor == nil {
		return ""
	}
	r, g, b := style.BackgroundColor.R, style.BackgroundColor.G, style.BackgroundColor.B
	return formatRGB(r, g, b)
}

func formatRGB(r, g, b int) string {
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

func TestSelector_NthChild_Even(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
li:nth-child(even) { background-color: #f0f0f0 }
</style></head>
<body><ul><li>1</li><li>2</li><li>3</li><li>4</li></ul></body></html>`
	// 1st, 3rd: no bg. 2nd, 4th: bg.
	assert.Equal(t, "", styleNthChild(t, html, "li", 1))
	assert.NotEqual(t, "", styleNthChild(t, html, "li", 2))
	assert.Equal(t, "", styleNthChild(t, html, "li", 3))
	assert.NotEqual(t, "", styleNthChild(t, html, "li", 4))
}

func TestSelector_FirstChild(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
li:first-child { background-color: #ffaa00 }
</style></head>
<body><ul><li>A</li><li>B</li><li>C</li></ul></body></html>`
	assert.NotEqual(t, "", styleNthChild(t, html, "li", 1))
	assert.Equal(t, "", styleNthChild(t, html, "li", 2))
	assert.Equal(t, "", styleNthChild(t, html, "li", 3))
}

func TestSelector_LastChild(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
li:last-child { background-color: #00aa00 }
</style></head>
<body><ul><li>A</li><li>B</li><li>C</li></ul></body></html>`
	assert.Equal(t, "", styleNthChild(t, html, "li", 1))
	assert.Equal(t, "", styleNthChild(t, html, "li", 2))
	assert.NotEqual(t, "", styleNthChild(t, html, "li", 3))
}

func TestSelector_Not(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
p:not(.intro) { background-color: #ddd }
</style></head>
<body><p class="intro">A</p><p>B</p><p>C</p></body></html>`
	assert.Equal(t, "", styleNthChild(t, html, "p", 1))    // .intro skipped
	assert.NotEqual(t, "", styleNthChild(t, html, "p", 2)) // matched
	assert.NotEqual(t, "", styleNthChild(t, html, "p", 3)) // matched
}

func TestSelector_AttributeExact(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
[data-status="ok"] { background-color: #aaffaa }
</style></head>
<body><span data-status="ok">A</span><span data-status="warn">B</span></body></html>`
	assert.NotEqual(t, "", styleNthChild(t, html, "span", 1))
	assert.Equal(t, "", styleNthChild(t, html, "span", 2))
}

func TestSelector_AttributePrefix(t *testing.T) {
	t.Parallel()
	html := `
<html><head><style>
a[href^="https://"] { background-color: #ccddff }
</style></head>
<body><a href="https://x.com">A</a><a href="http://y.com">B</a></body></html>`
	assert.NotEqual(t, "", styleNthChild(t, html, "a", 1))
	assert.Equal(t, "", styleNthChild(t, html, "a", 2))
}

func TestSelector_HoverNeverMatches(t *testing.T) {
	t.Parallel()
	// :hover is a state-dependent selector; in static PDF output it should
	// silently never match. We just verify no error / no panic.
	html := `
<html><head><style>
a:hover { background-color: red }
</style></head>
<body><a href="#">link</a></body></html>`
	assert.Equal(t, "", styleNthChild(t, html, "a", 1))
}

func TestSelectorMatcher_CombinatorsAttributesAndRoot(t *testing.T) {
	t.Parallel()

	doc, err := dom.Parse(`<html><body><div id="root"><p class="lead">one</p><span data-kind="notice-primary">two</span><span data-kind="notice">three</span></div></body></html>`)
	require.NoError(t, err)

	var nodes []*dom.Node
	doc.Walk(func(node *dom.Node) bool {
		if node.Tag() == "span" {
			nodes = append(nodes, node)
		}
		return true
	})
	require.Len(t, nodes, 2)

	matcher, err := compileSelector(`div#root > p.lead + span[data-kind|="notice"]`)
	require.NoError(t, err)
	assert.True(t, matcher.Match(nodes[0].RawNode()))
	assert.False(t, matcher.Match(nodes[1].RawNode()))

	root, err := compileSelector(":root")
	require.NoError(t, err)
	assert.True(t, root.Match(doc.HTMLElement().RawNode()))
}

func FuzzCompileSelector(f *testing.F) {
	for _, seed := range []string{
		"p", "#id > .class", `a[href^="https://"]`, ":not(.hidden)", "li:nth-child(2n+1)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		matcher, err := compileSelector(value)
		if err != nil {
			return
		}
		doc, parseErr := dom.Parse(`<html><body><p id="id" class="class">x</p></body></html>`)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		_ = matcher.Match(doc.HTMLElement().RawNode())
	})
}
