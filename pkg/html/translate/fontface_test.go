package translate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestParseFontFaceSrc_Truetype(t *testing.T) {
	t.Parallel()
	got, ok := parseFontFaceSrc(`url("./foo.ttf") format("truetype")`)
	assert.True(t, ok)
	assert.Equal(t, "./foo.ttf", got)
}

func TestParseFontFaceSrc_OpenType(t *testing.T) {
	t.Parallel()
	got, ok := parseFontFaceSrc(`url("foo.otf") format("opentype")`)
	assert.True(t, ok)
	assert.Equal(t, "foo.otf", got)
}

func TestParseFontFaceSrc_NoQuotes(t *testing.T) {
	t.Parallel()
	got, ok := parseFontFaceSrc(`url(foo.ttf) format(truetype)`)
	assert.True(t, ok)
	assert.Equal(t, "foo.ttf", got)
}

func TestParseFontFaceSrc_SkipLocal(t *testing.T) {
	t.Parallel()
	got, ok := parseFontFaceSrc(`local("Foo"), url("./foo.ttf") format("truetype")`)
	assert.True(t, ok)
	assert.Equal(t, "./foo.ttf", got)
}

func TestParseFontFaceSrc_SkipWoff(t *testing.T) {
	t.Parallel()
	_, ok := parseFontFaceSrc(`url("foo.woff") format("woff")`)
	assert.False(t, ok)
}

func TestExtractFontFace_ValidRule(t *testing.T) {
	t.Parallel()
	cssText := `@font-face { font-family: "MyFont"; src: url("./my.ttf") format("truetype") }`
	sheet, err := parser.Parse(cssText)
	require.NoError(t, err)
	require.NotEmpty(t, sheet.Rules)
	face, ok := extractFontFace(sheet.Rules[0])
	require.True(t, ok)
	assert.Equal(t, "myfont", face.family) // lowercased for predictable matching
	assert.Equal(t, "./my.ttf", face.srcURL)
}

func TestParseStylesheet_AdmitsFontFace(t *testing.T) {
	t.Parallel()
	// Verify the AtRule filter is changed: @font-face survives, @media still
	// dropped.
	ss := parseStylesheet(`
@font-face { font-family: "X"; src: url("./x.ttf") format("truetype") }
@media print { p { color: red } }
`)
	require.NotNil(t, ss)
	assert.Len(t, ss.fontFaces, 1, "@font-face rule should be captured")
}

func TestFontFace_EmitsFontRegistrationRow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fontPath := filepath.Join(dir, "fake.ttf")
	require.NoError(t, os.WriteFile(fontPath, []byte("FAKE_TTF_BYTES"), 0o644))

	htmlStr := `
<html><head><style>
@font-face { font-family: "MyFont"; src: url("fake.ttf") format("truetype") }
</style></head>
<body><p>x</p></body></html>`
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	rows, err := Translate(doc, WithStylesheetBaseDir(dir))
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	var foundReg bool
	for _, r := range rows {
		walkRowStructure(r.GetStructure(), func(s core.Structure) {
			if s.Type == "font_registration" {
				if name, _ := s.Details["family"].(string); name == "myfont" {
					foundReg = true
				}
			}
		})
	}
	assert.True(t, foundReg, "expected font_registration row for MyFont")
}

func TestFontFace_NoCrashWhenSourceMissing(t *testing.T) {
	t.Parallel()
	// Missing src file → resolver fails → font is skipped, no crash, no row.
	htmlStr := `
<html><head><style>
@font-face { font-family: "Gone"; src: url("missing.ttf") format("truetype") }
</style></head>
<body><p>ok</p></body></html>`
	doc, err := dom.Parse(htmlStr)
	require.NoError(t, err)
	_, err = Translate(doc, WithStylesheetBaseDir(t.TempDir()))
	require.NoError(t, err)
}
