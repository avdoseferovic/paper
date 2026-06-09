package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomUTF8FontPDFContainsCIDObjects(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "", 12)
	pdf.Write(5, "Zdravo, ćao 漢字")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		t.Fatalf("output custom font PDF: %v", err)
	}

	body := out.String()
	for _, marker := range []string{
		"%PDF",
		"/Subtype /CIDFontType2",
		"/W [",
		"/FontFile2",
		"/ToUnicode",
		"/CIDToGIDMap",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected custom-font PDF output to contain %q", marker)
		}
	}
}

func TestOutputReturnsErrorWhenUTF8FontSubsettingFails(t *testing.T) {
	fontBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read custom font fixture: %v", err)
	}

	pdf := NewCustom(&InitType{})
	pdf.AddUTF8FontFromBytes("arial-unicode-ms", "", fontBytes)
	if pdf.Error() != nil {
		t.Fatalf("register custom font: %v", pdf.Error())
	}

	font := pdf.fonts["arial-unicode-ms"]
	font.utf8File.fileReader.array = font.utf8File.fileReader.array[:64]

	pdf.AddPage()
	pdf.SetFont("arial-unicode-ms", "", 12)
	pdf.Write(5, "ćao")

	var out bytes.Buffer
	err = pdf.Output(&out)
	if err == nil {
		t.Fatal("expected Output to report UTF-8 font subsetting failure")
	}
	if !strings.Contains(err.Error(), "font") {
		t.Fatalf("expected font subsetting error, got %v", err)
	}
}

func TestUTF8FontFileGlyphDataRejectsOneByteGlyph(t *testing.T) {
	utf := &utf8FontFile{}

	_, ok := utf.glyphData([]byte{0x80}, 0, 1)

	if ok {
		t.Fatal("expected one-byte glyph to be rejected")
	}
	if utf.err == nil {
		t.Fatal("expected glyph error")
	}
}

func TestUTF8FontFileCompositeGlyphRejectsTruncatedComponent(t *testing.T) {
	utf := &utf8FontFile{}
	data := []byte{
		0x80, 0x00,
		0, 0, 0, 0, 0, 0, 0, 0,
		0x00, symbolContinue,
	}

	utf.rewriteCompositeGlyph(data, 1, map[int]int{})

	if utf.err == nil {
		t.Fatal("expected truncated composite glyph error")
	}
}
