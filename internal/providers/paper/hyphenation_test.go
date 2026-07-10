package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

func TestNewHyphenator_HyphenatesKnownWord(t *testing.T) {
	t.Parallel()

	h := NewHyphenator([]string{"hy3ph", "he2n", "hena4", "hen5at"})

	breaks := h.Hyphenate("hyphenation")
	assert.NotEmpty(t, breaks)
	for _, b := range breaks {
		assert.True(t, b > 0 && b < len("hyphenation"))
	}
}

func TestHyphenate_ShortWordsAndNonAlpha(t *testing.T) {
	t.Parallel()

	h := defaultRichTextHyphenator()
	require.NotNil(t, h)

	assert.Nil(t, h.Hyphenate("cat"), "words under 4 runes never hyphenate")
	assert.NotEmpty(t, h.Hyphenate("hyphenation"))
}

func TestIsAlphaHyphenationWord(t *testing.T) {
	t.Parallel()

	assert.True(t, isAlphaHyphenationWord("hyphenation"))
	assert.False(t, isAlphaHyphenationWord("abc123"))
	assert.False(t, isAlphaHyphenationWord("with-dash"))
}

func TestParseHyphenationPattern(t *testing.T) {
	t.Parallel()

	letters, values := parseHyphenationPattern("hy3ph")
	assert.Equal(t, "hyph", letters)
	assert.Equal(t, []int{0, 0, 3, 0, 0}, values)
}
