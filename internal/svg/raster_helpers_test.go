package svg

import (
	"image/color"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/srwiley/oksvg"
)

func TestMMFromPx_RoundTripsPxFromMM(t *testing.T) {
	t.Parallel()

	px := pxFromMM(25.4)

	assert.Equal(t, int(DPIForRaster), px)
	assert.InDelta(t, 25.4, MMFromPx(px), 0.2)
}

func TestPxFromMM_ClampsToOnePixel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, pxFromMM(0))
	assert.Equal(t, 1, pxFromMM(-3))
}

func TestTargetPixels(t *testing.T) {
	t.Parallel()

	withViewBox := &oksvg.SvgIcon{}
	withViewBox.ViewBox.W = 100
	withViewBox.ViewBox.H = 50
	noViewBox := &oksvg.SvgIcon{}

	for name, tc := range map[string]struct {
		icon         *oksvg.SvgIcon
		wMM, hMM     float64
		wantW, wantH int
	}{
		"both dimensions":             {withViewBox, 25.4, 25.4, int(DPIForRaster), int(DPIForRaster)},
		"width only keeps ratio":      {withViewBox, 25.4, 0, int(DPIForRaster), int(DPIForRaster) / 2},
		"width only without viewbox":  {noViewBox, 25.4, 0, int(DPIForRaster), int(DPIForRaster)},
		"height only keeps ratio":     {withViewBox, 0, 25.4, int(DPIForRaster) * 2, int(DPIForRaster)},
		"height only without viewbox": {noViewBox, 0, 25.4, int(DPIForRaster), int(DPIForRaster)},
		"no dimensions uses viewbox":  {withViewBox, 0, 0, 100, 50},
		"no dimensions no viewbox":    {noViewBox, 0, 0, 32, 32},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			w, h := targetPixels(tc.icon, tc.wMM, tc.hMM)

			assert.Equal(t, tc.wantW, w)
			assert.Equal(t, tc.wantH, h)
		})
	}
}

func TestSvgScale(t *testing.T) {
	t.Parallel()

	icon := &oksvg.SvgIcon{}
	icon.ViewBox.W = 100
	icon.ViewBox.H = 50

	sx, sy := svgScale(icon, 200, 200)

	assert.InDelta(t, 2.0, sx, 0.0001)
	assert.InDelta(t, 4.0, sy, 0.0001)
}

func TestSvgScale_FallsBackToPixelSizeWithoutViewBox(t *testing.T) {
	t.Parallel()

	sx, sy := svgScale(&oksvg.SvgIcon{}, 64, 32)

	assert.InDelta(t, 1.0, sx, 0.0001)
	assert.InDelta(t, 1.0, sy, 0.0001)
}

func TestParseColor(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		value string
		want  color.RGBA
	}{
		"empty":         {"", color.RGBA{}},
		"none":          {"none", color.RGBA{}},
		"transparent":   {"transparent", color.RGBA{}},
		"black":         {"black", color.RGBA{A: 255}},
		"white":         {"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		"hex 6":         {"#ff8000", color.RGBA{R: 255, G: 128, B: 0, A: 255}},
		"hex 3":         {"#f80", color.RGBA{R: 255, G: 136, B: 0, A: 255}},
		"hex invalid":   {"#zzzzzz", color.RGBA{A: 255}},
		"rgb()":         {"rgb(10, 20, 30)", color.RGBA{R: 10, G: 20, B: 30, A: 255}},
		"unknown value": {"chartreuse", color.RGBA{A: 255}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, parseColor(tc.value))
		})
	}
}

func TestParseTransform(t *testing.T) {
	t.Parallel()

	t.Run("empty returns identity", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, identity(), parseTransform(""))
	})

	t.Run("translate", func(t *testing.T) {
		t.Parallel()

		got := parseTransform("translate(5, 7)")

		assert.InDelta(t, 5.0, got.e, 0.0001)
		assert.InDelta(t, 7.0, got.f, 0.0001)
	})

	t.Run("scale with single arg applies to both axes", func(t *testing.T) {
		t.Parallel()

		got := parseTransform("scale(3)")

		assert.InDelta(t, 3.0, got.a, 0.0001)
		assert.InDelta(t, 3.0, got.d, 0.0001)
	})

	t.Run("matrix", func(t *testing.T) {
		t.Parallel()

		got := parseTransform("matrix(1 2 3 4 5 6)")

		assert.Equal(t, affine{a: 1, b: 2, c: 3, d: 4, e: 5, f: 6}, got)
	})

	t.Run("composed transforms multiply", func(t *testing.T) {
		t.Parallel()

		got := parseTransform("translate(10, 0) scale(2)")

		assert.InDelta(t, 2.0, got.a, 0.0001)
		assert.InDelta(t, 10.0, got.e, 0.0001)
	})
}

func TestParseNumber(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 12.0, parseNumber("12px"), 0.0001)
	assert.InDelta(t, 10.5, parseNumber("10.5pt"), 0.0001)
	assert.InDelta(t, 3.0, parseNumber("3mm"), 0.0001)
	assert.InDelta(t, 2.0, parseNumber("2cm"), 0.0001)
	assert.InDelta(t, 0.0, parseNumber(""), 0.0001)
	assert.InDelta(t, 0.0, parseNumber("abc"), 0.0001)
}

func TestNumberAt(t *testing.T) {
	t.Parallel()

	numbers := []float64{1, 2}

	assert.InDelta(t, 2.0, numberAt(numbers, 1, 9), 0.0001)
	assert.InDelta(t, 9.0, numberAt(numbers, 5, 9), 0.0001)
}

func TestIsBold(t *testing.T) {
	t.Parallel()

	assert.True(t, isBold("bold"))
	assert.True(t, isBold("BOLDER"))
	assert.True(t, isBold("700"))
	assert.False(t, isBold("400"))
	assert.False(t, isBold(""))
	assert.False(t, isBold("normal"))
}

func TestFontFace_SelectsVariants(t *testing.T) {
	t.Parallel()

	for name, style := range map[string]textStyle{
		"regular":   {},
		"bold":      {fontWeight: "bold"},
		"mono":      {fontFamily: "monospace"},
		"mono bold": {fontFamily: "monospace", fontWeight: "bold"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			face, err := fontFace(style, 12)

			require.NoError(t, err)
			assert.NotNil(t, face)
		})
	}
}
