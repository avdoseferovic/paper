package svg_test

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/svg"
)

const sampleSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="120px" height="60px" viewBox="0 0 240 120" preserveAspectRatio="xMinYMax slice">
  <defs><circle id="dot" cx="5" cy="5" r="5"/></defs>
  <g id="layer"><rect x="0" y="0" width="240" height="120" fill="#f00"/><text>Hello</text></g>
</svg>`

func TestParseExposesDimensionsRootAndDefs(t *testing.T) {
	t.Parallel()

	doc, err := svg.Parse(sampleSVG)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := doc.Width(); got != 120 {
		t.Fatalf("Width() = %.2f, want 120", got)
	}
	if got := doc.Height(); got != 60 {
		t.Fatalf("Height() = %.2f, want 60", got)
	}
	if got := doc.AspectRatio(); got != 2 {
		t.Fatalf("AspectRatio() = %.2f, want 2", got)
	}

	viewBox := doc.ViewBox()
	if !viewBox.Valid || viewBox.Width != 240 || viewBox.Height != 120 {
		t.Fatalf("ViewBox() = %+v, want valid 240x120", viewBox)
	}

	root := doc.Root()
	if root == nil || root.Tag != "svg" {
		t.Fatalf("Root() = %+v, want svg node", root)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root children = %d, want defs and g", len(root.Children))
	}
	if root.Children[1].Children[1].Text != "Hello" {
		t.Fatalf("text child = %q, want Hello", root.Children[1].Children[1].Text)
	}

	defs := doc.Defs()
	if defs["dot"] == nil || defs["dot"].Tag != "circle" {
		t.Fatalf("Defs()[dot] = %+v, want circle", defs["dot"])
	}

	par := doc.PreserveAspectRatio()
	if par.Align != svg.AlignXMinYMax || par.MeetOrSlice != svg.ScaleSlice || par.None {
		t.Fatalf("PreserveAspectRatio() = %+v, want xMinYMax slice", par)
	}
}

func TestParseReaderFindsNestedSVG(t *testing.T) {
	t.Parallel()

	doc, err := svg.ParseReader(strings.NewReader(`<html><body><svg viewBox="0 0 10 20"/></body></html>`))
	if err != nil {
		t.Fatalf("ParseReader() error = %v", err)
	}
	if got := doc.Width(); got != 10 {
		t.Fatalf("Width() = %.2f, want viewBox fallback", got)
	}
	if got := doc.Height(); got != 20 {
		t.Fatalf("Height() = %.2f, want viewBox fallback", got)
	}
}

func TestParseBytesClonesInput(t *testing.T) {
	t.Parallel()

	data := []byte(`<svg width="10" height="10"/>`)
	doc, err := svg.ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	data[5] = 'X'
	if !bytes.Contains(doc.Bytes(), []byte("<svg")) {
		t.Fatalf("Bytes() changed after input mutation: %q", doc.Bytes())
	}
}

func TestRasterizeProducesPNG(t *testing.T) {
	t.Parallel()

	doc, err := svg.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 16"><rect width="32" height="16" fill="red"/></svg>`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pngBytes, width, height, err := doc.Rasterize(16, 0)
	if err != nil {
		t.Fatalf("Rasterize() error = %v", err)
	}
	if width <= 0 || height <= 0 {
		t.Fatalf("Rasterize() dimensions = %dx%d, want positive", width, height)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("PNG decode error = %v", err)
	}
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Fatalf("PNG bounds = %dx%d, want %dx%d", img.Bounds().Dx(), img.Bounds().Dy(), width, height)
	}
}

func TestComponentAdaptersUseOriginalSVGBytes(t *testing.T) {
	t.Parallel()

	doc, err := svg.Parse(`<svg width="10" height="10"/>`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	component := doc.Component(props.Rect{Percent: 50})
	structure := component.GetStructure().GetData()
	if structure.Type != "bytesImage" {
		t.Fatalf("component type = %q, want bytesImage", structure.Type)
	}
	if structure.Details["extension"] != extension.Svg {
		t.Fatalf("extension = %v, want svg", structure.Details["extension"])
	}

	if doc.Col(6).GetSize() != 6 {
		t.Fatal("Col(6) did not preserve requested column size")
	}
	if doc.Row(12) == nil || doc.AutoRow() == nil {
		t.Fatal("Row adapters must return rows")
	}
}

func TestParseInvalidDocumentReturnsError(t *testing.T) {
	t.Parallel()

	_, err := svg.Parse(`<div></div>`)
	if err == nil {
		t.Fatal("Parse() expected error for document without svg root")
	}
}
