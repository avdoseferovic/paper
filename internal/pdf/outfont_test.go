package pdf

import (
	"fmt"
	"strings"
	"testing"
)

// TestOutFontSelect_MatchesFmtOutput pins outFontSelect to the fmt-based
// formatting it replaces, so the allocation optimization cannot silently change
// a single byte of emitted PDF.
func TestOutFontSelect_MatchesFmtOutput(t *testing.T) {
	sizes := []float64{
		0, 1, 8, 9, 9.5, 10, 10.005, 12.345, 0.001, 99.999,
		100, 1000.5, -3.25, 1e6, 0.125,
	}
	ids := []string{
		"1",
		"abc",
		"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	}

	for _, id := range ids {
		for _, size := range sizes {
			f := &PDF{}
			f.outFontSelect(id, size)
			got := f.buffer.String()

			want := fmt.Sprintf("BT /F%s %.2f Tf ET", id, size) + "\n"
			if got != want {
				t.Errorf("id=%q size=%v: got %q, want %q", id, size, got, want)
			}
		}
	}
}

// TestOutTextShow_MatchesFmtOutput pins outTextShow to the sprintf formulation
// it replaced in PDF.Text, including the colorFlag wrapping, so the fast path
// cannot alter a byte of emitted PDF.
func TestOutTextShow_MatchesFmtOutput(t *testing.T) {
	coords := []struct{ x, y float64 }{
		{0, 0}, {1, 1}, {10.5, 20.25}, {-3.5, 7.125}, {123.456, 654.321}, {0.001, 0.009},
	}
	texts := []string{"", "hello", "a b c", `esc\(paren\)`, "0123456789"}

	for _, colorFlag := range []bool{false, true} {
		for _, c := range coords {
			for _, txt := range texts {
				f := &PDF{k: 2.834, h: 297}
				f.colorFlag = colorFlag
				f.color.text.str = "0.500 g"
				f.outTextShow(c.x, c.y, txt)
				got := f.buffer.String()

				want := fmt.Sprintf("BT %.2f %.2f Td (%s) Tj ET", c.x*f.k, (f.h-c.y)*f.k, txt)
				if colorFlag {
					want = fmt.Sprintf("q %s %s Q", f.color.text.str, want)
				}
				want += "\n"

				if got != want {
					t.Errorf("colorFlag=%v x=%v y=%v txt=%q:\n got %q\nwant %q",
						colorFlag, c.x, c.y, txt, got, want)
				}
			}
		}
	}
}

// TestOutFontSelect_WritesToCurrentPage verifies the operator lands in the page
// content stream while rendering, matching out/outf routing.
func TestOutFontSelect_WritesToCurrentPage(t *testing.T) {
	f := NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", Size: SizeType{Wd: 100, Ht: 100}})
	f.AddPage()

	before := f.pages[f.page].Len()
	f.outFontSelect("abc", 12)
	after := f.pages[f.page].Len()

	if after <= before {
		t.Fatal("expected the font operator to be written to the current page")
	}
	if got := f.pages[f.page].String(); !strings.Contains(got, "BT /Fabc 12.00 Tf ET") {
		t.Errorf("page content missing font operator, got %q", got)
	}
}
