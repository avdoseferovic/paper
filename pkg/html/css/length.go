package css

import (
	"strconv"
	"strings"
)

const (
	mmPerPx = 0.264583
	mmPerPt = 0.352778
	mmPerCm = 10.0
)

// ParsePercentage parses a CSS percentage value (e.g. "25%") and returns
// the fractional equivalent (0.25 for "25%"). Returns (0, false) if val is
// not a valid percentage string.
func ParsePercentage(val string) (float64, bool) {
	val = strings.TrimSpace(val)
	numStr, ok := strings.CutSuffix(val, "%")
	if !ok || numStr == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, false
	}
	return v / 100.0, true
}

// ParseLength converts a CSS length string to mm.
// parentFontSize (mm) is used to resolve em units.
// Returns 0 for unparseable values.
//
// For calc() expressions, dispatches to the calc evaluator (without a width
// context; use ParseLengthCtx for % resolution inside calc()).
func ParseLength(value string, parentFontSize float64) float64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return 0
	}
	if strings.HasPrefix(value, "calc(") && strings.HasSuffix(value, ")") {
		expr := strings.TrimSpace(value[5 : len(value)-1])
		v, _ := evalCalc(expr, parentFontSize, 0)
		return v
	}

	units := []struct {
		suffix string
		factor float64
	}{
		{"mm", 1},
		{"cm", mmPerCm},
		{"pt", mmPerPt},
		{"px", mmPerPx},
		{"em", parentFontSize},
		{"rem", parentFontSize}, // approximate: treat rem same as em
	}

	for _, u := range units {
		numStr, ok := strings.CutSuffix(value, u.suffix)
		if !ok {
			continue
		}
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0
		}
		return num * u.factor
	}

	// Unitless number — return as-is (e.g. line-height multiplier).
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return v
}
