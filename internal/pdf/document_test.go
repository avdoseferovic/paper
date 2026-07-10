package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestXYGettersAndSetters(t *testing.T) {
	f := readyPDF(t)
	f.SetXY(30, 40)
	x, y := f.GetXY()
	if x != 30 || y != 40 {
		t.Fatalf("GetXY = %v, %v", x, y)
	}
	f.SetX(50)
	if f.GetX() != 50 {
		t.Fatalf("GetX = %v", f.GetX())
	}
	f.SetY(60) // SetY resets X to left margin
	if f.GetY() != 60 {
		t.Fatalf("GetY = %v", f.GetY())
	}
	f.SetXY(15, 25)
	f.SetHomeXY() // resets to left/top margins
	if f.GetX() != f.lMargin || f.GetY() != f.tMargin {
		t.Fatalf("SetHomeXY did not reset to margins: %v,%v", f.GetX(), f.GetY())
	}
}

func TestNegativeXYResolvesFromPageEdges(t *testing.T) {
	f := readyPDF(t)
	f.SetX(-20)
	if f.GetX() <= 0 {
		t.Fatalf("negative SetX should resolve from right edge, got %v", f.GetX())
	}
	f.SetY(-20)
	if f.GetY() <= 0 {
		t.Fatalf("negative SetY should resolve from bottom edge, got %v", f.GetY())
	}
}

func TestGetConversionRatio(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "pt"})
	if got := f.GetConversionRatio(); got != 1 {
		t.Fatalf("conversion ratio for pt = %v, want 1", got)
	}
}

func TestDocumentMetadataSetters(t *testing.T) {
	f := readyPDF(t)
	f.SetTitle("My Title", true)
	f.SetSubject("Subject", false)
	f.SetAuthor("Author", true)
	f.SetKeywords("k1 k2", false)
	f.SetCreator("Creator", true)
	f.SetCreationDate(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
	f.SetModificationDate(time.Date(2021, 2, 3, 4, 5, 6, 0, time.UTC))
	if f.Err() {
		t.Fatalf("metadata setters errored: %v", f.Error())
	}
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("/Title")) {
		t.Error("expected /Title in output")
	}
}

func TestXmpMetadataEmitsStream(t *testing.T) {
	f := readyPDF(t)
	xmp := []byte(`<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/"></x:xmpmeta><?xpacket end="w"?>`)
	f.SetXmpMetadata(xmp)
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("Metadata")) {
		t.Error("expected Metadata object for XMP")
	}
}

func TestInfoDatesIncludeTimezoneOffset(t *testing.T) {
	f := readyPDF(t)
	f.SetCreationDate(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC))
	f.SetModificationDate(time.Date(2021, 2, 3, 4, 5, 6, 0, time.FixedZone("IST", 5*3600+30*60)))
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("/CreationDate (D:20200102030405Z)")) {
		t.Errorf("expected UTC creation date with Z suffix, got %s",
			regexp.MustCompile(`/CreationDate [^\n]*`).Find(out))
	}
	if !bytes.Contains(out, []byte("/ModDate (D:20210203040506+05'30')")) {
		t.Errorf("expected mod date with +05'30' offset, got %s",
			regexp.MustCompile(`/ModDate [^\n]*`).Find(out))
	}
}

func TestInfoDatesNegativeOffset(t *testing.T) {
	f := readyPDF(t)
	f.SetCreationDate(time.Date(2020, 1, 2, 3, 4, 5, 0, time.FixedZone("EST", -5*3600)))
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("/CreationDate (D:20200102030405-05'00')")) {
		t.Errorf("expected creation date with -05'00' offset, got %s",
			regexp.MustCompile(`/CreationDate [^\n]*`).Find(out))
	}
}

func TestXmpMetadataReferencedFromCatalog(t *testing.T) {
	f := readyPDF(t)
	f.SetXmpMetadata([]byte(`<x:xmpmeta xmlns:x="adobe:ns:meta/"></x:xmpmeta>`))
	out := mustOutput(t, f)

	m := regexp.MustCompile(`(\d+) 0 obj\n<< /Type /Metadata`).FindSubmatch(out)
	if m == nil {
		t.Fatal("missing metadata stream object")
	}
	ref := []byte("/Metadata " + string(m[1]) + " 0 R")
	if !bytes.Contains(out, ref) {
		t.Fatalf("catalog does not reference metadata object: want %q", ref)
	}
	if n := len(regexp.MustCompile(`/Metadata \d+ 0 R`).FindAll(out, -1)); n != 1 {
		t.Fatalf("expected exactly one catalog /Metadata reference, found %d", n)
	}
}

func TestSetLanguageEmitsLangInCatalog(t *testing.T) {
	f := readyPDF(t)
	f.SetLanguage("en-US")
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("/Lang (en-US)")) {
		t.Fatal("expected /Lang entry in catalog")
	}
}

func TestAliasNbPagesReplacedInOutput(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "mm"})
	f.AliasNbPages("")
	f.AddPage()
	f.SetFont("Helvetica", "", 12)
	f.Cell(40, 10, "Page 1 of {nb}")
	f.AddPage()
	f.Cell(40, 10, "Page 2 of {nb}")
	if f.Err() {
		t.Fatalf("alias errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestRTLAndLTRToggle(t *testing.T) {
	f := readyPDF(t)
	f.RTL()
	f.Cell(40, 10, "rtl")
	f.LTR()
	f.Cell(40, 10, "ltr")
	if f.Err() {
		t.Fatalf("RTL/LTR errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestSetCatalogSortAndJavascript(t *testing.T) {
	f := readyPDF(t)
	f.SetCatalogSort(true)
	f.SetJavascript("app.alert('hi');")
	out := mustOutput(t, f)
	if !bytes.Contains(out, []byte("JavaScript")) {
		t.Error("expected JavaScript in output")
	}
}

func TestRegisterAliasReplacement(t *testing.T) {
	f := readyPDF(t)
	f.RegisterAlias("{author}", "Jane Doe")
	f.Cell(40, 10, "By {author}")
	if f.Err() {
		t.Fatalf("RegisterAlias errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestRawWriteStrAndBuf(t *testing.T) {
	f := readyPDF(t)
	f.RawWriteStr("% raw comment\n")
	f.RawWriteBuf(bytes.NewBufferString("% raw buffer\n"))
	if f.Err() {
		t.Fatalf("raw write errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestOutputAndCloseWritesAll(t *testing.T) {
	f := readyPDF(t)
	f.Cell(40, 10, "close me")
	var buf bytes.Buffer
	wc := nopWriteCloser{&buf}
	if err := f.OutputAndClose(wc); err != nil {
		t.Fatalf("OutputAndClose: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("OutputAndClose produced no PDF header")
	}
}

func TestOutputFileAndClose(t *testing.T) {
	f := readyPDF(t)
	f.Cell(40, 10, "to file")
	path := filepath.Join(t.TempDir(), "out.pdf")
	if err := f.OutputFileAndClose(path); err != nil {
		t.Fatalf("OutputFileAndClose: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output file is not a PDF")
	}
}

type nopWriteCloser struct{ *bytes.Buffer }

func (nopWriteCloser) Close() error { return nil }
