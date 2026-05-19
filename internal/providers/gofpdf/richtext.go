package gofpdf

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
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

// AddRichText renders a paragraph of mixed inline runs within cell.
// Font state is captured on entry and restored via defer so callers are unaffected.
// All intermediate state (line buffer, cursors) is method-local — safe for concurrent use.
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

	// Tokenise runs into words + explicit line breaks.
	tokens := tokeniseRuns(resolved)

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
		translated := s.translateUnicode(tokens[i].word, r.Family)
		tokens[i].translated = translated
		tokens[i].width = s.pdf.GetStringWidth(translated)
	}

	// Line-wrap: assign each token an x and lineY.
	curX := 0.0
	lineY := 0
	spaceWidth := 0.0
	lastRunIdx = -1
	for i := range tokens {
		t := &tokens[i]
		if t.isBreak {
			lineY++
			curX = 0
			lastRunIdx = -1
			continue
		}
		r := resolved[t.runIdx]
		if t.runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.styleWithUnderline(), r.Size)
			spaceWidth = s.pdf.GetStringWidth(" ")
			lastRunIdx = t.runIdx
		}

		need := t.width
		if curX > 0 {
			need += spaceWidth
		}
		if curX > 0 && curX+need > width {
			lineY++
			curX = 0
		}

		// Position this token: first on the line at x=0, otherwise after the
		// running x cursor plus a space gap. The gap MUST be added here so
		// adjacent words in the same run actually separate visually.
		if curX == 0 {
			t.x = 0
			curX = t.width
		} else {
			t.x = curX + spaceWidth
			curX = t.x + t.width
		}
		t.lineY = lineY
	}

	// Render.
	left, top, _, _ := s.pdf.GetMargins()
	lastRunIdx = -1
	for _, t := range tokens {
		if t.isBreak {
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
		x := cell.X + prop.Left + t.x + left
		y := cell.Y + prop.Top + float64(t.lineY)*lineHeight*lineMultiplier + lineHeight + top
		s.pdf.Text(x, y, t.translated)
		if r.Hyperlink != nil {
			s.pdf.LinkString(x, y-lineHeight, t.width, lineHeight, *r.Hyperlink)
		}
	}
}

// rtToken is the per-word state used by AddRichText's three-pass layout.
type rtToken struct {
	word       string
	translated string
	runIdx     int
	width      float64
	x          float64
	lineY      int
	isBreak    bool
}

// tokeniseRuns splits the resolved run sequence into words preserving order
// and inserting explicit isBreak tokens at every \n boundary.
func tokeniseRuns(runs []resolvedRun) []rtToken {
	var out []rtToken
	for i, r := range runs {
		if !strings.Contains(r.Text, "\n") {
			for _, w := range strings.Fields(r.Text) {
				out = append(out, rtToken{word: w, runIdx: i})
			}
			continue
		}
		parts := strings.Split(r.Text, "\n")
		for j, part := range parts {
			for _, w := range strings.Fields(part) {
				out = append(out, rtToken{word: w, runIdx: i})
			}
			if j < len(parts)-1 {
				out = append(out, rtToken{runIdx: i, isBreak: true})
			}
		}
	}
	return out
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
// For custom (UTF-8) fonts the text is returned unchanged.
func (s *Text) translateUnicode(text, family string) string {
	switch family {
	case fontfamily.Arial, fontfamily.Helvetica, fontfamily.Symbol,
		fontfamily.ZapBats, fontfamily.Courier:
		return s.pdf.UnicodeTranslatorFromDescriptor("")(text)
	default:
		return text
	}
}
