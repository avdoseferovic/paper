package pdf

import (
	"bytes"
	"regexp"
	"testing"
)

func TestSetPdfAEmitsMetadataAndOutputIntent(t *testing.T) {
	f := readyPDF(t)
	f.SetTitle("PDF/A Title", false)
	f.SetPdfA(ConformanceConfig{Level: ConformanceA2B})
	out := mustOutput(t, f)

	for _, want := range []string{
		"/OutputIntents [",
		"/S /GTS_PDFA1",
		"<pdfaid:part>2</pdfaid:part>",
		"<pdfaid:conformance>B</pdfaid:conformance>",
		"/Type /Metadata",
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("expected PDF/A output to contain %q", want)
		}
	}
	if n := len(regexp.MustCompile(`/Metadata \d+ 0 R`).FindAll(out, -1)); n != 1 {
		t.Fatalf("expected exactly one catalog /Metadata reference, found %d", n)
	}
}

func TestSetPdfALevelAEnablesTaggedOutput(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetPdfA(ConformanceConfig{Level: ConformanceA1A})
	f.AddPage()
	f.SetFont("Helvetica", "", 12)
	f.Cell(40, 10, "accessible")
	out := mustOutput(t, f)

	if !bytes.Contains(out, []byte("<pdfaid:conformance>A</pdfaid:conformance>")) {
		t.Fatal("missing level A conformance in XMP")
	}
	if !regexp.MustCompile(`/StructTreeRoot \d+ 0 R`).Match(out) {
		t.Fatal("level A PDF/A must emit a structure tree")
	}
}

func TestSetPdfAOverridesCustomXmpWithSingleMetadataRef(t *testing.T) {
	f := readyPDF(t)
	f.SetXmpMetadata([]byte(`<x:xmpmeta xmlns:x="adobe:ns:meta/"></x:xmpmeta>`))
	f.SetPdfA(ConformanceConfig{Level: ConformanceA3B})
	out := mustOutput(t, f)

	if !bytes.Contains(out, []byte("<pdfaid:part>3</pdfaid:part>")) {
		t.Fatal("expected the PDF/A XMP packet to replace the custom XMP stream")
	}
	if n := len(regexp.MustCompile(`/Metadata \d+ 0 R`).FindAll(out, -1)); n != 1 {
		t.Fatalf("expected exactly one catalog /Metadata reference, found %d", n)
	}
}
