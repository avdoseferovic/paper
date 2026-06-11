package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestTokenisePreservedText(t *testing.T) {
	t.Parallel()

	t.Run("splits text on newlines keeping break tokens", func(t *testing.T) {
		t.Parallel()
		tokens := tokenisePreservedText("ab\ncd\n", 3)

		require.Len(t, tokens, 4)
		assert.Equal(t, "ab", tokens[0].text)
		assert.Equal(t, 3, tokens[0].runIdx)
		assert.True(t, tokens[1].isBreak)
		assert.Equal(t, "cd", tokens[2].text)
		assert.True(t, tokens[3].isBreak)
	})

	t.Run("leading newline emits a break first", func(t *testing.T) {
		t.Parallel()
		tokens := tokenisePreservedText("\nx", 0)

		require.Len(t, tokens, 2)
		assert.True(t, tokens[0].isBreak)
		assert.Equal(t, "x", tokens[1].text)
	})

	t.Run("empty text yields no tokens", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, tokenisePreservedText("", 0), 0)
	})
}

func TestHasTextOnCurrentLine(t *testing.T) {
	t.Parallel()

	assert.False(t, hasTextOnCurrentLine(nil))
	assert.True(t, hasTextOnCurrentLine([]rtToken{{text: "a"}}))
	assert.False(t, hasTextOnCurrentLine([]rtToken{{text: "a"}, {isBreak: true}}))
	assert.True(t, hasTextOnCurrentLine([]rtToken{{isBreak: true}, {text: "b"}}))
	// Image tokens (empty text) do not count as text.
	assert.False(t, hasTextOnCurrentLine([]rtToken{{text: ""}}))
}

func TestRichTextLineCount(t *testing.T) {
	t.Parallel()

	t.Run("no tokens is one line", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, richTextLineCount(nil))
	})

	t.Run("breaks advance the count and skipped tokens are ignored", func(t *testing.T) {
		t.Parallel()
		tokens := []rtToken{
			{lineY: 0},
			{isBreak: true, lineY: 0},
			{skip: true, lineY: 5},
			{lineY: 1},
		}
		assert.Equal(t, 2, richTextLineCount(tokens))
	})
}

func TestAlignmentOffset(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0.0, alignmentOffset(consts.AlignLeft, 100, 40))
	assert.Equal(t, 30.0, alignmentOffset(consts.AlignCenter, 100, 40))
	assert.Equal(t, 60.0, alignmentOffset(consts.AlignRight, 100, 40))
	assert.Equal(t, 0.0, alignmentOffset(consts.AlignJustify, 100, 40))
	assert.Equal(t, 0.0, alignmentOffset(consts.Align("unknown"), 100, 40))
	// No slack → no offset regardless of alignment.
	assert.Equal(t, 0.0, alignmentOffset(consts.AlignRight, 40, 100))
}

func TestNormalizeRichTextWhiteSpace(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "pre", normalizeRichTextWhiteSpace(" PRE "))
	assert.Equal(t, "nowrap", normalizeRichTextWhiteSpace("nowrap"))
	assert.Equal(t, "pre-wrap", normalizeRichTextWhiteSpace("pre-wrap"))
	assert.Equal(t, "pre-line", normalizeRichTextWhiteSpace("pre-line"))
	assert.Equal(t, "normal", normalizeRichTextWhiteSpace(""))
	assert.Equal(t, "normal", normalizeRichTextWhiteSpace("anything-else"))
}

func TestResolvedRun_StyleWithUnderline(t *testing.T) {
	t.Parallel()

	underlined := resolvedRun{RichRun: props.RichRun{Style: fontstyle.Bold, Underline: true}}
	assert.Equal(t, fontstyle.Type("BU"), underlined.styleWithUnderline())

	plain := resolvedRun{RichRun: props.RichRun{Style: fontstyle.Bold}}
	assert.Equal(t, fontstyle.Bold, plain.styleWithUnderline())
}
