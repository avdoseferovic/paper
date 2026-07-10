package css

import (
	"strconv"
	"strings"
)

const (
	mmPerPx = 0.264583
	mmPerPt = 0.352778
	mmPerCm = 10.0
	mmPerIn = 25.4
	// defaultRemMM approximates 1rem (16px root font size) when a value uses
	// rem with no root context available.
	defaultRemMM = 16 * mmPerPx
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

	remFactor := parentFontSize
	if remFactor == 0 {
		remFactor = defaultRemMM
	}
	// Longer suffixes must be checked before their suffixes ("rem" before
	// "em", "mm"/"cm" before "m"): "1rem" also ends in "em" and must not be
	// rejected because "1r" fails to parse.
	units := []struct {
		suffix string
		factor float64
	}{
		{"rem", remFactor}, // approximate: rem resolves like em against the parent
		{"mm", 1},
		{"cm", mmPerCm},
		{"pt", mmPerPt},
		{"px", mmPerPx},
		{"in", mmPerIn},
		{"em", parentFontSize},
	}

	for _, u := range units {
		numStr, ok := strings.CutSuffix(value, u.suffix)
		if !ok {
			continue
		}
		num, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
		if err != nil {
			continue
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
