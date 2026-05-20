package css

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColor_HexFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		r, g  int
		b     int
		a     float64
	}{
		{"#rgb", "#f00", 255, 0, 0, 1.0},
		{"#rrggbb", "#ff0000", 255, 0, 0, 1.0},
		{"#rrggbb lowercase", "#3a7bd5", 58, 123, 213, 1.0},
		{"#rgba 4-digit", "#f008", 255, 0, 0, float64(0x88) / 255.0},
		{"#rrggbbaa 8-digit", "#ff000080", 255, 0, 0, float64(0x80) / 255.0},
		{"#rrggbbff full opaque", "#ff0000ff", 255, 0, 0, 1.0},
		{"#rrggbb00 full transparent", "#ff000000", 255, 0, 0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ParseColor(tt.input)
			require.NotNil(t, c, "expected non-nil color for %q", tt.input)
			assert.Equal(t, tt.r, c.R)
			assert.Equal(t, tt.g, c.G)
			assert.Equal(t, tt.b, c.B)
			assert.InDelta(t, tt.a, c.A, 0.005)
		})
	}
}

func TestParseColor_RGB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		r, g  int
		b     int
		a     float64
	}{
		{"rgb integers", "rgb(255, 0, 0)", 255, 0, 0, 1.0},
		{"rgb no spaces", "rgb(10,20,30)", 10, 20, 30, 1.0},
		{"rgb percentages", "rgb(100%, 0%, 0%)", 255, 0, 0, 1.0},
		{"rgb 50%", "rgb(50%, 50%, 50%)", 128, 128, 128, 1.0},
		{"rgba with alpha", "rgba(0, 0, 0, 0.5)", 0, 0, 0, 0.5},
		{"rgba alpha 1", "rgba(255, 255, 255, 1)", 255, 255, 255, 1.0},
		{"rgba alpha 0", "rgba(0, 0, 0, 0)", 0, 0, 0, 0.0},
		{"rgba pct alpha", "rgba(0, 0, 0, 50%)", 0, 0, 0, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ParseColor(tt.input)
			require.NotNil(t, c, "expected non-nil for %q", tt.input)
			assert.Equal(t, tt.r, c.R)
			assert.Equal(t, tt.g, c.G)
			assert.Equal(t, tt.b, c.B)
			assert.InDelta(t, tt.a, c.A, 0.005)
		})
	}
}

func TestParseColor_HSL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		r, g  int
		b     int
		a     float64
	}{
		{"hsl red", "hsl(0, 100%, 50%)", 255, 0, 0, 1.0},
		{"hsl green", "hsl(120, 100%, 50%)", 0, 255, 0, 1.0},
		{"hsl blue", "hsl(240, 100%, 50%)", 0, 0, 255, 1.0},
		{"hsl dark green", "hsl(120, 100%, 25%)", 0, 128, 0, 1.0},
		{"hsl grey", "hsl(0, 0%, 50%)", 128, 128, 128, 1.0},
		{"hsla", "hsla(0, 100%, 50%, 0.5)", 255, 0, 0, 0.5},
		{"hsla dark green 80pct", "hsla(120, 100%, 25%, 0.8)", 0, 128, 0, 0.8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ParseColor(tt.input)
			require.NotNil(t, c, "expected non-nil for %q", tt.input)
			assert.InDelta(t, tt.r, c.R, 2)
			assert.InDelta(t, tt.g, c.G, 2)
			assert.InDelta(t, tt.b, c.B, 2)
			assert.InDelta(t, tt.a, c.A, 0.005)
		})
	}
}

func TestParseColor_NamedColors(t *testing.T) {
	t.Parallel()
	// Spot-check at least 10 CSS named colors
	tests := []struct {
		name string
		r, g int
		b    int
	}{
		{"red", 255, 0, 0},
		{"green", 0, 128, 0},
		{"blue", 0, 0, 255},
		{"white", 255, 255, 255},
		{"black", 0, 0, 0},
		{"yellow", 255, 255, 0},
		{"cyan", 0, 255, 255},
		{"magenta", 255, 0, 255},
		{"orange", 255, 165, 0},
		{"purple", 128, 0, 128},
		{"pink", 255, 192, 203},
		{"lavender", 230, 230, 250},
		{"tomato", 255, 99, 71},
		{"steelblue", 70, 130, 180},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := ParseColor(tt.name)
			require.NotNil(t, c, "expected non-nil for named color %q", tt.name)
			assert.Equal(t, tt.r, c.R)
			assert.Equal(t, tt.g, c.G)
			assert.Equal(t, tt.b, c.B)
			assert.Equal(t, 1.0, c.A)
		})
	}
	// Verify full count
	assert.GreaterOrEqual(t, len(namedColorTable), 140, "should have ~147 named colors")
}

func TestParseColor_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{"", "notacolor", "#gg0000", "rgb()", "hsl(bad)", "##red"}
	for _, s := range invalid {
		assert.Nil(t, ParseColor(s), "expected nil for invalid %q", s)
	}
}

func TestRGBColor_DefaultAlpha(t *testing.T) {
	t.Parallel()
	// Colors produced via NewRGBColor must have A=1.0
	c := NewRGBColor(100, 150, 200)
	assert.Equal(t, 100, c.R)
	assert.Equal(t, 150, c.G)
	assert.Equal(t, 200, c.B)
	assert.Equal(t, 1.0, c.A)
}

func TestHSLToRGB(t *testing.T) {
	t.Parallel()
	r, g, b := hslToRGB(0, 1.0, 0.5)
	assert.Equal(t, 255, r)
	assert.Equal(t, 0, g)
	assert.Equal(t, 0, b)

	r2, g2, b2 := hslToRGB(240, 1.0, 0.5)
	assert.Equal(t, 0, r2)
	assert.Equal(t, 0, g2)
	assert.Equal(t, 255, b2)

	// Achromatic
	r3, g3, b3 := hslToRGB(0, 0, 0.5)
	assert.InDelta(t, 128, r3, 1)
	assert.InDelta(t, 128, g3, 1)
	assert.InDelta(t, 128, b3, 1)
}

// Ensure ParseColor handles currentColor/inherit (returns nil — no rendering context available).
func TestParseColor_CurrentColor(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ParseColor("currentColor"))
	assert.Nil(t, ParseColor("inherit"))
}

func TestParseColor_Transparent(t *testing.T) {
	t.Parallel()
	c := ParseColor("transparent")
	require.NotNil(t, c)
	assert.Equal(t, 0, c.R)
	assert.Equal(t, 0, c.G)
	assert.Equal(t, 0, c.B)
	assert.InDelta(t, 0.0, c.A, 0.001)
}

// --- helpers used in tests only ---

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 0.01 }
