package paper

import (
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func (s *Text) renderRichTextTokens(
	tokens []rtToken,
	lineWidths map[int]float64,
	resolved []resolvedRun,
	cell *entity.Cell,
	prop *props.RichText,
	lineHeight float64,
	lineMultiplier float64,
	origColor *props.Color,
) {
	left, top, _, _ := s.pdf.GetMargins()
	lastRunIdx := -1
	width := cell.Width - prop.Left - prop.Right
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

		s.renderTokenBackground(r, x, y, t.width, lineHeight)
		s.renderTokenShadow(r, origColor, x, y, t.translated)
		s.renderTokenText(r, x, y, t.translated)
		s.renderTokenLinks(r, prop, x, y, t.width, lineHeight)
	}
}

func (s *Text) renderTokenBackground(r resolvedRun, x, y, width, lineHeight float64) {
	if r.Background == nil {
		return
	}
	s.pdf.SetFillColor(r.Background.Red, r.Background.Green, r.Background.Blue)
	if r.Background.Alpha != nil && *r.Background.Alpha < 1 {
		s.pdf.SetAlpha(*r.Background.Alpha, "Normal")
		s.pdf.Rect(x, y-lineHeight, width, lineHeight, "F")
		s.pdf.SetAlpha(1, "Normal")
	} else {
		s.pdf.Rect(x, y-lineHeight, width, lineHeight, "F")
	}
	s.pdf.SetFillColor(255, 255, 255)
}

func (s *Text) renderTokenShadow(r resolvedRun, origColor *props.Color, x, y float64, translated string) {
	if r.TextShadow == nil || r.TextShadow.Color == nil {
		return
	}
	sc := r.TextShadow.Color
	s.pdf.SetTextColor(sc.Red, sc.Green, sc.Blue)
	s.pdf.Text(x+r.TextShadow.OffsetX, y+r.TextShadow.OffsetY, translated)
	// Restore run colour before drawing normal text.
	if r.Color != nil {
		s.pdf.SetTextColor(r.Color.Red, r.Color.Green, r.Color.Blue)
	} else {
		s.pdf.SetTextColor(origColor.Red, origColor.Green, origColor.Blue)
	}
}

func (s *Text) renderTokenText(r resolvedRun, x, y float64, translated string) {
	if r.LetterSpacing > 0 {
		curX := x
		for _, ch := range translated {
			charStr := string(ch)
			s.pdf.Text(curX, y, charStr)
			curX += s.pdf.GetStringWidth(charStr) + r.LetterSpacing
		}
		return
	}
	s.pdf.Text(x, y, translated)
}

func (s *Text) renderTokenLinks(r resolvedRun, prop *props.RichText, x, y, width, lineHeight float64) {
	if r.Hyperlink != nil {
		s.pdf.LinkString(x, y-lineHeight, width, lineHeight, *r.Hyperlink)
	}
	if r.LocalAnchor != "" && prop.AnchorResolver != nil {
		linkID := prop.AnchorResolver(r.LocalAnchor)
		s.pdf.Link(x, y-lineHeight, width, lineHeight, linkID)
	}
}
