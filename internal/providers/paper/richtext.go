package paper

import (
	"strings"
	"unicode"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

const richTextWhiteSpacePre = "pre"

// MeasureString sets the current font and returns the string width in mm.
// All intermediate state is method-local; no struct mutation occurs beyond the
// font context (which callers may need to restore themselves).
func (s *Text) MeasureString(text string, prop *props.Text) float64 {
	s.font.SetFont(prop.Family, prop.Style, prop.Size)
	translated := s.translateUnicode(text, prop.Family)
	return s.pdf.GetStringWidth(translated)
}

// AddTextAt renders text at an absolute (x, y) position in mm.
// Baseline-positioned, matching gofpdf's Text() convention.
func (s *Text) AddTextAt(x, y float64, text string, prop *props.Text) {
	s.font.SetFont(prop.Family, prop.Style, prop.Size)
	left, top, _, _ := s.pdf.GetMargins()
	translated := s.translateUnicode(text, prop.Family)
	s.pdf.Text(x+left, y+top, translated)
}

// resolvedRun is a RichRun with empty font fields filled in from the surrounding default.
type resolvedRun struct {
	props.RichRun
}

// AddRichText renders a paragraph of mixed inline runs within cell.
// Font state is captured on entry and restored via defer so callers are unaffected.
// All intermediate state (line buffer, cursors) is method-local — safe for concurrent use.
//
//nolint:gocognit,gocyclo,maintidx // Preserved renderer branching; this task only renames the provider backend.
func (s *Text) AddRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) {
	if len(runs) == 0 {
		return
	}

	// Capture and restore font state.
	origFamily, origStyle, origSize := s.font.GetFont()
	origColor := s.font.GetColor()
	defer func() {
		s.font.SetFont(origFamily, origStyle, origSize)
		s.font.SetColor(origColor)
	}()

	// Resolve each run's font fields against the original/default once.
	resolved := make([]resolvedRun, len(runs))
	for i, r := range runs {
		rr := r
		if rr.Family == "" {
			rr.Family = origFamily
		}
		if rr.Size == 0 {
			rr.Size = origSize
		}
		// rr.Style may legitimately be "" (Normal), so don't override.
		resolved[i] = resolvedRun{RichRun: rr}
	}

	width := cell.Width - prop.Left - prop.Right
	if width <= 0 {
		return
	}

	// Determine line height up front using the first run's resolved font.
	first := resolved[0]
	s.font.SetFont(first.Family, first.styleWithUnderline(), first.Size)
	lineHeight := s.font.GetHeight(first.Family, first.styleWithUnderline(), first.Size)
	lineMultiplier := prop.LineHeight
	if lineMultiplier <= 0 {
		lineMultiplier = 1.0
	}

	whiteSpace := normalizeRichTextWhiteSpace(prop.WhiteSpace)

	// Tokenise runs into renderable text spans + explicit line breaks.
	tokens := tokeniseRuns(resolved, whiteSpace)

	// Measure each token using its run's font and unicode translation.
	lastRunIdx := -1
	for i := range tokens {
		if tokens[i].isBreak {
			continue
		}
		r := resolved[tokens[i].runIdx]
		if tokens[i].runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			lastRunIdx = tokens[i].runIdx
		}
		translated := s.translateUnicode(tokens[i].text, r.Family)
		tokens[i].translated = translated
		tokens[i].width = s.pdf.GetStringWidth(translated)
		if r.LetterSpacing > 0 {
			runeCount := len([]rune(translated))
			if runeCount > 1 {
				tokens[i].width += float64(runeCount-1) * r.LetterSpacing
			}
		}
	}

	// Line-wrap: assign each token an x and lineY.
	lineY := 0
	curX := firstXForLine(lineY, prop.FirstLineIndent)
	noWrap := whiteSpace == "nowrap" || whiteSpace == richTextWhiteSpacePre
	lastRunIdx = -1
	for i := range tokens {
		t := &tokens[i]
		if t.isBreak {
			lineY++
			curX = firstXForLine(lineY, prop.FirstLineIndent)
			lastRunIdx = -1
			continue
		}
		r := resolved[t.runIdx]
		if t.runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			lastRunIdx = t.runIdx
		}

		lineStart := firstXForLine(lineY, prop.FirstLineIndent)
		if t.skipAtLineStart && curX == lineStart {
			t.skip = true
			continue
		}
		if !noWrap && curX > lineStart && curX+t.width > width {
			lineY++
			curX = firstXForLine(lineY, prop.FirstLineIndent)
			if t.skipAtLineStart {
				t.skip = true
				continue
			}
		}
		t.x = curX
		curX += t.width
		t.lineY = lineY
	}

	lineWidths := make(map[int]float64)
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		if right := t.x + t.width; right > lineWidths[t.lineY] {
			lineWidths[t.lineY] = right
		}
	}

	// Render.
	left, top, _, _ := s.pdf.GetMargins()
	lastRunIdx = -1
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		r := resolved[t.runIdx]
		if t.runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			if r.Color != nil {
				s.font.SetColor(r.Color)
			} else {
				s.font.SetColor(origColor)
			}
			lastRunIdx = t.runIdx
		}
		x := cell.X + prop.Left + alignmentOffset(prop.Align, width, lineWidths[t.lineY]) + t.x + left
		y := cell.Y + prop.Top + float64(t.lineY)*lineHeight*lineMultiplier + lineHeight + top

		if r.Background != nil {
			s.pdf.SetFillColor(r.Background.Red, r.Background.Green, r.Background.Blue)
			if r.Background.Alpha != nil && *r.Background.Alpha < 1 {
				s.pdf.SetAlpha(*r.Background.Alpha, "Normal")
				s.pdf.Rect(x, y-lineHeight, t.width, lineHeight, "F")
				s.pdf.SetAlpha(1, "Normal")
			} else {
				s.pdf.Rect(x, y-lineHeight, t.width, lineHeight, "F")
			}
			s.pdf.SetFillColor(255, 255, 255)
		}

		if r.TextShadow != nil && r.TextShadow.Color != nil {
			sc := r.TextShadow.Color
			s.pdf.SetTextColor(sc.Red, sc.Green, sc.Blue)
			s.pdf.Text(x+r.TextShadow.OffsetX, y+r.TextShadow.OffsetY, t.translated)
			// Restore run colour before drawing normal text.
			if r.Color != nil {
				s.pdf.SetTextColor(r.Color.Red, r.Color.Green, r.Color.Blue)
			} else {
				s.pdf.SetTextColor(origColor.Red, origColor.Green, origColor.Blue)
			}
		}

		if r.LetterSpacing > 0 {
			curX := x
			for _, ch := range t.translated {
				charStr := string(ch)
				s.pdf.Text(curX, y, charStr)
				curX += s.pdf.GetStringWidth(charStr) + r.LetterSpacing
			}
		} else {
			s.pdf.Text(x, y, t.translated)
		}

		if r.Hyperlink != nil {
			s.pdf.LinkString(x, y-lineHeight, t.width, lineHeight, *r.Hyperlink)
		}
		if r.LocalAnchor != "" && prop.AnchorResolver != nil {
			linkID := prop.AnchorResolver(r.LocalAnchor)
			s.pdf.Link(x, y-lineHeight, t.width, lineHeight, linkID)
		}
	}
}

// rtToken is the per-word state used by AddRichText's three-pass layout.
type rtToken struct {
	text            string
	translated      string
	runIdx          int
	width           float64
	x               float64
	lineY           int
	isBreak         bool
	skip            bool
	skipAtLineStart bool
}

// tokeniseRuns splits the resolved run sequence into renderable text spans,
// preserving or collapsing whitespace according to CSS white-space semantics.
func tokeniseRuns(runs []resolvedRun, whiteSpace string) []rtToken {
	var out []rtToken
	pendingCollapsedSpace := false
	for i, r := range runs {
		switch whiteSpace {
		case richTextWhiteSpacePre, "pre-wrap":
			out = append(out, tokenisePreservedText(r.Text, i)...)
		case "pre-line":
			out, pendingCollapsedSpace = appendCollapsedTokens(out, r.Text, i, true, pendingCollapsedSpace)
		default:
			out, pendingCollapsedSpace = appendCollapsedTokens(out, r.Text, i, false, pendingCollapsedSpace)
		}
	}
	return out
}

func appendCollapsedTokens(out []rtToken, text string, runIdx int, preserveNewlines bool, pendingSpace bool) ([]rtToken, bool) {
	var b strings.Builder
	flushWord := func() {
		if b.Len() == 0 {
			return
		}
		if pendingSpace && hasTextOnCurrentLine(out) {
			out = append(out, rtToken{text: " ", runIdx: runIdx, skipAtLineStart: true})
		}
		pendingSpace = false
		out = append(out, rtToken{text: b.String(), runIdx: runIdx})
		b.Reset()
	}
	for _, r := range text {
		if r == '\n' && preserveNewlines {
			flushWord()
			out = append(out, rtToken{runIdx: runIdx, isBreak: true})
			pendingSpace = false
			continue
		}
		if unicode.IsSpace(r) {
			flushWord()
			pendingSpace = true
			continue
		}
		b.WriteRune(r)
	}
	flushWord()
	return out, pendingSpace
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

func firstXForLine(lineY int, firstLineIndent float64) float64 {
	if lineY == 0 && firstLineIndent > 0 {
		return firstLineIndent
	}
	return 0
}

func alignmentOffset(a align.Type, width, lineWidth float64) float64 {
	slack := width - lineWidth
	if slack <= 0 {
		return 0
	}
	switch a {
	case align.Left, align.Top, align.Bottom, align.Middle:
		return 0
	case align.Center:
		return slack / 2
	case align.Right:
		return slack
	default:
		return 0
	}
}

func normalizeRichTextWhiteSpace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "nowrap", richTextWhiteSpacePre, "pre-wrap", "pre-line":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "normal"
	}
}

// styleWithUnderline appends "U" to the gofpdf style string when underline is set.
func (r resolvedRun) styleWithUnderline() fontstyle.Type {
	if r.Underline {
		return fontstyle.Type(string(r.Style) + "U")
	}
	return r.Style
}

// translateUnicode applies the gofpdf Unicode translator for built-in font families
// (Arial, Helvetica, Courier, Symbol, ZapBats) which expect Latin-1 codepoints.
// Comparison is case-insensitive because callers commonly use "Helvetica" while
// the fontfamily constants are lowercase. For custom (UTF-8) fonts text passes through.
func (s *Text) translateUnicode(text, family string) string {
	switch strings.ToLower(family) {
	case fontfamily.Arial, fontfamily.Helvetica, fontfamily.Symbol,
		fontfamily.ZapBats, fontfamily.Courier:
		return s.pdf.UnicodeTranslatorFromDescriptor("")(text)
	default:
		return text
	}
}
