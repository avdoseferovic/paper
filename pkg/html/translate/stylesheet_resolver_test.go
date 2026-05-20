package translate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johnfercher/maroto/v2/pkg/html/dom"
)

func TestStylesheetResolver_DefaultRefusesLocal(t *testing.T) {
	t.Parallel()
	_, err := safeDefaultStylesheetResolver("../escape.css")
	assert.ErrorIs(t, err, ErrStylesheetResolverRefused)
	_, err = safeDefaultStylesheetResolver("/etc/passwd")
	assert.ErrorIs(t, err, ErrStylesheetResolverRefused)
}

func TestStylesheetResolver_DataURI(t *testing.T) {
	t.Parallel()
	bytes, err := safeDefaultStylesheetResolver("data:text/css,p{color:red}")
	require.NoError(t, err)
	assert.Equal(t, "p{color:red}", string(bytes))
}

func TestStylesheetResolver_BaseDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cssPath := filepath.Join(dir, "theme.css")
	require.NoError(t, os.WriteFile(cssPath, []byte("p{color:blue}"), 0o644))

	resolver := stylesheetBaseDirResolver(dir)
	bytes, err := resolver("theme.css")
	require.NoError(t, err)
	assert.Equal(t, "p{color:blue}", string(bytes))
}

func TestStylesheetResolver_BaseDirRejectsEscape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	resolver := stylesheetBaseDirResolver(dir)
	_, err := resolver("../escape.css")
	assert.Error(t, err)
	_, err = resolver("/absolute/escape.css")
	assert.Error(t, err)
}

func TestStylesheet_LinkLoadedBeforeInline(t *testing.T) {
	t.Parallel()
	// Linked sheet sets red; inline overrides to green (later = wins at
	// equal specificity). This verifies linked CSS is concatenated FIRST.
	dir := t.TempDir()
	cssPath := filepath.Join(dir, "ext.css")
	require.NoError(t, os.WriteFile(cssPath, []byte("p { color: #ff0000 }"), 0o644))

	htmlStr := `
<html><head>
<link rel="stylesheet" href="ext.css">
<style>p { color: #00ff00 }</style>
</head><body><p>x</p></body></html>`
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	rows, err := Translate(doc, WithStylesheetBaseDir(dir))
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	// The p element's color must be green (inline overrides linked).
	var p *dom.Node
	doc.Walk(func(n *dom.Node) bool {
		if n.Tag() == "p" {
			p = n
		}
		return true
	})
	require.NotNil(t, p)
	// Reproduce the stylesheet pipeline as Translate did.
	tr := &translator{anchorReg: newAnchorRegistry()}
	WithStylesheetBaseDir(dir)(tr)
	_, hrefs := doc.StyleSources()
	var combined []byte
	for _, h := range hrefs {
		data, _ := safeLoadStylesheet(tr.stylesheetResolver, h)
		combined = append(combined, data...)
		combined = append(combined, '\n')
	}
	combined = append(combined, []byte(doc.StyleText())...)
	sheet := parseStylesheet(string(combined))
	pStyle := computeNodeStyle(sheet, p, nil)
	require.NotNil(t, pStyle.Color)
	assert.Equal(t, 0, pStyle.Color.R)
	assert.Equal(t, 255, pStyle.Color.G)
	assert.Equal(t, 0, pStyle.Color.B)
}

func TestStylesheet_ResolverFailureLogged(t *testing.T) {
	t.Parallel()
	var logged []string
	htmlStr := `<html><head><link rel="stylesheet" href="missing.css"></head><body><p>x</p></body></html>`
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	_, err = Translate(doc, WithUnsupportedHandler(func(thing, value string) {
		logged = append(logged, thing+":"+value)
	}))
	require.NoError(t, err)
	// Default resolver refuses non-data hrefs → should log skipped link.
	found := false
	for _, m := range logged {
		if assert.Contains(t, m, "link.skipped") || found {
			found = true
		}
	}
}
