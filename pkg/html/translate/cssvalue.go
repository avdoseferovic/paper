// Shared tokenizers for CSS values whose grammar nests inside parentheses
// (transform functions, grid templates, repeat()).

package translate

import (
	"strings"
	"unicode"
)

// splitTopLevelCommas splits value on commas that are not inside parentheses.
// Each part is trimmed; empty parts are preserved so positional arguments keep
// their index.
func splitTopLevelCommas(value string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(value[start:i]))
				start = i + len(string(r))
			}
		}
	}
	return append(out, strings.TrimSpace(value[start:]))
}

// splitTopLevelFields splits value on whitespace that is not inside
// parentheses, dropping empty tokens.
func splitTopLevelFields(value string) []string {
	var out []string
	var b strings.Builder
	depth := 0
	flush := func() {
		if token := strings.TrimSpace(b.String()); token != "" {
			out = append(out, token)
		}
		b.Reset()
	}
	for _, r := range value {
		switch {
		case r == '(':
			depth++
			b.WriteRune(r)
		case r == ')':
			if depth > 0 {
				depth--
			}
			b.WriteRune(r)
		case depth == 0 && unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}

// matchingCloseParen returns the index of the ')' that closes the '(' at open,
// or -1 when the parentheses are unbalanced.
func matchingCloseParen(value string, open int) int {
	depth := 0
	for i := open; i < len(value); i++ {
		switch value[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
