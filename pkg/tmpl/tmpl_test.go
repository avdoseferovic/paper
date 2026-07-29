package tmpl

import (
	"bytes"
	htmltpl "html/template"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
)

func TestRenderExecutesTemplateToRows(t *testing.T) {
	t.Parallel()

	rows, err := Render(t.Context(), `<p>Hello, {{.Name}}!</p>`, map[string]string{"Name": "Acme"}, nil)

	require.NoError(t, err)
	assert.True(t, len(rows) > 0)
}

func TestRenderSupportsCustomFuncsAndDict(t *testing.T) {
	t.Parallel()

	opts := &Options{
		Funcs: htmltpl.FuncMap{"upper": strings.ToUpper},
	}

	rows, err := Render(
		t.Context(),
		`{{$d := dict "name" .Name}}<p>{{upper $d.name}}</p>`,
		map[string]string{"Name": "acme"},
		opts,
	)

	require.NoError(t, err)
	assert.True(t, len(rows) > 0)
}

func TestRenderRejectsInvalidDictUsage(t *testing.T) {
	t.Parallel()

	_, err := Render(t.Context(), `{{dict "a" "b" "orphan"}}`, nil, nil)

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "odd number of arguments"))
}

func TestRenderUsesBaseTemplateClone(t *testing.T) {
	t.Parallel()

	base := htmltpl.Must(htmltpl.New("base").Parse(`{{define "line"}}<p>{{.}}</p>{{end}}`))

	rows, err := Render(
		t.Context(),
		`{{template "line" .Name}}`,
		map[string]string{"Name": "Acme"},
		&Options{BaseTemplate: base},
	)

	require.NoError(t, err)
	assert.True(t, len(rows) > 0)
}

func TestRenderDocumentProducesPDF(t *testing.T) {
	t.Parallel()

	doc, err := RenderDocument(
		t.Context(),
		`<style>@page { size: 120mm 90mm; margin: 8mm; }</style><h1>{{.Title}}</h1>`,
		map[string]string{"Title": "Invoice"},
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
}

func TestRenderToWritesPDF(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	err := RenderTo(t.Context(), &out, `<p>{{.}}</p>`, "ok", &Options{
		Config: config.NewBuilder().WithDimensions(80, 80).Build(),
	})

	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out.Bytes(), []byte("%PDF-")))
}

func TestRenderFileToUsesTemplateDirectoryForAssets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "pixel.png"))
	templatePath := filepath.Join(dir, "invoice.html")
	err := os.WriteFile(templatePath, []byte(`<p>{{.Name}}</p><img src="pixel.png">`), 0o600)
	require.NoError(t, err)

	var out bytes.Buffer
	err = RenderFileTo(t.Context(), &out, templatePath, map[string]string{"Name": "Acme"}, nil)

	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(out.Bytes(), []byte("%PDF-")))
	assert.True(t, bytes.Contains(out.Bytes(), []byte("/Subtype /Image")))
}

func TestRenderFileWritesPDF(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "simple.html")
	outPath := filepath.Join(dir, "out.pdf")
	err := os.WriteFile(templatePath, []byte(`<p>{{.}}</p>`), 0o600)
	require.NoError(t, err)

	err = RenderFile(t.Context(), templatePath, "ok", nil, outPath)

	require.NoError(t, err)
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := range 2 {
		for x := range 2 {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 10, B: 20, A: 255})
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))
}
