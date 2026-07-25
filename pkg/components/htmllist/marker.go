package htmllist

import (
	"strconv"

	"github.com/avdoseferovic/paper/internal/listnum"
)

// FormatMarker returns the formatted marker string for a given style and 0-based index.
func FormatMarker(style StyleType, idx int) string {
	switch style {
	case None:
		return ""
	case Bullet:
		return "•"
	case Decimal:
		return strconv.Itoa(idx+1) + "."
	case DecimalCircle:
		// Circle markers render the bare number (no trailing period) centred in a disc.
		return strconv.Itoa(idx + 1)
	case LowerAlpha:
		if idx < 0 {
			return strconv.Itoa(idx+1) + "." // alphabetic counters have no zero/negative form
		}
		return listnum.Alpha(idx+1, false) + "."
	case UpperAlpha:
		if idx < 0 {
			return strconv.Itoa(idx+1) + "."
		}
		return listnum.Alpha(idx+1, true) + "."
	case LowerRoman:
		if idx < 0 {
			return strconv.Itoa(idx+1) + "." // roman numerals have no zero/negative form
		}
		return listnum.Roman(idx+1, false) + "."
	case UpperRoman:
		if idx < 0 {
			return strconv.Itoa(idx+1) + "."
		}
		return listnum.Roman(idx+1, true) + "."
	default: // Unknown styles render as bullets.
		return "•"
	}
}
