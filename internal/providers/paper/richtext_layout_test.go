package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestLayoutRichTextTokensWrapsAndPreservesOrder(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{Text: "alpha beta gamma"}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: align.Left},
		width:      9,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 5)
	assert.Equal(t, "alpha", tokens[0].text)
	assert.Equal(t, 0, tokens[0].lineY)
	assert.Equal(t, " ", tokens[1].text)
	assert.Equal(t, 0, tokens[1].lineY)
	assert.Equal(t, "beta", tokens[2].text)
	assert.Equal(t, 1, tokens[2].lineY)
	assert.Equal(t, "gamma", tokens[4].text)
	assert.Equal(t, 2, tokens[4].lineY)
	assert.Equal(t, 6.0, lineWidths[0])
	assert.Equal(t, 5.0, lineWidths[1])
	assert.Equal(t, 5.0, lineWidths[2])
}

func TestLayoutRichTextTokensAddsLetterSpacingToMeasuredWidth(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{
		Text:          "abc",
		LetterSpacing: 0.5,
	}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: align.Left},
		width:      10,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 1)
	assert.Equal(t, 4.0, tokens[0].width)
	assert.Equal(t, 4.0, lineWidths[0])
}

func TestLayoutRichTextTokensJustifiesWrappedLines(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{Text: "aa bb cc"}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: align.Justify},
		width:      5.5,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 5)
	assert.Equal(t, "aa", tokens[0].text)
	assert.Equal(t, 0.0, tokens[0].x)
	assert.Equal(t, " ", tokens[1].text)
	assert.Equal(t, 1.5, tokens[1].width)
	assert.Equal(t, "bb", tokens[2].text)
	assert.Equal(t, 3.5, tokens[2].x)
	assert.True(t, tokens[3].skip, "space before wrapped line should not be rendered")
	assert.Equal(t, "cc", tokens[4].text)
	assert.Equal(t, 1, tokens[4].lineY)
	assert.Equal(t, 0.0, tokens[4].x, "last line remains left aligned")
	assert.Equal(t, 5.5, lineWidths[0])
	assert.Equal(t, 2.0, lineWidths[1])
}

func TestLayoutRichTextTokensTreatsOnlyEmptyImageTokenAsImage(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "A "}},
		{RichRun: props.RichRun{Image: &props.RichImage{Width: 4, Height: 3}}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: align.Left},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 3)
	assert.Equal(t, "A", tokens[0].text)
	assert.Equal(t, " ", tokens[1].text)
	assert.False(t, tokens[1].isImage(runs[tokens[1].runIdx]))
	assert.True(t, tokens[2].isImage(runs[tokens[2].runIdx]))
	assert.Equal(t, 1.0, tokens[1].width)
	assert.Equal(t, 4.0, tokens[2].width)
	assert.Equal(t, 6.0, lineWidths[0])
}
