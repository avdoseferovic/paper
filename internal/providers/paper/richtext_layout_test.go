package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestLayoutRichTextTokensWrapsAndPreservesOrder(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{Text: "alpha beta gamma"}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
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

func TestLayoutRichTextTokensPreservesNonBreakingSpaces(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "a"}},
		{RichRun: props.RichRun{Text: "\u00a0\u00a0 "}},
		{RichRun: props.RichRun{Text: "b"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len([]rune(text)))
		},
	})

	require.Len(t, tokens, 4)
	assert.Equal(t, "a", tokens[0].text)
	assert.Equal(t, "\u00a0\u00a0", tokens[1].text)
	assert.Equal(t, " ", tokens[2].text)
	assert.Equal(t, "b", tokens[3].text)
	assert.Equal(t, 5.0, lineWidths[0])
}

func TestLayoutRichTextTokensGluesAdjacentRunsWithoutWhitespace(t *testing.T) {
	t.Parallel()

	// `unter <strong>Zeitdruck oder Stress</strong>?` — no whitespace between
	// the bold run's last word and the trailing "?": CSS defines no break
	// opportunity there, so "Stress?" must wrap as one unit instead of
	// orphaning the "?" on its own line.
	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "unter "}},
		{RichRun: props.RichRun{Text: "Zeitdruck oder Stress"}},
		{RichRun: props.RichRun{Text: "?"}},
	}
	// Width fits "unter Zeitdruck oder Stress" (27) exactly, but not the "?"
	// glued behind it — the whole "Stress?" unit must wrap, not just the "?".
	tokens, _ := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      27,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len([]rune(text)))
		},
	})

	byText := map[string]rtToken{}
	for _, tok := range tokens {
		byText[tok.text] = tok
	}
	require.Equal(t, byText["Stress"].lineY, byText["?"].lineY,
		"'?' has no break opportunity before it and must stay glued to 'Stress'")
	assert.Equal(t, byText["Stress"].right, byText["?"].x,
		"'?' must render immediately after 'Stress'")
}

func TestLayoutRichTextTokensSpaceStillBreaksBetweenRuns(t *testing.T) {
	t.Parallel()

	// A space before the second run keeps the break opportunity.
	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "alpha "}},
		{RichRun: props.RichRun{Text: "beta"}},
	}
	tokens, _ := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      6,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len([]rune(text)))
		},
	})
	byText := map[string]rtToken{}
	for _, tok := range tokens {
		byText[tok.text] = tok
	}
	assert.Equal(t, 0, byText["alpha"].lineY)
	assert.Equal(t, 1, byText["beta"].lineY)
}

func TestLayoutRichTextTokensKeepsNowrapInlineBoxTogether(t *testing.T) {
	t.Parallel()

	boxColor := &props.Color{Red: 1}
	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "lead"}},
		{RichRun: props.RichRun{
			Image:             &props.RichImage{Width: 2, Height: 2},
			Background:        boxColor,
			BgPadLeft:         1,
			BgPadRight:        1,
			InlineMarginRight: 1,
			InlineBoxID:       7,
			WhiteSpace:        "nowrap",
		}},
		{RichRun: props.RichRun{
			Text:        "two words",
			Background:  boxColor,
			BgPadLeft:   1,
			BgPadRight:  1,
			InlineBoxID: 7,
			WhiteSpace:  "nowrap",
		}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      12,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 3)
	assert.Equal(t, "lead", tokens[0].text)
	assert.Equal(t, 0, tokens[0].lineY)
	assert.True(t, tokens[1].isImage(runs[tokens[1].runIdx]))
	assert.Equal(t, "two words", tokens[2].text)
	assert.Equal(t, 1, tokens[1].lineY, "nowrap inline box should start on the next line as one unit")
	assert.Equal(t, 1, tokens[2].lineY, "text should stay with the generated icon")
	assert.Equal(t, 4.0, lineWidths[0])
	assert.Equal(t, 14.0, lineWidths[1])
}

func TestLayoutRichTextTokensAddsLetterSpacingToMeasuredWidth(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{
		Text:          "abc",
		LetterSpacing: 0.5,
	}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      10,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	// "abc" = 3 measured units + trailing letter-spacing after each of the 3
	// glyphs (3 * 0.5). The render path advances by letter-spacing after every
	// glyph (incl. the last), so layout reserves the same to avoid overlap and
	// to keep word gaps from collapsing.
	require.Len(t, tokens, 1)
	assert.Equal(t, 4.5, tokens[0].width)
	assert.Equal(t, 4.5, lineWidths[0])
}

func TestLayoutRichTextTokensReservesFixedInlineBoxWidth(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{InlineBoxWidth: 4, InlineBoxHeight: 4, BorderColor: &props.Color{Red: 1}, BorderWidth: 0.2}},
		{RichRun: props.RichRun{Text: "x"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 2)
	assert.Equal(t, 4.0, tokens[0].width)
	assert.Equal(t, 4.4, tokens[1].x)
	assert.Equal(t, 5.4, lineWidths[0])
}

func TestLayoutRichTextTokensReservesInlineMarginAfterBox(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "a", Background: &props.Color{Red: 1}, BgPadX: 1, InlineMarginRight: 2}},
		{RichRun: props.RichRun{Text: "b"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 2)
	assert.Equal(t, 1.0, tokens[0].x)
	assert.Equal(t, 5.0, tokens[1].x)
	assert.Equal(t, 6.0, lineWidths[0])
}

func TestLayoutRichTextTokensReservesRunMarginInsideSharedInlineBox(t *testing.T) {
	t.Parallel()

	boxColor := &props.Color{Red: 1}
	runs := []resolvedRun{
		{RichRun: props.RichRun{
			Text:              "a",
			Background:        boxColor,
			BgPadX:            1,
			InlineBoxID:       7,
			InlineMarginRight: 2,
		}},
		{RichRun: props.RichRun{
			Text:        "b",
			Background:  boxColor,
			BgPadX:      1,
			InlineBoxID: 7,
		}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 2)
	assert.Equal(t, 1.0, tokens[0].x, "shared box left padding applies once")
	assert.Equal(t, 4.0, tokens[1].x, "run margin creates an internal gap")
	assert.Equal(t, 6.0, lineWidths[0], "shared box right padding still applies once")
}

func TestLayoutRichTextTokensReservesInlineMarginBeforeBox(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "a", Background: &props.Color{Red: 1}, BgPadX: 1, InlineMarginLeft: 2}},
		{RichRun: props.RichRun{Text: "b"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 2)
	assert.Equal(t, 3.0, tokens[0].x)
	assert.Equal(t, 5.0, tokens[1].x)
	assert.Equal(t, 6.0, lineWidths[0])
}

func TestLayoutRichTextTokensReservesAsymmetricInlinePadding(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{
		{RichRun: props.RichRun{Text: "a", Background: &props.Color{Red: 1}, BgPadLeft: 2, BgPadRight: 3}},
		{RichRun: props.RichRun{Text: "b"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 2)
	assert.Equal(t, 2.0, tokens[0].x)
	assert.Equal(t, 6.0, tokens[1].x)
	assert.Equal(t, 7.0, lineWidths[0])
}

func TestLayoutRichTextTokensJustifiesWrappedLines(t *testing.T) {
	t.Parallel()

	runs := []resolvedRun{{RichRun: props.RichRun{Text: "aa bb cc"}}}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignJustify},
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
		prop:       &props.RichText{Align: consts.AlignLeft},
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

func TestLayoutRichTextTokensIncludesImageInSharedInlineBox(t *testing.T) {
	t.Parallel()

	boxColor := &props.Color{Red: 1}
	runs := []resolvedRun{
		{RichRun: props.RichRun{
			Image:       &props.RichImage{Width: 2, Height: 2},
			Background:  boxColor,
			BgPadX:      1,
			InlineBoxID: 7,
		}},
		{RichRun: props.RichRun{
			Text:        "ok",
			Background:  boxColor,
			BgPadX:      1,
			InlineBoxID: 7,
		}},
		{RichRun: props.RichRun{Text: "z"}},
	}
	tokens, lineWidths := layoutRichTextTokens(runs, richTextLayoutInput{
		prop:       &props.RichText{Align: consts.AlignLeft},
		width:      20,
		whiteSpace: "normal",
		measure: func(_ resolvedRun, text string) (string, float64) {
			return text, float64(len(text))
		},
	})

	require.Len(t, tokens, 3)
	assert.True(t, tokens[0].isImage(runs[tokens[0].runIdx]))
	assert.Equal(t, 1.0, tokens[0].x, "shared box left padding should apply before the image")
	assert.Equal(t, 3.0, tokens[1].x)
	assert.Equal(t, 6.0, tokens[2].x, "shared box right padding should apply after text, not before it")
	assert.Equal(t, 7.0, lineWidths[0])
}
