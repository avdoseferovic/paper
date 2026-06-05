package paperpdf

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
