package translate

import (
	"context"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html/dom"
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

func TestStylesheet_PrintMediaRulesApply(t *testing.T) {
	t.Parallel()
	html := `<html><head><style>
p { color: red }
@media print { p { color: #00ff00 } }
@media screen { p { font-size: 22pt } }
@media all { p { font-weight: bold } }
</style></head><body><p>text</p></body></html>`

	doc, err := dom.Parse(html)
	require.NoError(t, err)

	sheet := parseStylesheet(doc.StyleText())
	style := computeNodeStyle(sheet, findFirstNode(t, doc, "p"), nil)
	require.NotNil(t, style.Color)
	assert.Equal(t, 0, style.Color.R)
	assert.Equal(t, 255, style.Color.G)
	assert.Equal(t, "bold", style.FontWeight)
	assert.Zero(t, style.FontSize, "screen-only media rules should not apply to PDF output")
}

func TestStylesheet_PrintMediaWidthQueriesApplyAgainstContentWidth(t *testing.T) {
	t.Parallel()
	html := `<html><head><style>
p { color: red }
@media print and (min-width: 600px) { p { color: #00ff00 } }
@media print and (max-width: 500px) { p { color: #0000ff } }
</style></head><body><p>text</p></body></html>`

	doc, err := dom.Parse(html)
	require.NoError(t, err)

	sheet := parseStylesheet(doc.StyleText())
	style := computeNodeStyle(sheet, findFirstNode(t, doc, "p"), nil)
	require.NotNil(t, style.Color)
	assert.Equal(t, 0, style.Color.R)
	assert.Equal(t, 255, style.Color.G)
	assert.Equal(t, 0, style.Color.B)
}

func TestStylesheet_PrintMediaWidthQueriesUseConfiguredContentWidth(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><head><style>
@media print and (max-width: 500px) { p::before { content:"narrow " } }
@media print and (min-width: 600px) { p::before { content:"wide " } }
</style></head><body><p>body</p></body></html>`)
	require.NoError(t, err)

	rows, err := Translate(context.Background(), doc, WithContentWidth(80))
	require.NoError(t, err)

	assert.Equal(t, []string{"narrow body"}, richTextValues(rows))
}

func TestStylesheet_PrintMediaPseudoRulesApply(t *testing.T) {
	t.Parallel()

	runs := runsFromHTML(t, `<style>
@media print { p::before { content:"Print: " } }
@media screen { p::after { content:" screen" } }
</style><p>body</p>`)
	require.Len(t, runs, 2)
	assert.Equal(t, "Print: ", runs[0].Text)
	assert.Equal(t, "body", runs[1].Text)
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

func TestBlockCellStyle_FilterDropShadowMapsToBoxShadow(t *testing.T) {
	t.Parallel()
	doc, err := dom.Parse(`<html><body><div style="filter:drop-shadow(2mm 3mm 4mm #00000080)"></div></body></html>`)
	require.NoError(t, err)

	style := computeNodeStyle(nil, findFirstNode(t, doc, "div"), nil)
	cell := (&translator{}).blockCellStyle(style)
	require.NotNil(t, cell)
	require.Len(t, cell.BoxShadow, 1)
	assert.InDelta(t, 2.0, cell.BoxShadow[0].OffsetX, 0.001)
	assert.InDelta(t, 3.0, cell.BoxShadow[0].OffsetY, 0.001)
	assert.InDelta(t, 4.0, cell.BoxShadow[0].BlurRadius, 0.001)
}

func TestParseInlineStyle_DataURIURLKeepsSemicolon(t *testing.T) {
	t.Parallel()

	decls := parseInlineStyle(`background-image:url("data:image/png;base64,abc"); padding:1mm`)

	assert.Equal(t, `url("data:image/png;base64,abc")`, decls["background-image"])
	assert.Equal(t, "1mm", decls["padding-top"])
}
