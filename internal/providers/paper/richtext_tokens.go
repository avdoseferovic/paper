package paper

import (
	"strings"
	"unicode"
)

// rtToken is the per-word state used by AddRichText's three-pass layout.
type rtToken struct {
	text            string
	translated      string
	runIdx          int
	width           float64
	x               float64
	right           float64
	lineY           int
	isBreak         bool
	skip            bool
	skipAtLineStart bool
	// gluePrev marks a word that continues the previous word token with no
	// whitespace between them (e.g. `<strong>Stress</strong>?`). CSS defines no
	// break opportunity there, so layout wraps the glued sequence as one unit.
	gluePrev bool
}

func (t rtToken) isImage(run resolvedRun) bool {
	return run.Image != nil && t.text == "" && !t.isBreak
}

func (t rtToken) isFixedInlineBox(run resolvedRun) bool {
	return run.hasFixedInlineBox() && t.text == "" && !t.isBreak
}

// tokeniseRuns splits the resolved run sequence into renderable text spans,
// preserving or collapsing whitespace according to CSS white-space semantics.
func tokeniseRuns(runs []resolvedRun, whiteSpace string) []rtToken {
	var out []rtToken
	pendingCollapsedSpace := false
	pendingCollapsedSpaceRunIdx := -1
	for i, r := range runs {
		if r.ForceBreak {
			out = append(out, rtToken{runIdx: i, isBreak: true})
			pendingCollapsedSpace = false
			pendingCollapsedSpaceRunIdx = -1
			continue
		}
		if r.Image != nil {
			if pendingCollapsedSpace && hasTextOnCurrentLine(out) {
				out = append(out, rtToken{text: " ", runIdx: pendingCollapsedSpaceRunIdx, skipAtLineStart: true})
			}
			pendingCollapsedSpace = false
			pendingCollapsedSpaceRunIdx = -1
			out = append(out, rtToken{runIdx: i})
			continue
		}
		if r.hasFixedInlineBox() && r.Text == "" {
			if pendingCollapsedSpace && hasTextOnCurrentLine(out) {
				out = append(out, rtToken{text: " ", runIdx: pendingCollapsedSpaceRunIdx, skipAtLineStart: true})
			}
			pendingCollapsedSpace = false
			pendingCollapsedSpaceRunIdx = -1
			out = append(out, rtToken{runIdx: i})
			continue
		}
		runWhiteSpace := normalizeRichTextWhiteSpace(r.WhiteSpace)
		switch {
		case runWhiteSpace == richTextWhiteSpaceNoWrap:
			out, pendingCollapsedSpace, pendingCollapsedSpaceRunIdx = appendCollapsedNowrapToken(
				out,
				r.Text,
				i,
				pendingCollapsedSpace,
				pendingCollapsedSpaceRunIdx,
			)
		case whiteSpace == richTextWhiteSpacePre || whiteSpace == richTextWhiteSpacePreWrap:
			out = append(out, tokenisePreservedText(r.Text, i)...)
		case whiteSpace == richTextWhiteSpacePreLine:
			out, pendingCollapsedSpace, pendingCollapsedSpaceRunIdx = appendCollapsedTokens(
				out,
				r.Text,
				i,
				true,
				pendingCollapsedSpace,
				pendingCollapsedSpaceRunIdx,
			)
		default:
			out, pendingCollapsedSpace, pendingCollapsedSpaceRunIdx = appendCollapsedTokens(
				out,
				r.Text,
				i,
				false,
				pendingCollapsedSpace,
				pendingCollapsedSpaceRunIdx,
			)
		}
	}
	return out
}

func appendCollapsedNowrapToken(
	out []rtToken,
	text string,
	runIdx int,
	pendingSpace bool,
	pendingSpaceRunIdx int,
) ([]rtToken, bool, int) {
	before := len(out)
	out, pendingSpace, pendingSpaceRunIdx = appendCollapsedTokens(out, text, runIdx, false, pendingSpace, pendingSpaceRunIdx)
	if len(out) == before {
		return out, pendingSpace, pendingSpaceRunIdx
	}

	merged := make([]rtToken, 0, len(out))
	merged = append(merged, out[:before]...)
	var b strings.Builder
	glue := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		merged = append(merged, rtToken{text: b.String(), runIdx: runIdx, gluePrev: glue})
		b.Reset()
		glue = false
	}
	for _, t := range out[before:] {
		if !t.isBreak && t.text != "" && t.runIdx == runIdx {
			if b.Len() == 0 {
				glue = t.gluePrev
			}
			b.WriteString(t.text)
			continue
		}
		flush()
		merged = append(merged, t)
	}
	flush()
	return merged, pendingSpace, pendingSpaceRunIdx
}

func appendCollapsedTokens(
	out []rtToken,
	text string,
	runIdx int,
	preserveNewlines bool,
	pendingSpace bool,
	pendingSpaceRunIdx int,
) ([]rtToken, bool, int) {
	var b strings.Builder
	flushWord := func() {
		if b.Len() == 0 {
			return
		}
		if pendingSpace && hasTextOnCurrentLine(out) {
			out = append(out, rtToken{text: " ", runIdx: pendingSpaceRunIdx, skipAtLineStart: true})
		}
		glue := !pendingSpace && endsInGlueableWord(out)
		pendingSpace = false
		pendingSpaceRunIdx = -1
		out = append(out, rtToken{text: b.String(), runIdx: runIdx, gluePrev: glue})
		b.Reset()
	}
	for _, r := range text {
		if r == '\n' && preserveNewlines {
			flushWord()
			out = append(out, rtToken{runIdx: runIdx, isBreak: true})
			pendingSpace = false
			pendingSpaceRunIdx = -1
			continue
		}
		if isCollapsibleRichSpace(r) {
			flushWord()
			pendingSpace = true
			pendingSpaceRunIdx = runIdx
			continue
		}
		b.WriteRune(r)
	}
	flushWord()
	return out, pendingSpace, pendingSpaceRunIdx
}

func isCollapsibleRichSpace(r rune) bool {
	return unicode.IsSpace(r) && r != '\u00a0'
}

// endsInGlueableWord reports whether the last token is a word another word can
// glue to. Images and fixed inline boxes are replaced elements \u2014 CSS allows
// breaks around those even without whitespace \u2014 so only text words glue.
func endsInGlueableWord(tokens []rtToken) bool {
	if len(tokens) == 0 {
		return false
	}
	last := tokens[len(tokens)-1]
	return !last.isBreak && last.text != "" && last.text != " "
}

func tokenisePreservedText(text string, runIdx int) []rtToken {
	var out []rtToken
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, rtToken{text: b.String(), runIdx: runIdx})
		b.Reset()
	}
	for _, r := range text {
		if r == '\n' {
			flush()
			out = append(out, rtToken{runIdx: runIdx, isBreak: true})
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

func hasTextOnCurrentLine(tokens []rtToken) bool {
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].isBreak {
			return false
		}
		if tokens[i].text != "" {
			return true
		}
	}
	return false
}
