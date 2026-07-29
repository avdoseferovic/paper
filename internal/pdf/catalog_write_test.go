package pdf

import (
	"bytes"
	"testing"
)

func newCatalogTestPDF(t *testing.T, pages int) *PDF {
	t.Helper()
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.SetCompression(false)
	for range pages {
		f.AddPage()
	}
	f.SetFont("Helvetica", "", 12)
	f.Cell(40, 10, "catalog")
	return f
}

func assertContainsAll(t *testing.T, out []byte, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("expected output to contain %q", want)
		}
	}
}

func TestSetViewerPreferencesEmitsCatalogEntries(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 2)
	f.SetViewerPreferences(ViewerPreferences{
		PageLayout:      "TwoPageRight",
		PageMode:        "UseThumbs",
		HideToolbar:     true,
		DisplayDocTitle: true,
		OpenPage:        1,
		OpenZoom:        "FitH",
	})
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/PageLayout /TwoPageRight",
		"/PageMode /UseThumbs",
		"/HideToolbar true",
		"/DisplayDocTitle true",
		"/ViewerPreferences <<",
		"/OpenAction [5 0 R /FitH null]",
	)
}

func TestViewerPreferencesZeroOpenPageAloneEmitsNoOpenAction(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetViewerPreferences(ViewerPreferences{HideMenubar: true})
	out := mustOutput(t, f)

	if bytes.Contains(out, []byte("/OpenAction")) {
		t.Fatal("unexpected /OpenAction for zero-value OpenPage")
	}
	assertContainsAll(t, out, "/HideMenubar true")
}

func TestViewerPreferencesPercentZoomOpenAction(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetViewerPreferences(ViewerPreferences{OpenZoom: "125%"})
	out := mustOutput(t, f)

	assertContainsAll(t, out, "/OpenAction [3 0 R /XYZ null null 1.25]")
}

func TestSetPageLabelsEmitsNumberTree(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 2)
	f.SetPageLabels(
		PageLabelRange{PageIndex: 1, Style: "D", Start: 1},
		PageLabelRange{PageIndex: 0, Style: "r", Prefix: "front-"},
	)
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/PageLabels << /Nums [",
		"0 << /S /r /P (front-) >>",
		"1 << /S /D /St 1 >>",
	)
}

func TestSetAttachmentsEmitsEmbeddedFiles(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetAttachments(FileAttachment{
		FileName:       "invoice.xml",
		MIMEType:       "application/xml",
		Description:    "Invoice XML",
		AFRelationship: "Data",
		Data:           []byte("<invoice/>"),
	})
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/Type /EmbeddedFile",
		"/Subtype /application#2Fxml",
		"<invoice/>",
		"/Type /Filespec",
		"/F (invoice.xml)",
		"/Desc (Invoice XML)",
		"/AFRelationship /Data",
		"/EF <<",
		"/EmbeddedFiles << /Names [(invoice.xml)",
		"/AF [",
	)
}

func TestSetAttachmentsDefaultsMIMEAndRelationship(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetAttachments(FileAttachment{FileName: "raw.bin", Data: []byte{1, 2, 3}})
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/Subtype /application#2Foctet-stream",
		"/AFRelationship /Unspecified",
	)
}

func TestSetNamedDestinationsEmitsSortedDestsNameTree(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 3)
	f.SetNamedDestinations(
		NamedDestination{Name: "section", PageIndex: 2, FitType: "FitH", Top: 700},
		NamedDestination{Name: "top", PageIndex: 0},
		NamedDestination{Name: "detail", PageIndex: 1, FitType: "XYZ", Left: 10, Top: 20, Zoom: 1.5},
		NamedDestination{Name: "", PageIndex: 0},
		NamedDestination{Name: "invalid", PageIndex: 9},
	)
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/Dests << /Names [(detail) [5 0 R /XYZ 10.00 20.00 1.50] (section) [7 0 R /FitH 700.00] (top) [3 0 R /Fit]] >>",
	)
	if bytes.Contains(out, []byte("(invalid)")) {
		t.Fatal("destination with out-of-range page index should be skipped")
	}
}

func TestSetPageAnnotationsEmitsAnnotationDicts(t *testing.T) {
	t.Parallel()

	destPage := 1
	f := newCatalogTestPDF(t, 2)
	f.SetPageAnnotations(
		PageAnnotation{PageIndex: 0, Subtype: "Link", Rect: [4]float64{40, 50, 120, 70}, URI: "https://example.com"},
		PageAnnotation{PageIndex: 0, Subtype: "Link", Rect: [4]float64{40, 80, 120, 100}, DestName: "section"},
		PageAnnotation{PageIndex: 0, Subtype: "Link", Rect: [4]float64{40, 110, 120, 130}, DestPage: &destPage},
		PageAnnotation{PageIndex: 1, Subtype: "Text", Rect: [4]float64{10, 20, 30, 40}, Contents: "note body", Name: "Comment", Open: true},
		PageAnnotation{PageIndex: 1, Subtype: "Highlight", Rect: [4]float64{90, 100, 140, 120}, Color: &[3]float64{1, 1, 0}},
	)
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/Subtype /Link /Rect [40.00 50.00 120.00 70.00] /Border [0 0 0] /A <</S /URI /URI (https://example.com)>>",
		"/Dest (section)",
		"/Dest [5 0 R /Fit]",
		"/Subtype /Text /Rect [10.00 20.00 30.00 40.00] /Contents (note body) /Name /Comment /Open true",
		"/Subtype /Highlight /Rect [90.00 100.00 140.00 120.00] /QuadPoints [90.00 120.00 140.00 120.00 90.00 100.00 140.00 100.00] /C [1.00 1.00 0.00]",
	)
}

func TestTextAnnotationIconDefaultsToNote(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetPageAnnotations(PageAnnotation{PageIndex: 0, Subtype: "Text", Rect: [4]float64{1, 2, 3, 4}, Contents: "x"})
	out := mustOutput(t, f)

	assertContainsAll(t, out, "/Name /Note")
}

func TestSetPageGeometriesEmitsPageEntries(t *testing.T) {
	t.Parallel()

	crop := [4]float64{10, 20, 210, 290}
	trim := [4]float64{15, 25, 205, 285}
	f := newCatalogTestPDF(t, 2)
	f.SetPageGeometries(
		PageGeometry{PageIndex: 0, Rotate: 90, CropBox: &crop},
		PageGeometry{PageIndex: 1, Rotate: -90, TrimBox: &trim},
	)
	out := mustOutput(t, f)

	assertContainsAll(t, out,
		"/Rotate 90",
		"/Rotate 270",
		"/CropBox [10.00 20.00 210.00 290.00]",
		"/TrimBox [15.00 25.00 205.00 285.00]",
	)
}

func TestSetPageGeometriesSkipsInvalidRotation(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetPageGeometries(PageGeometry{PageIndex: 0, Rotate: 45})
	out := mustOutput(t, f)

	if bytes.Contains(out, []byte("/Rotate")) {
		t.Fatal("rotation not multiple of 90 should be skipped")
	}
}

func TestSetFileIDWritesTrailerID(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetFileID([]byte{0xab, 0xcd, 0xef})
	out := mustOutput(t, f)

	assertContainsAll(t, out, "/ID [<abcdef><abcdef>]")
}

func TestDeterministicOutputIsStable(t *testing.T) {
	t.Parallel()

	build := func() []byte {
		f := newCatalogTestPDF(t, 1)
		f.SetDeterministic(true)
		return mustOutput(t, f)
	}
	first := build()
	second := build()

	if !bytes.Equal(first, second) {
		t.Fatal("deterministic mode should produce identical output")
	}
	if !bytes.Contains(first, []byte("/ID [<")) {
		t.Fatal("deterministic mode should derive a trailer /ID")
	}
}

func TestExplicitFileIDBeatsDeterministicID(t *testing.T) {
	t.Parallel()

	f := newCatalogTestPDF(t, 1)
	f.SetDeterministic(true)
	f.SetFileID([]byte{0x01})
	out := mustOutput(t, f)

	assertContainsAll(t, out, "/ID [<01><01>]")
}
