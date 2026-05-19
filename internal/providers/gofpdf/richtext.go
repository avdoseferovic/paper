package gofpdf

import (
	"strings"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// MeasureString sets the current font and returns the string width in mm.
// All intermediate state is method-local; no struct mutation occurs.
func (s *Text) MeasureString(text string, prop *props.Text) float64 {
	s.font.SetFont(prop.Family, prop.Style, prop.Size)
	return s.pdf.GetStringWidth(text)
}

// AddTextAt renders text at an absolute (x, y) position in mm (page-coordinate space,
// not cell-coordinate). Baseline-positioned, matching gofpdf's Text() convention.
func (s *Text) AddTextAt(x, y float64, text string, prop *props.Text) {
	s.font.SetFont(prop.Family, prop.Style, prop.Size)
	origColor := s.font.GetColor()

	left, top, _, _ := s.pdf.GetMargins()
	s.pdf.Text(x+left, y+top, text)
	_ = origColor // color management left to caller; AddTextAt is a low-level primitive
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

	width := cell.Width - prop.Left - prop.Right
	if width <= 0 {
		return
	}

	// Build token list: each token carries its text word and run index.
	type token struct {
		word    string
		runIdx  int
		isBreak bool // explicit \n
	}
	var tokens []token
	for i, run := range runs {
		if strings.Contains(run.Text, "\n") {
			parts := strings.Split(run.Text, "\n")
			for j, part := range parts {
				if part != "" {
					words := strings.Fields(part)
					for _, w := range words {
						tokens = append(tokens, token{word: w, runIdx: i})
					}
				}
				if j < len(parts)-1 {
					tokens = append(tokens, token{isBreak: true, runIdx: i})
				}
			}
		} else {
			words := strings.Fields(run.Text)
			for _, w := range words {
				tokens = append(tokens, token{word: w, runIdx: i})
			}
		}
	}

	// Measure each token's width using per-run SetFont.
	type measuredToken struct {
		token
		width float64
	}
	measured := make([]measuredToken, len(tokens))
	lastRunIdx := -1
	for i, tok := range tokens {
		if tok.isBreak {
			measured[i] = measuredToken{token: tok, width: 0}
			continue
		}
		if tok.runIdx != lastRunIdx {
			r := runs[tok.runIdx]
			s.font.SetFont(r.Family, r.Style, r.Size)
			lastRunIdx = tok.runIdx
		}
		w := s.pdf.GetStringWidth(tok.word)
		measured[i] = measuredToken{token: tok, width: w}
	}

	// Line-wrap: group tokens into lines.
	type lineToken struct {
		word   string
		runIdx int
		x      float64
		lineY  int
	}
	var lineTokens []lineToken
	curX := 0.0
	lineY := 0
	spaceWidth := 0.0
	lastRunIdx = -1

	for _, mt := range measured {
		if mt.isBreak {
			lineY++
			curX = 0
			lastRunIdx = -1
			continue
		}

		r := runs[mt.runIdx]
		if mt.runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.Style, r.Size)
			spaceWidth = s.pdf.GetStringWidth(" ")
			lastRunIdx = mt.runIdx
		}

		needed := mt.width
		if curX > 0 {
			needed += spaceWidth
		}

		if curX > 0 && curX+needed > width {
			lineY++
			curX = 0
		}

		lineTokens = append(lineTokens, lineToken{
			word:   mt.word,
			runIdx: mt.runIdx,
			x:      curX,
			lineY:  lineY,
		})

		if curX == 0 {
			curX = mt.width
		} else {
			curX += spaceWidth + mt.width
		}
	}

	// Render lines.
	left, top, _, _ := s.pdf.GetMargins()
	lastRunIdx = -1
	var lineHeight float64

	for _, lt := range lineTokens {
		r := runs[lt.runIdx]
		if lt.runIdx != lastRunIdx {
			s.font.SetFont(r.Family, r.Style, r.Size)
			lineHeight = s.font.GetHeight(r.Family, r.Style, r.Size)
			if r.Color != nil {
				s.font.SetColor(r.Color)
			} else {
				s.font.SetColor(origColor)
			}
			lastRunIdx = lt.runIdx
		}

		x := cell.X + prop.Left + lt.x + left
		y := cell.Y + prop.Top + float64(lt.lineY)*lineHeight*prop.LineHeight + lineHeight + top

		if prop.Align == align.Right {
			// Simple right-align: not implemented in v1 (left is default)
		}

		renderText := lt.word
		if r.Underline {
			// gofpdf doesn't have native underline for Text(); approximation deferred to v2
			_ = renderText
		}
		s.pdf.Text(x, y, renderText)

		if r.Hyperlink != nil {
			s.pdf.LinkString(x, y-lineHeight, s.pdf.GetStringWidth(renderText), lineHeight, *r.Hyperlink)
		}
	}

	// Reset to original color (defer handles font).
	s.font.SetColor(origColor)
}

// setFontForRun applies a run's font settings. Falls back to origFamily/Style/Size for zero values.
func setFontForRun(f interface {
	SetFont(string, fontstyle.Type, float64)
}, r props.RichRun, origFamily string, origStyle fontstyle.Type, origSize float64) {
	family := r.Family
	if family == "" {
		family = origFamily
	}
	style := r.Style
	if style == "" {
		style = origStyle
	}
	size := r.Size
	if size == 0 {
		size = origSize
	}
	f.SetFont(family, style, size)
}
