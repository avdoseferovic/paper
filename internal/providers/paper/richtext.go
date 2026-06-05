package paper

import (
	"strings"
	"unicode"

	"github.com/avdoseferovic/paper/v2/pkg/consts/align"
	"github.com/avdoseferovic/paper/v2/pkg/consts/fontfamily"
	"github.com/avdoseferovic/paper/v2/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
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
// Layout state is computed by pure helpers; PDF drawing is isolated in render helpers.
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

	tokens, lineWidths := layoutRichTextTokens(resolved, richTextLayoutInput{
		prop:       prop,
		width:      width,
		whiteSpace: whiteSpace,
		measure: func(r resolvedRun, text string) (string, float64) {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			translated := s.translateUnicode(text, r.Family)
			return translated, s.pdf.GetStringWidth(translated)
		},
	})

	s.renderRichTextTokens(tokens, lineWidths, resolved, cell, prop, lineHeight, lineMultiplier, origColor)
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
