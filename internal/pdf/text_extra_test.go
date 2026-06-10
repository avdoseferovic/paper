package pdf

import (
	"strings"
	"testing"
)

func TestGetStringWidthScalesWithFontSize(t *testing.T) {
	f := readyPDF(t)
	w12 := f.GetStringWidth("Hello")
	if w12 <= 0 {
		t.Fatalf("expected positive width, got %v", w12)
	}
	f.SetFontSize(24)
	w24 := f.GetStringWidth("Hello")
	if !floatNear(w24, w12*2, 1e-6) {
		t.Fatalf("width should double with font size: %v vs %v", w24, w12)
	}
}

func TestGetStringSymbolWidthEmptyString(t *testing.T) {
	f := readyPDF(t)
	if got := f.GetStringSymbolWidth(""); got != 0 {
		t.Fatalf("empty string width = %d", got)
	}
}

func TestCellAndCellf(t *testing.T) {
	f := readyPDF(t)
	f.Cell(40, 10, "Plain cell")
	f.Ln(10)
	f.Cellf(40, 10, "Value: %d", 42)
	f.Ln(10)
	f.CellFormat(60, 10, "Bordered", "1", 1, "C", true, 0, "")
	if f.Err() {
		t.Fatalf("cell errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestTextAndWrite(t *testing.T) {
	f := readyPDF(t)
	f.Text(20, 20, "Absolute text")
	f.SetXY(20, 40)
	f.Write(6, "Flowing write text that is reasonably long to wrap maybe")
	f.Writef(6, " formatted %d", 7)
	if f.Err() {
		t.Fatalf("text/write errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestMultiCellWrapsAndAligns(t *testing.T) {
	f := readyPDF(t)
	long := strings.Repeat("word ", 40)
	f.MultiCell(80, 6, long, "1", "J", false)
	f.MultiCell(80, 6, "Short centered", "0", "C", true)
	if f.Err() {
		t.Fatalf("multicell errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestSplitLines(t *testing.T) {
	f := readyPDF(t)
	lines := f.SplitLines([]byte(strings.Repeat("alpha beta gamma ", 10)), 60)
	if len(lines) < 2 {
		t.Fatalf("expected multiple split lines, got %d", len(lines))
	}
}

func TestSplitText(t *testing.T) {
	f := readyPDF(t)
	lines := f.SplitText(strings.Repeat("alpha beta gamma ", 10), 60)
	if len(lines) < 2 {
		t.Fatalf("expected multiple split lines, got %d", len(lines))
	}
}

func TestWriteAlignedVariants(t *testing.T) {
	f := readyPDF(t)
	for _, a := range []string{"L", "C", "R"} {
		f.SetXY(10, 10)
		f.WriteAligned(0, 6, "aligned "+a, a)
	}
	if f.Err() {
		t.Fatalf("WriteAligned errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestInternalLinks(t *testing.T) {
	f := readyPDF(t)
	link := f.AddLink()
	if link <= 0 {
		t.Fatalf("AddLink returned %d", link)
	}
	f.SetLink(link, 0, -1)
	f.Link(10, 10, 40, 10, link)
	f.WriteLinkID(6, "go to link", link)
	if f.Err() {
		t.Fatalf("internal link errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestExternalLinks(t *testing.T) {
	f := readyPDF(t)
	f.LinkString(10, 30, 40, 10, "https://example.com")
	f.SetXY(10, 50)
	f.WriteLinkString(6, "click here", "https://example.com")
	if f.Err() {
		t.Fatalf("external link errored: %v", f.Error())
	}
	mustOutput(t, f)
}

func TestBookmark(t *testing.T) {
	f := readyPDF(t)
	f.Bookmark("Chapter 1", 0, 0)
	f.Bookmark("Section 1.1", 1, 0)
	if f.Err() {
		t.Fatalf("bookmark errored: %v", f.Error())
	}
	out := mustOutput(t, f)
	if !strings.Contains(string(out), "Outlines") {
		t.Error("expected Outlines dictionary in output with bookmarks")
	}
}

func TestWordSpacingAndRenderingMode(t *testing.T) {
	f := readyPDF(t)
	f.SetWordSpacing(2)
	f.SetTextRenderingMode(1)
	f.Cell(40, 10, "spaced text")
	if f.Err() {
		t.Fatalf("word spacing / render mode errored: %v", f.Error())
	}
	mustOutput(t, f)
}
