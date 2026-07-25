// Package listnum formats the CSS numbering systems shared by list markers
// (<ol>) and CSS counters (counter(), counters()).
package listnum

import "strings"

var romanNumerals = []struct {
	value int
	upper string
	lower string
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

// Roman converts a positive ordinal to a Roman numeral, in upper or lower case.
// Values above 3999 keep repeating "M"; n <= 0 has no Roman form and returns "".
func Roman(n int, upper bool) string {
	var b strings.Builder
	for _, numeral := range romanNumerals {
		for n >= numeral.value {
			if upper {
				b.WriteString(numeral.upper)
			} else {
				b.WriteString(numeral.lower)
			}
			n -= numeral.value
		}
	}
	return b.String()
}

// Alpha converts a positive ordinal to a bijective base-26 letter sequence
// (1 → a, 26 → z, 27 → aa), in upper or lower case. n <= 0 returns "".
func Alpha(n int, upper bool) string {
	first := byte('a')
	if upper {
		first = 'A'
	}
	var out []byte
	for n > 0 {
		n--
		out = append(out, first+byte(n%26))
		n /= 26
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
