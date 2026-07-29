package paper

import (
	"cmp"

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
		if tokens[i].isFixedInlineBox(r) {
			tokens[i].width = r.InlineBoxWidth
			continue
		}
		translated, width := input.measure(r, tokens[i].text)
		tokens[i].translated = translated
		tokens[i].width = width
		if r.LetterSpacing > 0 {
			// CSS letter-spacing adds spacing after EVERY glyph, including the
			// last one of a token and space glyphs between words. Reserving the
			// trailing spacing keeps word gaps (space glyph + spacing) from
			// collapsing so letter-spaced text like uppercase kickers doesn't
			// merge into one run. Matches the per-glyph advance in renderTokenText.
			if runeCount := len([]rune(translated)); runeCount > 0 {
				tokens[i].width += float64(runeCount) * r.LetterSpacing
			}
		}
	}

	firstLineIndent := 0.0
	if input.prop != nil {
		firstLineIndent = input.prop.FirstLineIndent
	}
	boxStart, boxEnd := inlineBoxGroupBoundaries(tokens, runs)
	runStart, runEnd := runTokenBoundaries(tokens)
	lineY := 0
	curX := firstXForLine(lineY, firstLineIndent)
	noWrap := input.whiteSpace == richTextWhiteSpaceNoWrap || input.whiteSpace == richTextWhiteSpacePre
	for i := 0; i < len(tokens); {
		t := &tokens[i]
		if t.isBreak {
			t.lineY = lineY
			lineY++
			curX = firstXForLine(lineY, firstLineIndent)
			i++
			continue
		}

		lineStart := firstXForLine(lineY, firstLineIndent)
		if t.skipAtLineStart && curX == lineStart {
			t.skip = true
			i++
			continue
		}
		groupEnd := wrapGroupEnd(tokens, runs, i)
		groupAdvance := tokenGroupAdvance(tokens, runs, boxStart, boxEnd, runStart, runEnd, i, groupEnd)
		if !noWrap && curX > lineStart && curX+groupAdvance > input.width {
			lineY++
			curX = firstXForLine(lineY, firstLineIndent)
			if t.skipAtLineStart {
				t.skip = true
				i++
				continue
			}
		}
		for j := i; j < groupEnd; j++ {
			run := runs[tokens[j].runIdx]
			before, after := tokenInlineSpacing(run, boxStart[j], boxEnd[j], runStart[j], runEnd[j])
			curX += before
			tokens[j].x = curX
			curX += tokens[j].width
			tokens[j].right = curX + after
			curX = tokens[j].right
			tokens[j].lineY = lineY
		}
		i = groupEnd
	}

	hangTrailingWrapSpaces(tokens)

	lineWidths := lineWidths(tokens)
	if input.prop != nil && input.prop.Align == consts.AlignJustify {
		justifyRichTextLines(tokens, lineWidths, input.width)
	}
	return tokens, lineWidths
}

// hangTrailingWrapSpaces marks collapse-mode space tokens that end a line
// (i.e. the following word wrapped or a forced break follows) as skipped.
// Browsers hang spaces at a wrap point: they contribute neither to the
// measured line width used for right/center alignment nor to justification.
// Only collapse-mode spaces hang — preserved whitespace (white-space: pre /
// pre-wrap) is tokenised inside word tokens and is never affected; the
// skipAtLineStart flag identifies collapse-mode spaces.
func hangTrailingWrapSpaces(tokens []rtToken) {
	lastOnLine := make(map[int]int)
	for i := range tokens {
		if tokens[i].isBreak || tokens[i].skip {
			continue
		}
		lastOnLine[tokens[i].lineY] = i
	}
	for _, i := range lastOnLine {
		if tokens[i].text == " " && tokens[i].skipAtLineStart {
			tokens[i].skip = true
		}
	}
}

func tokenGroupAdvance(tokens []rtToken, runs []resolvedRun, boxStart, boxEnd, runStart, runEnd []bool, start, end int) float64 {
	total := 0.0
	for i := start; i < end; i++ {
		run := runs[tokens[i].runIdx]
		before, after := tokenInlineSpacing(run, boxStart[i], boxEnd[i], runStart[i], runEnd[i])
		total += before + tokens[i].width + after
	}
	return total
}

// wrapGroupEnd returns the exclusive end of the token group that must wrap as
// one unit starting at start: the token's nowrap group, extended across glued
// tokens (words adjacent with no whitespace, e.g. `<strong>Stress</strong>?`)
// and their nowrap groups.
func wrapGroupEnd(tokens []rtToken, runs []resolvedRun, start int) int {
	end := nowrapTokenGroupEnd(tokens, runs, start)
	for end < len(tokens) && !tokens[end].isBreak && tokens[end].gluePrev {
		end = nowrapTokenGroupEnd(tokens, runs, end)
	}
	return end
}

func nowrapTokenGroupEnd(tokens []rtToken, runs []resolvedRun, start int) int {
	key := nowrapTokenGroupKey(tokens[start], runs)
	if key == 0 {
		return start + 1
	}
	end := start + 1
	for end < len(tokens) && !tokens[end].isBreak && nowrapTokenGroupKey(tokens[end], runs) == key {
		end++
	}
	return end
}

func nowrapTokenGroupKey(t rtToken, runs []resolvedRun) int {
	if t.runIdx < 0 || t.runIdx >= len(runs) {
		return 0
	}
	run := runs[t.runIdx]
	if normalizeRichTextWhiteSpace(run.WhiteSpace) != richTextWhiteSpaceNoWrap {
		return 0
	}
	if run.InlineBoxID > 0 {
		return run.InlineBoxID
	}
	return -(t.runIdx + 1)
}

func tokenInlineSpacing(run resolvedRun, boxStart, boxEnd, runStart, runEnd bool) (float64, float64) {
	before := 0.0
	after := 0.0
	// Reserve a background run's horizontal padding in the flow so adjacent
	// pills/badges don't overlap — their backgrounds are drawn inflated by the
	// same padding in renderRunBackgrounds. Padding belongs to the shared inline
	// box, while margins belong to the individual run. This matters for nested
	// generated content such as `.choice::before { margin-right: ... }` inside a
	// larger pill: the marker's margin must create an internal gap without
	// repeating the parent pill padding.
	if run.hasInlineBox() {
		if boxStart {
			before += run.inlineBorderWidth() + run.inlinePadLeft()
		}
		if boxEnd {
			after += run.inlinePadRight() + run.inlineBorderWidth()
		}
	}
	if runStart {
		before += run.InlineMarginLeft
	}
	if runEnd {
		after += run.InlineMarginRight
	}
	return before, after
}

// inlineBoxGroupBoundaries marks whether each token is the first/last
// renderable token of a contiguous same-inline-box group. Used to reserve
// background padding only at the outer edges of a shared inline box.
func inlineBoxGroupBoundaries(tokens []rtToken, runs []resolvedRun) ([]bool, []bool) {
	start := make([]bool, len(tokens))
	end := make([]bool, len(tokens))
	prev := -1
	for i := range tokens {
		if tokens[i].isBreak || tokens[i].skip {
			continue
		}
		if prev < 0 || richTextInlineBoxKey(tokens[prev], runs) != richTextInlineBoxKey(tokens[i], runs) {
			start[i] = true
			if prev >= 0 {
				end[prev] = true
			}
		}
		prev = i
	}
	if prev >= 0 {
		end[prev] = true
	}
	return start, end
}

// runTokenBoundaries marks whether each token is the first/last renderable
// token of a contiguous same-run group. Used for run-local margins, including
// generated-content pseudo-elements nested inside a shared parent inline box.
func runTokenBoundaries(tokens []rtToken) ([]bool, []bool) {
	start := make([]bool, len(tokens))
	end := make([]bool, len(tokens))
	prev := -1
	for i := range tokens {
		if tokens[i].isBreak || tokens[i].skip {
			continue
		}
		if prev < 0 || tokens[prev].runIdx != tokens[i].runIdx {
			start[i] = true
			if prev >= 0 {
				end[prev] = true
			}
		}
		prev = i
	}
	if prev >= 0 {
		end[prev] = true
	}
	return start, end
}

func richTextInlineBoxKey(t rtToken, runs []resolvedRun) int {
	if t.runIdx < 0 || t.runIdx >= len(runs) {
		return 0
	}
	if id := runs[t.runIdx].InlineBoxID; id > 0 {
		return id
	}
	return -(t.runIdx + 1)
}

func lineWidths(tokens []rtToken) map[int]float64 {
	lineWidths := make(map[int]float64)
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		right := cmp.Or(t.right, t.x+t.width)
		lineWidths[t.lineY] = max(lineWidths[t.lineY], right)
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
		last = max(last, t.lineY)
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
