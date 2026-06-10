package pdf

import "testing"

// loadUTF8FontFile parses the repository TTF fixture into a utf8FontFile with
// its glyf table and symbol positions populated.
func loadUTF8FontFile(t *testing.T) *utf8FontFile {
	t.Helper()
	reader := fileReader{readerPosition: 0, array: utf8FontBytes(t)}
	utf := newUTF8Font(&reader)
	if err := utf.parseFile(); err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	return utf
}

func TestParseGlyphOutlineProducesContours(t *testing.T) {
	utf := loadUTF8FontFile(t)
	if len(utf.symbolPosition) == 0 {
		t.Skip("font has no glyf symbol positions")
	}

	parsedAny := false
	emittedPath := false
	// Walk a bounded range of glyph IDs; many will be simple glyphs, and
	// composite glyphs (numContours < 0) are exercised opportunistically.
	limit := len(utf.symbolPosition) - 1
	if limit > 2000 {
		limit = 2000
	}
	for gid := 0; gid < limit; gid++ {
		outline := utf.parseGlyphOutline(uint16(gid))
		if outline == nil {
			continue
		}
		parsedAny = true
		if len(outline.contours) == 0 {
			continue
		}
		path := glyphOutlineToPDFPath(outline, 10, 10, 0.01, 72.0/25.4)
		if path != "" {
			emittedPath = true
			break
		}
	}
	if !parsedAny {
		t.Fatal("expected to parse at least one glyph outline")
	}
	if !emittedPath {
		t.Fatal("expected at least one glyph to emit PDF path operators")
	}
}

func TestParseGlyphOutlineOutOfRangeReturnsNil(t *testing.T) {
	utf := loadUTF8FontFile(t)
	if got := utf.parseGlyphOutline(uint16(len(utf.symbolPosition) + 100)); got != nil {
		t.Fatal("expected nil outline for out-of-range glyph id")
	}
}

func TestParseGlyphDataRejectsTruncatedHeader(t *testing.T) {
	utf := loadUTF8FontFile(t)
	if got := utf.parseGlyphData([]byte{0, 1, 2}, nil); got != nil {
		t.Fatal("expected nil for truncated glyph header (<10 bytes)")
	}
}

func TestGlyphOutlineToPDFPathEmptyOutline(t *testing.T) {
	if got := glyphOutlineToPDFPath(nil, 0, 0, 1, 1); got != "" {
		t.Fatalf("expected empty path for nil outline, got %q", got)
	}
	if got := glyphOutlineToPDFPath(&glyphOutline{}, 0, 0, 1, 1); got != "" {
		t.Fatalf("expected empty path for empty outline, got %q", got)
	}
}

func TestContourToPDFOpsTooFewPoints(t *testing.T) {
	if got := contourToPDFOps(glyphContour{{x: 1, y: 1, onCurve: true}}, 0, 0, 1, 1); got != "" {
		t.Fatalf("expected empty ops for single-point contour, got %q", got)
	}
}

func TestRead2Dot14(t *testing.T) {
	// 0x4000 == 1.0 in F2Dot14 fixed-point.
	if got := read2Dot14([]byte{0x40, 0x00}); !floatNear(got, 1.0, 1e-9) {
		t.Fatalf("read2Dot14(0x4000) = %v, want 1.0", got)
	}
	// 0x0000 == 0.0
	if got := read2Dot14([]byte{0x00, 0x00}); got != 0 {
		t.Fatalf("read2Dot14(0x0000) = %v, want 0", got)
	}
}
