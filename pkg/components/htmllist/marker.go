package htmllist

import (
	"strconv"
	"strings"
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
		return toAlpha(idx, false) + "."
	case UpperAlpha:
		return toAlpha(idx, true) + "."
	case LowerRoman:
		return toRoman(idx+1, false) + "."
	case UpperRoman:
		return toRoman(idx+1, true) + "."
	default: // Unknown styles render as bullets.
		return "•"
	}
}

// toAlpha converts a 0-based index to a lowercase or uppercase letter sequence (a, b, ... z, aa, ab, ...).
func toAlpha(idx int, upper bool) string {
	var b strings.Builder
	n := idx + 1
	for n > 0 {
		n--
		r := rune('a' + n%26)
		if upper {
			r = rune('A' + n%26)
		}
		b.WriteRune(r)
		n /= 26
	}
	// Reverse.
	s := []rune(b.String())
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return string(s)
}

var romanNumerals = []struct {
	val  int
	sym  string
	lsym string
}{
	{1000, "M", "m"},
	{900, "CM", "cm"},
	{500, "D", "d"},
	{400, "CD", "cd"},
	{100, "C", "c"},
	{90, "XC", "xc"},
	{50, "L", "l"},
	{40, "XL", "xl"},
	{10, "X", "x"},
	{9, "IX", "ix"},
	{5, "V", "v"},
	{4, "IV", "iv"},
	{1, "I", "i"},
}

// toRoman converts a positive integer to a Roman numeral string.
func toRoman(n int, upper bool) string {
	var b strings.Builder
	for _, r := range romanNumerals {
		for n >= r.val {
			if upper {
				b.WriteString(r.sym)
			} else {
				b.WriteString(r.lsym)
			}
			n -= r.val
		}
	}
	return b.String()
}
