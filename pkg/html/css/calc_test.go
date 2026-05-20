package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalc_AddSubtract(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 15.0, ParseLength("calc(10mm + 5mm)", 0), 0.001)
	assert.InDelta(t, 15.0, ParseLength("calc(2cm - 5mm)", 0), 0.001)
}

func TestCalc_MultiplyDivide(t *testing.T) {
	t.Parallel()
	// 10pt = 3.52778mm; 10pt * 1.5 = 5.29mm
	assert.InDelta(t, 5.29, ParseLength("calc(10pt * 1.5)", 0), 0.05)
	assert.InDelta(t, 5.0, ParseLength("calc(10mm / 2)", 0), 0.001)
}

func TestCalc_Percent(t *testing.T) {
	t.Parallel()
	// 100% of contextWidth 170mm - 20mm = 150mm
	assert.InDelta(t, 150.0, ParseLengthCtx("calc(100% - 20mm)", 0, 170.0), 0.001)
	// 100%/4 of 160 = 40
	assert.InDelta(t, 40.0, ParseLengthCtx("calc(100% / 4)", 0, 160.0), 0.001)
}

func TestCalc_Parens(t *testing.T) {
	t.Parallel()
	// (10mm + 2mm) * 2 = 24mm
	assert.InDelta(t, 24.0, ParseLength("calc((10mm + 2mm) * 2)", 0), 0.001)
}

func TestCalc_LenientWhitespace(t *testing.T) {
	t.Parallel()
	// Browsers accept calc(100%-20mm) without spaces.
	assert.InDelta(t, 150.0, ParseLengthCtx("calc(100%-20mm)", 0, 170.0), 0.001)
}

func TestCalc_Malformed_ReturnsZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, ParseLength("calc()", 0))
	assert.Equal(t, 0.0, ParseLength("calc(", 0))
}

func TestCalc_DivisionByZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, ParseLength("calc(10mm / 0)", 0))
}
