package paper

import (
	"strings"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

const (
	richTextWhiteSpaceNoWrap  = "nowrap"
	richTextWhiteSpacePre     = "pre"
	richTextWhiteSpacePreLine = "pre-line"
	richTextWhiteSpacePreWrap = "pre-wrap"
)

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

func (r resolvedRun) hasInlineBox() bool {
	return r.Background != nil || (r.BorderColor != nil && r.BorderWidth > 0) || len(r.BoxShadows) > 0
}

func (r resolvedRun) inlinePadLeft() float64 {
	if r.BgPadLeft != 0 || r.BgPadRight != 0 {
		return r.BgPadLeft
	}
	return r.BgPadX
}

func (r resolvedRun) inlinePadRight() float64 {
	if r.BgPadLeft != 0 || r.BgPadRight != 0 {
		return r.BgPadRight
	}
	return r.BgPadX
}

func (r resolvedRun) inlineBorderWidth() float64 {
	if r.BorderColor != nil && r.BorderWidth > 0 {
		return r.BorderWidth
	}
	return 0
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
	resolved := resolveRichTextRuns(runs, origFamily, origSize)

	width := cell.Width - prop.Left - prop.Right
	if width <= 0 {
		return
	}

	lineHeight, lineMultiplier := s.richTextLineMetrics(resolved, prop)

	whiteSpace := normalizeRichTextWhiteSpace(prop.WhiteSpace)

	// Collect glyphs the code page translation replaced, deduped across all
	// tokens, so one render issue is recorded per AddRichText call.
	var missing []rune
	seenMissing := make(map[rune]bool)

	tokens, lineWidths := layoutRichTextTokens(resolved, richTextLayoutInput{
		prop:       prop,
		width:      width,
		whiteSpace: whiteSpace,
		measure: func(r resolvedRun, text string) (string, float64) {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			translated := s.translateUnicode(text, r.Family)
			if s.issueSink != nil {
				for _, missingRune := range unsupportedGlyphs(text, translated) {
					if !seenMissing[missingRune] {
						seenMissing[missingRune] = true
						missing = append(missing, missingRune)
					}
				}
			}
			return translated, s.pdf.GetStringWidth(translated)
		},
	})
	s.reportMissingRunes(missing)

	s.renderRichTextTokens(tokens, lineWidths, resolved, cell, prop, lineHeight, lineMultiplier, origColor)
}

// MeasureRichText returns the height AddRichText would need for the same runs,
// cell, and paragraph properties without drawing anything.
func (s *Text) MeasureRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) float64 {
	if len(runs) == 0 || cell == nil {
		return 0
	}
	if prop == nil {
		prop = &props.RichText{}
	}

	origFamily, origStyle, origSize := s.font.GetFont()
	defer s.font.SetFont(origFamily, origStyle, origSize)

	resolved := resolveRichTextRuns(runs, origFamily, origSize)
	width := cell.Width - prop.Left - prop.Right
	if width <= 0 {
		return 0
	}

	lineHeight, lineMultiplier := s.richTextLineMetrics(resolved, prop)
	whiteSpace := normalizeRichTextWhiteSpace(prop.WhiteSpace)
	tokens, _ := layoutRichTextTokens(resolved, richTextLayoutInput{
		prop:       prop,
		width:      width,
		whiteSpace: whiteSpace,
		measure: func(r resolvedRun, text string) (string, float64) {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			translated := s.translateUnicode(text, r.Family)
			return translated, s.pdf.GetStringWidth(translated)
		},
	})

	return float64(richTextLineCount(tokens))*lineHeight*lineMultiplier + prop.Top + prop.Bottom
}

func resolveRichTextRuns(runs []props.RichRun, defaultFamily string, defaultSize float64) []resolvedRun {
	resolved := make([]resolvedRun, len(runs))
	for i, r := range runs {
		rr := r
		if rr.Family == "" {
			rr.Family = defaultFamily
		}
		if rr.Size == 0 {
			rr.Size = defaultSize
		}
		if rr.SizeScale > 0 {
			rr.Size *= rr.SizeScale
		}
		// rr.Style may legitimately be "" (Normal), so don't override.
		resolved[i] = resolvedRun{RichRun: rr}
	}
	return resolved
}

func (s *Text) richTextLineMetrics(resolved []resolvedRun, prop *props.RichText) (float64, float64) {
	lineHeight := 0.0
	for _, run := range resolved {
		h := s.font.GetHeight(run.Family, run.styleWithUnderline(), run.Size)
		if h > lineHeight {
			lineHeight = h
		}
	}
	if lineHeight <= 0 {
		lineHeight = 1
	}

	lineMultiplier := 1.0
	if prop != nil && prop.LineHeight > 0 {
		lineMultiplier = prop.LineHeight
	}
	if imageHeight := maxRichRunImageHeight(resolved); imageHeight > lineHeight*lineMultiplier {
		lineMultiplier = imageHeight / lineHeight
	}
	// Reserve vertical room for inline-box pills (background/border runs with
	// vertical padding). The per-line advance is uniform (lineHeight ×
	// lineMultiplier), so without this a wrapped row of padded pills overlaps the
	// row above — the vertical analogue of reserving horizontal pill padding.
	if pillHeight := maxRichRunInlineBoxPillHeight(resolved, lineHeight); pillHeight > lineHeight*lineMultiplier {
		lineMultiplier = pillHeight / lineHeight
	}
	return lineHeight, lineMultiplier
}

// maxRichRunInlineBoxPillHeight returns the tallest inline-box pill, including
// its vertical padding and border — the full painted height from
// drawRunBackground (boxLineHeight + 2×(BgPadY+border)).
func maxRichRunInlineBoxPillHeight(runs []resolvedRun, lineHeight float64) float64 {
	maxHeight := 0.0
	for i := range runs {
		r := runs[i]
		if !r.hasInlineBox() {
			continue
		}
		h := richRunInlineBoxLineHeight(r, lineHeight) + 2*(r.BgPadY+r.inlineBorderWidth())
		if h > maxHeight {
			maxHeight = h
		}
	}
	return maxHeight
}

func richTextLineCount(tokens []rtToken) int {
	maxLine := 0
	for _, t := range tokens {
		if t.isBreak {
			if nextLine := t.lineY + 1; nextLine > maxLine {
				maxLine = nextLine
			}
			continue
		}
		if t.skip {
			continue
		}
		if t.lineY > maxLine {
			maxLine = t.lineY
		}
	}
	return maxLine + 1
}

func firstXForLine(lineY int, firstLineIndent float64) float64 {
	if lineY == 0 && firstLineIndent > 0 {
		return firstLineIndent
	}
	return 0
}

func alignmentOffset(a consts.Align, width, lineWidth float64) float64 {
	slack := width - lineWidth
	if slack <= 0 {
		return 0
	}
	switch a {
	case consts.AlignLeft, consts.AlignTop, consts.AlignBottom, consts.AlignMiddle, consts.AlignJustify:
		return 0
	case consts.AlignCenter:
		return slack / 2
	case consts.AlignRight:
		return slack
	default:
		return 0
	}
}

func normalizeRichTextWhiteSpace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case richTextWhiteSpaceNoWrap, richTextWhiteSpacePre, richTextWhiteSpacePreWrap, richTextWhiteSpacePreLine:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "normal"
	}
}

func maxRichRunImageHeight(runs []resolvedRun) float64 {
	maxHeight := 0.0
	for _, run := range runs {
		if run.Image != nil && run.Image.Height > maxHeight {
			maxHeight = run.Image.Height
		}
		if run.InlineBoxHeight > maxHeight {
			maxHeight = run.InlineBoxHeight
		}
	}
	return maxHeight
}

// styleWithUnderline appends "U" to the gofpdf style string when underline is set.
func (r resolvedRun) styleWithUnderline() fontstyle.Type {
	if r.Underline {
		return fontstyle.Type(string(r.Style) + "U")
	}
	return r.Style
}

func (r resolvedRun) hasFixedInlineBox() bool {
	return r.InlineBoxWidth > 0 || r.InlineBoxHeight > 0
}
