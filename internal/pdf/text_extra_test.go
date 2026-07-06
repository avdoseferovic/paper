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

func TestUTF8NonBreakingSpaceUsesSpaceAdvance(t *testing.T) {
	f := &PDF{
		isCurrentUTF8: true,
		currentFont: fontDefType{
			Cw: make([]int, 256*256),
		},
	}
	f.currentFont.Cw[' '] = 250
	f.currentFont.Cw['\u00a0'] = 65535
	f.fontSize = 10

	space := f.GetStringWidth(" ")
	nbsp := f.GetStringWidth("\u00a0")
	if nbsp != space {
		t.Fatalf("NBSP width = %v, want regular space width %v", nbsp, space)
	}
}

func TestUTF8TextFallsBackToRegisteredFontForUnsupportedRune(t *testing.T) {
	f := fallbackFontTestPDF()

	if got, want := f.GetStringWidth("A\u2264"), 13.0; got != want {
		t.Fatalf("fallback-aware width = %v, want %v", got, want)
	}

	f.Text(10, 20, "A\u2264A")

	body := f.buffer.String()
	if !strings.Contains(body, "/F1") || !strings.Contains(body, "/F2") {
		t.Fatalf("expected text operations to select both primary and fallback fonts, got:\n%s", body)
	}
	if _, ok := f.fonts["primary"].usedRunes[0x2264]; ok {
		t.Fatalf("primary font should not be assigned unsupported <= rune")
	}
	if got := f.fonts["fallback"].usedRunes[0x2264]; got != 0x2264 {
		t.Fatalf("fallback font used rune = %x, want 2264", got)
	}
}

func TestUTF8CellFormatFallsBackToRegisteredFontForUnsupportedRune(t *testing.T) {
	f := fallbackFontTestPDF()

	f.CellFormat(40, 10, "A\u2264A", "", 0, "L", false, 0, "")

	body := f.buffer.String()
	if !strings.Contains(body, "/F1") || !strings.Contains(body, "/F2") {
		t.Fatalf("expected cell text operations to select both primary and fallback fonts, got:\n%s", body)
	}
	if _, ok := f.fonts["primary"].usedRunes[0x2264]; ok {
		t.Fatalf("primary font should not be assigned unsupported <= rune")
	}
	if got := f.fonts["fallback"].usedRunes[0x2264]; got != 0x2264 {
		t.Fatalf("fallback font used rune = %x, want 2264", got)
	}
}

func fallbackFontTestPDF() *PDF {
	primary := fallbackTestFont("primary", "1", map[rune]int{
		'A': 600,
		' ': 250,
	})
	fallback := fallbackTestFont("fallback", "2", map[rune]int{
		'\u2264': 700,
	})
	return &PDF{
		k:                1,
		h:                100,
		page:             1,
		fontSize:         10,
		fontSizePt:       10,
		fonts:            map[string]fontDefType{"primary": primary, "fallback": fallback},
		currentFont:      primary,
		isCurrentUTF8:    true,
		x:                10,
		y:                10,
		pageBreakTrigger: 1000,
	}
}

func fallbackTestFont(name, id string, widths map[rune]int) fontDefType {
	cw := make([]int, 256*256)
	cmap := make(map[int]int, len(widths))
	for r, w := range widths {
		cw[int(r)] = w
		cmap[int(r)] = int(r)
	}
	return fontDefType{
		Tp:        fontTypeUTF8,
		Name:      name,
		i:         id,
		Cw:        cw,
		CwExtra:   map[int]int{},
		usedRunes: map[int]int{},
		runeToCID: map[int]int{},
		utf8File: &utf8FontFile{
			charSymbolDictionary: cmap,
		},
	}
}
