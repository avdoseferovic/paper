package pdf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// utf8FontBytes loads the repository's UTF-8 test font fixture.
func utf8FontBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "assets", "fonts", "arial-unicode-ms.ttf"))
	if err != nil {
		t.Fatalf("read UTF-8 font fixture: %v", err)
	}
	return b
}

func readyUTF8PDF(t *testing.T) *PDF {
	t.Helper()
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
	f.AddUTF8FontFromBytes("arial", "", utf8FontBytes(t))
	f.AddPage()
	f.SetFont("arial", "", 12)
	if f.Err() {
		t.Fatalf("UTF-8 PDF setup failed: %v", f.Error())
	}
	return f
}

func TestUTF8StringWidthPositive(t *testing.T) {
	f := readyUTF8PDF(t)
	if w := f.GetStringWidth("Zdravo ćao"); w <= 0 {
		t.Fatalf("UTF-8 string width = %v", w)
	}
}

func TestUTF8JustifiedMultiCell(t *testing.T) {
	f := readyUTF8PDF(t)
	long := strings.Repeat("Zdravo ćao svijete ", 30)
	// Justified alignment exercises appendJustifiedUTF8CellText / blankCount.
	f.MultiCell(70, 6, long, "1", "J", false)
	if f.Err() {
		t.Fatalf("justified UTF-8 multicell errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestUTF8MultiCellExplicitNewlinesAndOverflow(t *testing.T) {
	f := readyUTF8PDF(t)
	text := "Line one\n" + strings.Repeat("supercalifragilisticexpialidocious", 4) + "\nLast line"
	f.MultiCell(50, 6, text, "0", "L", false)
	if f.Err() {
		t.Fatalf("multicell overflow errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestUTF8WriteWrapsAcrossLines(t *testing.T) {
	f := readyUTF8PDF(t)
	f.Write(6, strings.Repeat("wrapping words across the page width ", 20))
	if f.Err() {
		t.Fatalf("UTF-8 write errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestUTF8SplitText(t *testing.T) {
	f := readyUTF8PDF(t)
	lines := f.SplitText(strings.Repeat("alpha beta gamma délta ", 12), 50)
	if len(lines) < 2 {
		t.Fatalf("expected multiple lines, got %d", len(lines))
	}
}

func TestUnderlineAndStrikeoutRendering(t *testing.T) {
	f := readyPDF(t)
	f.SetFont("Helvetica", "U", 12)
	f.Cell(40, 10, "underlined")
	f.Ln(10)
	f.SetFont("Helvetica", "S", 12) // strikeout style
	f.Cell(40, 10, "struck out")
	if f.Err() {
		t.Fatalf("underline/strikeout errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestSubWriteOffsetsBaseline(t *testing.T) {
	f := readyPDF(t)
	f.SetXY(10, 30)
	f.Write(6, "H")
	f.SubWrite(6, "2", 8, 2, 0, "") // subscript style write
	f.Write(6, "O")
	if f.Err() {
		t.Fatalf("SubWrite errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestFontSizeHelpers(t *testing.T) {
	f := readyPDF(t)
	f.SetFontUnitSize(10)
	sizePt, sizeUnit := f.GetFontSize()
	if sizePt <= 0 || sizeUnit <= 0 {
		t.Fatalf("GetFontSize = %v, %v", sizePt, sizeUnit)
	}
	f.SetFontStyle("B")
	f.SetUnderlineThickness(2)
	f.Cell(40, 10, "bold")
	if f.Err() {
		t.Fatalf("font helpers errored: %v", f.Error())
	}
}

func TestGetFontDescCoreVsUTF8(t *testing.T) {
	// Core fonts do not carry a full descriptor; the lookup must still succeed.
	core := readyPDF(t)
	_ = core.GetFontDesc("Helvetica", "")
	if core.Err() {
		t.Fatalf("GetFontDesc for core font errored: %v", core.Error())
	}

	// An embedded UTF-8 font has real descriptor metrics.
	f := readyUTF8PDF(t)
	desc := f.GetFontDesc("arial", "")
	if desc.Ascent == 0 && desc.Descent == 0 {
		t.Fatal("expected non-zero descriptor metrics for embedded UTF-8 font")
	}
	// Empty family returns the current font's descriptor.
	if cur := f.GetFontDesc("", ""); cur.Ascent != desc.Ascent {
		t.Fatal("empty family should return current font descriptor")
	}
}
