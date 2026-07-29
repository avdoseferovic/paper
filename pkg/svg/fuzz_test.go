package svg_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/svg"
)

const fuzzMaxPixels = 1 << 18

func svgSeeds() []string {
	return []string{
		"",
		"<svg/>",
		`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`,
		`<svg viewBox="0 0 10 10"><path d="M0 0 L10 10 C 1 2 3 4 5 6 Z"/></svg>`,
		`<svg viewBox="0 0 1 1"><g transform="matrix(1,2,3,4,5,6) rotate(45)"><circle cx=".5" cy=".5" r=".5"/></g></svg>`,
		`<svg><text x="0" y="0" style="fill:#abc;font-size:12">hi</text></svg>`,
		`<svg width="1e30" height="1e30"><rect width="1e30" height="1e30"/></svg>`,
		`<svg viewBox="0 0 NaN Infinity"><path d="M NaN NaN L 1 1"/></svg>`,
	}
}

// FuzzParseBytes drives the SVG document parser. Documents reach this path
// from HTML <img src> content, so malformed markup must return an error rather
// than panic.
func FuzzParseBytes(f *testing.F) {
	for _, seed := range svgSeeds() {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		parsed, err := svg.ParseBytes(data)
		if err != nil {
			if parsed != nil {
				t.Fatalf("ParseBytes() returned both a document and an error: %v", err)
			}
			return
		}
		if parsed == nil {
			t.Fatal("ParseBytes() returned no document and no error")
		}
	})
}

// FuzzRasterizeWithLimit asserts the pixel budget actually holds: the budget is
// the only thing standing between a few bytes of hostile SVG and an
// out-of-memory rasterisation.
func FuzzRasterizeWithLimit(f *testing.F) {
	for _, seed := range svgSeeds() {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, width, height, err := svg.RasterizeWithLimit(data, 50, 50, fuzzMaxPixels)
		if err != nil {
			return
		}
		if width < 0 || height < 0 {
			t.Fatalf("RasterizeWithLimit() returned negative dimensions %dx%d", width, height)
		}
		if int64(width)*int64(height) > fuzzMaxPixels {
			t.Fatalf("RasterizeWithLimit() produced %dx%d pixels, over the %d budget",
				width, height, fuzzMaxPixels)
		}
	})
}
