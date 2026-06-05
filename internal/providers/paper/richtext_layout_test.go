package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/v2/pkg/consts/align"
	"github.com/avdoseferovic/paper/v2/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
