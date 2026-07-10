package paper

import (
	"fmt"
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts"
)

// unsupportedGlyphsIssueKey is the render issue operation recorded when the
// core-font code page translation replaces glyphs it cannot encode.
const unsupportedGlyphsIssueKey = "text.unsupported-glyphs"

// isCoreFontFamily reports whether family is one of the built-in core font
// families (Arial, Helvetica, Courier, Symbol, ZapfDingbats) which expect
// Latin-1/cp1252 codepoints. Comparison is case-insensitive because callers
// commonly use "Helvetica" while the fontfamily constants are lowercase.
func isCoreFontFamily(family string) bool {
	switch strings.ToLower(family) {
	case consts.FontFamilyArial, consts.FontFamilyHelvetica, consts.FontFamilySymbol,
		consts.FontFamilyZapBats, consts.FontFamilyCourier:
		return true
	default:
		return false
	}
}

// translateUnicode applies the gofpdf Unicode translator for built-in font
// families. For custom (UTF-8) fonts text passes through.
func (s *Text) translateUnicode(text, family string) string {
	if isCoreFontFamily(family) {
		return s.translateDefaultCodePage(text)
	}
	return text
}

func (s *Text) translateDefaultCodePage(txt string) string {
	if s.defaultCodeTranslator == nil {
		s.defaultCodeTranslator = s.pdf.UnicodeTranslatorFromDescriptor("")
	}
	return s.defaultCodeTranslator(txt)
}

// SetRenderIssueSink registers fn to receive render fallback issues, such as
// glyphs that cannot be encoded in the core-font code page.
func (s *Text) SetRenderIssueSink(fn func(operation, message string)) {
	s.issueSink = fn
}

// reportUnsupportedGlyphs records one render issue when the code page
// translation replaced unmappable runes. original is the pre-translation
// UTF-8 text; translated is the code-page byte string produced from it.
func (s *Text) reportUnsupportedGlyphs(original, translated string) {
	if s.issueSink == nil {
		return
	}
	s.reportMissingRunes(unsupportedGlyphs(original, translated))
}

// reportMissingRunes records a single deduped render issue for the given
// unmappable runes. No-op when the list is empty or no sink is wired.
func (s *Text) reportMissingRunes(missing []rune) {
	if s.issueSink == nil || len(missing) == 0 {
		return
	}
	s.issueSink(unsupportedGlyphsIssueKey, fmt.Sprintf(
		"characters %q are not supported by the built-in font code page (cp1252) and were replaced",
		string(missing),
	))
}

// unsupportedGlyphs returns the distinct runes of original that the code page
// translation replaced with the '.' fallback byte. The translator emits
// exactly one byte per input rune, so original rune i maps to translated byte
// i; mapped code page bytes are always >= 0x80 for non-ASCII runes, which
// makes '.' an unambiguous fallback marker there.
func unsupportedGlyphs(original, translated string) []rune {
	if original == translated {
		// Pure ASCII or no translation applied — nothing was replaced.
		return nil
	}
	runes := []rune(original)
	if len(runes) != len(translated) {
		// Not a per-rune code page translation (translator inactive).
		return nil
	}
	var missing []rune
	seen := make(map[rune]bool)
	for i, r := range runes {
		if r >= 0x80 && translated[i] == '.' && !seen[r] {
			seen[r] = true
			missing = append(missing, r)
		}
	}
	return missing
}
