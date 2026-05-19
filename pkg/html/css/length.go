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

// ParseLength converts a CSS length string to mm.
// parentFontSize (mm) is used to resolve em units.
// Returns 0 for unparseable values.
func ParseLength(value string, parentFontSize float64) float64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return 0
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
