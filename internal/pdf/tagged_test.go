package pdf

import (
	"bytes"
	"regexp"
	"testing"
)

func TestTaggedPDFEmitsStructureTree(t *testing.T) {
	t.Parallel()

	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetCompression(false)
	f.SetTaggedPDF(true)
	f.AddPage()
	f.SetFont("Helvetica", "", 12)
	f.Cell(40, 10, "page one")
	f.AddPage()
	f.Cell(40, 10, "page two")
	out := mustOutput(t, f)

	if !bytes.Contains(out, []byte("/MarkInfo << /Marked true >>")) {
		t.Fatal("catalog missing /MarkInfo << /Marked true >>")
	}
	if !regexp.MustCompile(`/StructTreeRoot \d+ 0 R`).Match(out) {
		t.Fatal("catalog missing /StructTreeRoot reference")
	}
	for _, want := range []string{
		"/Type /StructTreeRoot",
		"/S /Document",
		"/ParentTree",
		"/StructParents 0",
		"/StructParents 1",
		"/P << /MCID 0 >> BDC",
		"EMC",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("expected tagged PDF output to contain %q", want)
		}
	}
}

func TestUntaggedPDFHasNoStructureTree(t *testing.T) {
	t.Parallel()

	f := readyPDF(t)
	out := mustOutput(t, f)
	if bytes.Contains(out, []byte("/StructTreeRoot")) {
		t.Fatal("unexpected /StructTreeRoot without tagged PDF")
	}
	if bytes.Contains(out, []byte("/MarkInfo")) {
		t.Fatal("unexpected /MarkInfo without tagged PDF")
	}
}
