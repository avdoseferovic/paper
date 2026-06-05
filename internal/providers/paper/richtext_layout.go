package paper

import "github.com/avdoseferovic/paper/pkg/props"

type richTextMeasureFunc func(run resolvedRun, text string) (translated string, width float64)

type richTextLayoutInput struct {
	prop       *props.RichText
	width      float64
	whiteSpace string
	measure    richTextMeasureFunc
}

func layoutRichTextTokens(runs []resolvedRun, input richTextLayoutInput) ([]rtToken, map[int]float64) {
	tokens := tokeniseRuns(runs, input.whiteSpace)

	for i := range tokens {
		if tokens[i].isBreak {
			continue
		}
		r := runs[tokens[i].runIdx]
		translated, width := input.measure(r, tokens[i].text)
		tokens[i].translated = translated
		tokens[i].width = width
		if r.LetterSpacing > 0 {
			runeCount := len([]rune(translated))
			if runeCount > 1 {
				tokens[i].width += float64(runeCount-1) * r.LetterSpacing
			}
		}
	}

	firstLineIndent := 0.0
	if input.prop != nil {
		firstLineIndent = input.prop.FirstLineIndent
	}
	lineY := 0
	curX := firstXForLine(lineY, firstLineIndent)
	noWrap := input.whiteSpace == "nowrap" || input.whiteSpace == richTextWhiteSpacePre
	for i := range tokens {
		t := &tokens[i]
		if t.isBreak {
			lineY++
			curX = firstXForLine(lineY, firstLineIndent)
			continue
		}

		lineStart := firstXForLine(lineY, firstLineIndent)
		if t.skipAtLineStart && curX == lineStart {
			t.skip = true
			continue
		}
		if !noWrap && curX > lineStart && curX+t.width > input.width {
			lineY++
			curX = firstXForLine(lineY, firstLineIndent)
			if t.skipAtLineStart {
				t.skip = true
				continue
			}
		}
		t.x = curX
		curX += t.width
		t.lineY = lineY
	}

	return tokens, lineWidths(tokens)
}

func lineWidths(tokens []rtToken) map[int]float64 {
	lineWidths := make(map[int]float64)
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		if right := t.x + t.width; right > lineWidths[t.lineY] {
			lineWidths[t.lineY] = right
		}
	}
	return lineWidths
}
