package css

import (
	"strings"
	"unicode"
)

// ApplyTextTransform returns text with the CSS text-transform applied.
// Supported values: "none", "uppercase", "lowercase", "capitalize". Unknown
// values pass through unchanged. Capitalize is English-centric (per the CSS
// spec it's locale-dependent; this implementation uppercases the first rune
// of each whitespace-delimited word).
func ApplyTextTransform(text, transform string) string {
	switch strings.ToLower(strings.TrimSpace(transform)) {
	case "uppercase":
		return strings.ToUpper(text)
	case "lowercase":
		return strings.ToLower(text)
	case "capitalize":
		return capitalizeWords(text)
	default:
		return text
	}
}

func capitalizeWords(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
			b.WriteRune(r)
			continue
		}
		if !inWord {
			b.WriteRune(unicode.ToUpper(r))
			inWord = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
