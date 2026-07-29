package svg

import (
	"image"
	"testing"
)

// newTestPath builds a path on a 1:1 renderer so parsed user coordinates reach
// path.current unscaled and can be asserted directly.
func newTestPath() *svgPath {
	dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
	renderer := &svgRenderer{dst: dst, viewBox: svgViewBox{w: 64, h: 64}, scaleX: 1, scaleY: 1}
	return newSVGPath(renderer, identity())
}

func parsePath(t *testing.T, data string) (*svgPath, bool) {
	t.Helper()
	path := newTestPath()
	ok := path.parse(data)
	return path, ok
}

func assertPoint(t *testing.T, got svgPoint, wantX, wantY float64) {
	t.Helper()
	const tolerance = 1e-9
	if diff := got.x - wantX; diff > tolerance || diff < -tolerance {
		t.Fatalf("x = %v, want %v", got.x, wantX)
	}
	if diff := got.y - wantY; diff > tolerance || diff < -tolerance {
		t.Fatalf("y = %v, want %v", got.y, wantY)
	}
}

func TestParsePathCurrentPointPerCommand(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		data   string
		wantX  float64
		wantY  float64
		wantOK bool
	}{
		"moveto lineto absolute":    {"M10 10 L20 30", 20, 30, true},
		"moveto implies lineto":     {"M10 10 20 30", 20, 30, true},
		"relative moveto implies l": {"m10 10 5 5", 15, 15, true},
		"horizontal absolute":       {"M10 10 H40", 40, 10, true},
		"horizontal relative":       {"M10 10 h5", 15, 10, true},
		"vertical absolute":         {"M10 10 V40", 10, 40, true},
		"vertical relative":         {"M10 10 v5", 10, 15, true},
		"cubic absolute":            {"M0 0 C10 0 10 20 20 20", 20, 20, true},
		"cubic relative":            {"M10 10 c10 0 10 20 20 20", 30, 30, true},
		"quadratic absolute":        {"M0 0 Q10 0 20 20", 20, 20, true},
		"quadratic relative":        {"M10 10 q10 0 20 20", 30, 30, true},
		"arc draws chord to end":    {"M0 0 A5 5 0 0 1 20 20", 20, 20, true},
		"arc relative":              {"M10 10 a5 5 0 0 1 10 10", 20, 20, true},
		"closepath returns start":   {"M10 10 L30 30 Z", 10, 10, true},
		"exponent notation":         {"M0 0 L1e1 2E1", 10, 20, true},
		"compact negative decimals": {"M0 0 L-.5-.5", -0.5, -0.5, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path, ok := parsePath(t, tc.data)

			if ok != tc.wantOK {
				t.Fatalf("parse(%q) = %v, want %v", tc.data, ok, tc.wantOK)
			}
			assertPoint(t, path.current, tc.wantX, tc.wantY)
		})
	}
}

// TestParsePathSmoothCubicReflectsControlPoint pins the S behaviour: after a
// cubic, the first control point is the reflection of the previous second
// control point through the current point.
func TestParsePathSmoothCubicReflectsControlPoint(t *testing.T) {
	t.Parallel()

	path, ok := parsePath(t, "M0 0 C0 10 10 10 10 0 S20 -10 20 0")
	if !ok {
		t.Fatal("expected the cubic + smooth cubic path to parse")
	}
	assertPoint(t, path.current, 20, 0)
	// The S command's own second control point becomes the new lastCubic.
	assertPoint(t, path.lines[len(path.lines)-1][1], 20, 0)
}

// TestParsePathSmoothCubicWithoutPrecedingCubic checks that S falls back to the
// current point when the previous command was not a cubic, rather than
// reflecting a stale control point.
func TestParsePathSmoothCubicWithoutPrecedingCubic(t *testing.T) {
	t.Parallel()

	path, ok := parsePath(t, "M0 0 L10 0 S20 -10 20 0")
	if !ok {
		t.Fatal("expected the lineto + smooth cubic path to parse")
	}
	assertPoint(t, path.current, 20, 0)
}

// TestParsePathSmoothQuadraticReflectsControlPoint pins the T behaviour after a
// quadratic, and the fallback when no quadratic precedes it.
func TestParsePathSmoothQuadraticReflectsControlPoint(t *testing.T) {
	t.Parallel()

	for name, data := range map[string]string{
		"after quadratic": "M0 0 Q10 10 20 0 T40 0",
		"after lineto":    "M0 0 L20 0 T40 0",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path, ok := parsePath(t, data)

			if !ok {
				t.Fatalf("expected %q to parse", data)
			}
			assertPoint(t, path.current, 40, 0)
		})
	}
}

func TestParsePathRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	for name, data := range map[string]string{
		"empty":                    "",
		"separators only":          " ,\t\n",
		"numbers before a command": "10 20 L30 40",
		"unknown command":          "M0 0 B10 10",
		"lineto missing operand":   "M0 0 L10",
		"cubic missing operands":   "M0 0 C10 0 10 20",
		"quadratic missing":        "M0 0 Q10",
		"arc missing operands":     "M0 0 A5 5 0 0 1 20",
		"horizontal without point": "H10",
		"vertical without point":   "V10",
		"smooth cubic no current":  "S10 10 20 20",
		"smooth quad no current":   "T10 10",
		// Closepath takes no arguments, so a number after one cannot be an
		// implicit repeat of it. Both of these used to loop forever: the second
		// is the fuzz crasher from FuzzRasterizeWithLimit, which spun on a Z
		// with no current point, and the first grew path.ops without bound
		// because its close() had a current point to append.
		"number after closepath":          "M0 0 L10 10 Z0 0",
		"numbers after leading closepath": "Z0 0L10 10C1 2 3 4 5 6Z",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, ok := parsePath(t, data); ok {
				t.Fatalf("parse(%q) = true, want false", data)
			}
		})
	}
}

// TestParsePathClosepathOnlyIsNotAPath guards the final hasCurrent check: a
// path that never established a current point is not renderable.
func TestParsePathClosepathOnlyIsNotAPath(t *testing.T) {
	t.Parallel()

	if _, ok := parsePath(t, "Z"); ok {
		t.Fatal(`parse("Z") = true, want false`)
	}
}

func TestScanPathNumber(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		value         string
		wantEnd       int
		wantHasDigits bool
	}{
		"integer":                 {"123", 3, true},
		"signed integer":          {"-42", 3, true},
		"plus sign":               {"+7", 2, true},
		"decimal":                 {"1.5", 3, true},
		"leading dot":             {".5", 2, true},
		"trailing dot":            {"1.", 2, true},
		"exponent":                {"1e3", 3, true},
		"signed exponent":         {"1e-3", 4, true},
		"exponent without digits": {"1e", 1, true},
		"sign only":               {"-", 1, false},
		"dot only":                {".", 1, false},
		"stops at command":        {"12L", 2, true},
		"stops at separator":      {"12 34", 2, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			end, hasDigits := scanPathNumber(tc.value, 0)

			if end != tc.wantEnd || hasDigits != tc.wantHasDigits {
				t.Fatalf("scanPathNumber(%q, 0) = (%d, %v), want (%d, %v)",
					tc.value, end, hasDigits, tc.wantEnd, tc.wantHasDigits)
			}
		})
	}
}

func TestScanPathTokens(t *testing.T) {
	t.Parallel()

	tokens := scanPathTokens("M10,20 l-.5 1e2 Z")

	want := []pathToken{
		{command: 'M'},
		{number: 10},
		{number: 20},
		{command: 'l'},
		{number: -0.5},
		{number: 100},
		{command: 'Z'},
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens (%v), want %d", len(tokens), tokens, len(want))
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d = %+v, want %+v", i, tokens[i], want[i])
		}
	}
}

// TestScanPathTokensSkipsGarbage checks that unparseable runs advance the
// scanner instead of looping forever.
func TestScanPathTokensSkipsGarbage(t *testing.T) {
	t.Parallel()

	tokens := scanPathTokens("M0 0 ...--.. L5 5")

	var commands []byte
	for _, token := range tokens {
		if token.command != 0 {
			commands = append(commands, token.command)
		}
	}
	if string(commands) != "ML" {
		t.Fatalf("commands = %q, want %q", commands, "ML")
	}
}
