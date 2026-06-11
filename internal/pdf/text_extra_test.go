package pdf

import (
	"strings"
	"testing"
)

func TestWriteHandlesNewlinesAndUnbrokenOverflow(t *testing.T) {
	f := readyPDF(t)
	startY := f.GetY()

	f.Write(5, "first line\nsecond line")
	if f.GetY() <= startY {
		t.Fatal("expected explicit newline to advance the y position")
	}

	// Unbroken text first overflows mid-line (x > left margin), then wraps
	// repeatedly from the left margin.
	f.Write(5, "lead-in ")
	f.Write(5, strings.Repeat("a", 300))
	if f.Err() {
		t.Fatalf("Write errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestCellFormatVerticalAlignment(t *testing.T) {
	f := readyPDF(t)
	for _, align := range []string{"LT", "LB", "LA"} {
		f.CellFormat(40, 12, "aligned "+align, "", 1, align, false, 0, "")
	}
	if f.Err() {
		t.Fatalf("CellFormat errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestFontDescentFallsBackWithoutDescentMetric(t *testing.T) {
	f := readyPDF(t)
	if d := f.fontDescent(); d >= 0 {
		t.Fatalf("expected negative descent from font metrics, got %v", d)
	}

	f.currentFont.Desc.Descent = 0
	want := -0.19 * f.fontSize
	if d := f.fontDescent(); d != want {
		t.Fatalf("expected fallback descent %v, got %v", want, d)
	}
}

func TestCellFormatTriggersAutomaticPageBreak(t *testing.T) {
	f := readyPDF(t)
	f.SetAutoPageBreak(true, 20)
	f.SetY(285)
	f.ws = 1.5 // pending word spacing must be reset across the page break
	f.CellFormat(40, 20, "breaks onto a new page", "", 1, "L", false, 0, "")
	if f.Err() {
		t.Fatalf("CellFormat errored: %v", f.Error())
	}
	if f.PageCount() != 2 {
		t.Fatalf("expected automatic page break to add a page, got %d pages", f.PageCount())
	}
	mustOutput(t, f)
}

func TestSplitTextUnbrokenAndNewlineLines(t *testing.T) {
	f := readyPDF(t)

	lines := f.SplitText(strings.Repeat("a", 120), 30)
	if len(lines) < 2 {
		t.Fatalf("expected unbroken text to be split across lines, got %d", len(lines))
	}
	for _, line := range lines {
		if line == "" {
			t.Fatal("expected non-empty split lines")
		}
	}

	lines = f.SplitText("ab\ncd ef gh ij kl", 18)
	if len(lines) < 2 {
		t.Fatalf("expected newline and width splits, got %v", lines)
	}
	if lines[0] != "ab" {
		t.Fatalf("expected first line to end at the newline, got %q", lines[0])
	}
}

func TestBookmarkSiblingsAndOutlineOutput(t *testing.T) {
	f := readyPDF(t)
	f.SetCompression(false)
	f.Bookmark("Chapter 1", 0, -1)
	f.Bookmark("Section 1.1", 1, -1)
	f.Bookmark("Section 1.2", 1, -1)
	f.Bookmark("Chapter 2", 0, 10)

	out := string(mustOutput(t, f))
	for _, marker := range []string{"/Outlines", "(Chapter 1)", "/Prev ", "/Next ", "/First ", "/Last "} {
		if !strings.Contains(out, marker) {
			t.Fatalf("expected outline output to contain %q", marker)
		}
	}
}

func TestMultiCellJustifiedRTLUsesRightAlignment(t *testing.T) {
	f := readyUTF8PDF(t)
	f.RTL()
	f.MultiCell(60, 5, "many short words that wrap across multiple lines in this cell", "", "J", false)
	if f.Err() {
		t.Fatalf("MultiCell errored: %v", f.Error())
	}

	f.LTR()
	f.MultiCell(60, 5, "many short words that wrap across multiple lines in this cell\nwith newline", "", "J", false)
	if f.Err() {
		t.Fatalf("MultiCell errored: %v", f.Error())
	}
	mustOutput(t, f)
}
