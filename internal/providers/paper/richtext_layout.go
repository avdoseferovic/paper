package paper

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/props"
)

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
		if tokens[i].isImage(r) {
			tokens[i].width = r.Image.Width
			continue
		}
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
			t.lineY = lineY
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

	lineWidths := lineWidths(tokens)
	if input.prop != nil && input.prop.Align == consts.AlignJustify {
		justifyRichTextLines(tokens, lineWidths, input.width)
	}
	return tokens, lineWidths
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

func justifyRichTextLines(tokens []rtToken, lineWidths map[int]float64, targetWidth float64) {
	if targetWidth <= 0 {
		return
	}
	lastLine := lastRenderableLine(tokens)
	forcedBreaks := forcedBreakLines(tokens)
	for lineY, lineWidth := range lineWidths {
		if lineY == lastLine || forcedBreaks[lineY] {
			continue
		}
		slack := targetWidth - lineWidth
		if slack <= 0 {
			continue
		}
		spaceCount := justifySpaceCount(tokens, lineY)
		if spaceCount == 0 {
			continue
		}
		expandRichTextLine(tokens, lineY, slack/float64(spaceCount))
		lineWidths[lineY] = targetWidth
	}
}

func lastRenderableLine(tokens []rtToken) int {
	last := 0
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		if t.lineY > last {
			last = t.lineY
		}
	}
	return last
}

func forcedBreakLines(tokens []rtToken) map[int]bool {
	lines := make(map[int]bool)
	for _, t := range tokens {
		if t.isBreak {
			lines[t.lineY] = true
		}
	}
	return lines
}

func justifySpaceCount(tokens []rtToken, lineY int) int {
	count := 0
	for _, t := range tokens {
		if t.lineY == lineY && isJustifiableSpace(t) {
			count++
		}
	}
	return count
}

func expandRichTextLine(tokens []rtToken, lineY int, extraPerSpace float64) {
	offset := 0.0
	for i := range tokens {
		if tokens[i].lineY != lineY || tokens[i].isBreak || tokens[i].skip {
			continue
		}
		tokens[i].x += offset
		if isJustifiableSpace(tokens[i]) {
			tokens[i].width += extraPerSpace
			offset += extraPerSpace
		}
	}
}

func isJustifiableSpace(t rtToken) bool {
	return !t.isBreak && !t.skip && t.text == " "
}
