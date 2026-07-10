package paper

import (
	"strings"
	"unicode"
)

// richTextHyphenator implements the Liang-Knuth hyphenation algorithm used by
// TeX. It is intentionally provider-internal for now: the public API exposes
// CSS hyphen controls through HTML/RichText properties rather than a standalone
// hyphenation package.
type richTextHyphenator struct {
	patterns map[string][]int
}

type Hyphenator = richTextHyphenator

func NewHyphenator(patterns []string) *Hyphenator {
	return newRichTextHyphenator(patterns)
}

func newRichTextHyphenator(patterns []string) *richTextHyphenator {
	h := &richTextHyphenator{
		patterns: make(map[string][]int, len(patterns)),
	}
	for _, p := range patterns {
		letters, values := parseHyphenationPattern(p)
		h.patterns[letters] = values
	}
	return h
}

func parseHyphenationPattern(pattern string) (string, []int) {
	var letters []rune
	var values []int
	for _, r := range pattern {
		if r >= '0' && r <= '9' {
			for len(values) <= len(letters) {
				values = append(values, 0)
			}
			values[len(letters)] = int(r - '0')
			continue
		}
		letters = append(letters, r)
	}
	for len(values) <= len(letters) {
		values = append(values, 0)
	}
	return string(letters), values
}

func (h *richTextHyphenator) Hyphenate(word string) []int {
	runes := []rune(strings.ToLower(word))
	if len(runes) < 4 {
		return nil
	}

	wrapped := make([]rune, 0, len(runes)+2)
	wrapped = append(wrapped, '.')
	wrapped = append(wrapped, runes...)
	wrapped = append(wrapped, '.')

	levels := make([]int, len(wrapped)+1)
	for i := range wrapped {
		for j := i + 1; j <= len(wrapped); j++ {
			sub := string(wrapped[i:j])
			values, ok := h.patterns[sub]
			if !ok {
				continue
			}
			for k, value := range values {
				pos := i + k
				if pos < len(levels) && value > levels[pos] {
					levels[pos] = value
				}
			}
		}
	}

	var breaks []int
	for i := 2; i < len(runes); i++ {
		pos := i + 1
		if pos < len(levels) && levels[pos]%2 == 1 && i <= len(runes)-2 {
			breaks = append(breaks, i)
		}
	}
	return breaks
}

func defaultRichTextHyphenator() *richTextHyphenator {
	initDefaultHyphenator()
	return defaultHyphenator
}

func isAlphaHyphenationWord(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
