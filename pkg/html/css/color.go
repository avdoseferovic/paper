package css

import (
	"math"
	"strconv"
	"strings"
)

// RGBColor holds a parsed CSS color with optional alpha.
// A is in the range [0, 1] where 0 = transparent and 1 = opaque.
// Colors created by NewRGBColor or ParseColor always have A = 1.0 unless
// an alpha channel was explicitly specified.
type RGBColor struct {
	R, G, B int
	A       float64 // 0 = transparent, 1 = opaque; default is 1.0
}

// NewRGBColor constructs an opaque RGBColor.
func NewRGBColor(r, g, b int) RGBColor {
	return RGBColor{R: r, G: g, B: b, A: 1.0}
}

// ParseColor parses common CSS color formats:
//   - Named colors (full CSS Color Level 4 set, ~147 entries)
//   - #rgb, #rrggbb, #rgba (4-digit), #rrggbbaa (8-digit)
//   - rgb(), rgba() — integer or percentage channels
//   - hsl(), hsla()
//   - "transparent" → {0,0,0,0}
//
// Returns nil for unrecognised or invalid input.
func ParseColor(val string) *RGBColor {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}

	lower := strings.ToLower(val)

	// Special keywords
	switch lower {
	case "transparent":
		return &RGBColor{R: 0, G: 0, B: 0, A: 0.0}
	case "currentcolor", "inherit", "initial", "unset":
		return nil
	}

	// Hex
	if strings.HasPrefix(val, "#") {
		return parseHexColor(val[1:])
	}

	// Functional notations
	if strings.HasPrefix(lower, "rgba(") {
		return parseRGBAFunc(val[5:])
	}
	if strings.HasPrefix(lower, "rgb(") {
		return parseRGBFunc(val[4:])
	}
	if strings.HasPrefix(lower, "hsla(") {
		return parseHSLAFunc(val[5:])
	}
	if strings.HasPrefix(lower, "hsl(") {
		return parseHSLFunc(val[4:])
	}

	// Named color
	if c, ok := namedColorTable[lower]; ok {
		cp := c
		return &cp
	}
	return nil
}

// parseHexColor parses a hex string (without the leading '#').
func parseHexColor(hex string) *RGBColor {
	hex = strings.ToLower(hex)
	switch len(hex) {
	case 3: // #rgb → #rrggbb
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		return parseHex6(hex, 1.0)
	case 4: // #rgba → #rrggbbaa
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2], hex[3], hex[3]})
		return parseHex8(hex)
	case 6:
		return parseHex6(hex, 1.0)
	case 8:
		return parseHex8(hex)
	}
	return nil
}

func parseHex6(hex string, a float64) *RGBColor {
	r, err1 := strconv.ParseInt(hex[0:2], 16, 32)
	g, err2 := strconv.ParseInt(hex[2:4], 16, 32)
	b, err3 := strconv.ParseInt(hex[4:6], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil
	}
	return &RGBColor{R: int(r), G: int(g), B: int(b), A: a}
}

func parseHex8(hex string) *RGBColor {
	c := parseHex6(hex[:6], 1.0)
	if c == nil {
		return nil
	}
	av, err := strconv.ParseInt(hex[6:8], 16, 32)
	if err != nil {
		return nil
	}
	c.A = float64(av) / 255.0
	return c
}

// parseRGBFunc parses the argument portion of rgb(…) — i.e. "255, 0, 0)".
func parseRGBFunc(args string) *RGBColor {
	args = stripTrailingParen(args)
	parts := splitCSSArgs(args)
	if len(parts) != 3 {
		return nil
	}
	r, ok1 := parseColorChannel(parts[0])
	g, ok2 := parseColorChannel(parts[1])
	b, ok3 := parseColorChannel(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return nil
	}
	return &RGBColor{R: clamp255(r), G: clamp255(g), B: clamp255(b), A: 1.0}
}

// parseRGBAFunc parses the argument portion of rgba(…).
func parseRGBAFunc(args string) *RGBColor {
	args = stripTrailingParen(args)
	parts := splitCSSArgs(args)
	if len(parts) != 4 {
		return nil
	}
	r, ok1 := parseColorChannel(parts[0])
	g, ok2 := parseColorChannel(parts[1])
	b, ok3 := parseColorChannel(parts[2])
	a, ok4 := parseAlphaChannel(parts[3])
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil
	}
	return &RGBColor{R: clamp255(r), G: clamp255(g), B: clamp255(b), A: a}
}

// parseHSLFunc parses hsl(hue, sat%, light%).
func parseHSLFunc(args string) *RGBColor {
	args = stripTrailingParen(args)
	parts := splitCSSArgs(args)
	if len(parts) != 3 {
		return nil
	}
	h, ok1 := parseFloat(parts[0])
	s, ok2 := parsePctOrFloat(parts[1])
	l, ok3 := parsePctOrFloat(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return nil
	}
	r, g, b := hslToRGB(h, s, l)
	return &RGBColor{R: r, G: g, B: b, A: 1.0}
}

// parseHSLAFunc parses hsla(hue, sat%, light%, alpha).
func parseHSLAFunc(args string) *RGBColor {
	args = stripTrailingParen(args)
	parts := splitCSSArgs(args)
	if len(parts) != 4 {
		return nil
	}
	h, ok1 := parseFloat(parts[0])
	s, ok2 := parsePctOrFloat(parts[1])
	l, ok3 := parsePctOrFloat(parts[2])
	a, ok4 := parseAlphaChannel(parts[3])
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil
	}
	r, g, b := hslToRGB(h, s, l)
	return &RGBColor{R: r, G: g, B: b, A: a}
}

// hslToRGB converts HSL (h in [0,360), s and l in [0,1]) to RGB [0,255].
func hslToRGB(h, s, l float64) (int, int, int) {
	if s == 0 {
		v := int(math.Round(l * 255))
		return v, v, v
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	h /= 360
	r := hueToRGB(p, q, h+1.0/3)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3)
	return int(math.Round(r * 255)), int(math.Round(g * 255)), int(math.Round(b * 255))
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 1.0/2:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	default:
		return p
	}
}

// parseColorChannel parses an RGB channel: integer 0-255 or "50%" → 50*255/100.
func parseColorChannel(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, false
		}
		return v * 255.0 / 100.0, true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseAlphaChannel parses an alpha value: 0-1 float or "50%".
func parseAlphaChannel(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, false
		}
		return clamp01(v / 100.0), true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return clamp01(v), true
}

// parsePctOrFloat parses a value that should be a percentage (0%-100%) or a [0,1] float.
func parsePctOrFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, false
		}
		return clamp01(v / 100.0), true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return clamp01(v), true
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func clamp255(v float64) int {
	i := int(math.Round(v))
	if i < 0 {
		return 0
	}
	if i > 255 {
		return 255
	}
	return i
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func stripTrailingParen(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimSuffix(s, ")")
}

// splitCSSArgs splits "255, 0, 0" or "255,0,0" on commas, trimming whitespace.
func splitCSSArgs(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
