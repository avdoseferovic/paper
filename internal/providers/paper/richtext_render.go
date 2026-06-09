package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"

	gofpdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
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
		y += richRunBaselineOffset(r, lineHeight)

		if r.Hidden {
			continue
		}
		if t.isImage(r) {
			s.renderTokenImage(r, x, y)
			s.renderTokenLinks(r, prop, x, y, t.width, r.Image.Height)
			continue
		}
		s.renderTokenBackground(r, x, y, t.width, lineHeight)
		s.renderTokenShadows(r, origColor, x, y, t.translated)
		s.renderTokenText(r, x, y, t.translated)
		s.renderTokenLinks(r, prop, x, y, t.width, lineHeight)
	}
}

func (s *Text) renderTokenImage(r resolvedRun, x, baselineY float64) {
	if r.Image == nil || len(r.Image.Bytes) == 0 || r.Image.Width <= 0 || r.Image.Height <= 0 {
		return
	}
	digest := sha256.Sum256(r.Image.Bytes)
	name := "rich-image-" + hex.EncodeToString(digest[:16])
	info := s.pdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(r.Image.Extension),
		},
		bytes.NewReader(r.Image.Bytes),
	)
	if info == nil {
		return
	}
	if usesRichImageObjectBox(r.Image) {
		boxX := x
		boxY := baselineY - r.Image.Height
		imageWidth, imageHeight := validImageInfoSize(info, r.Image.Width, r.Image.Height)
		rect := objectImageRect(r.Image.ObjectFit, r.Image.ObjectPosition, imageWidth, imageHeight, boxX, boxY, r.Image.Width, r.Image.Height)
		s.pdf.ClipRect(boxX, boxY, r.Image.Width, r.Image.Height, false)
		s.pdf.Image(name, rect.X, rect.Y, rect.Width, rect.Height, false, "", 0, "")
		s.pdf.ClipEnd()
		return
	}
	s.pdf.Image(name, x, baselineY-r.Image.Height, r.Image.Width, r.Image.Height, false, "", 0, "")
}

func usesRichImageObjectBox(image *props.RichImage) bool {
	return image != nil && (strings.TrimSpace(image.ObjectFit) != "" || strings.TrimSpace(image.ObjectPosition) != "")
}

func validImageInfoSize(info *gofpdf.ImageInfoType, fallbackWidth, fallbackHeight float64) (float64, float64) {
	width := info.Width()
	height := info.Height()
	if width <= 0 || height <= 0 || math.IsNaN(width) || math.IsNaN(height) || math.IsInf(width, 0) || math.IsInf(height, 0) {
		return fallbackWidth, fallbackHeight
	}
	return width, height
}

func richRunBaselineOffset(r resolvedRun, lineHeight float64) float64 {
	switch strings.ToLower(r.VerticalAlign) {
	case "sub":
		return lineHeight * 0.2
	case "super", "sup":
		return -lineHeight * 0.35
	default:
		return 0
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

func (s *Text) renderTokenShadows(r resolvedRun, origColor *props.Color, x, y float64, translated string) {
	shadows := r.TextShadows
	if len(shadows) == 0 && r.TextShadow != nil {
		shadows = []props.Shadow{*r.TextShadow}
	}
	if len(shadows) == 0 {
		return
	}
	painted := false
	for _, shadow := range shadows {
		if shadow.Color == nil {
			continue
		}
		sc := shadow.Color
		s.pdf.SetTextColor(sc.Red, sc.Green, sc.Blue)
		s.pdf.Text(x+shadow.OffsetX, y+shadow.OffsetY, translated)
		painted = true
	}
	if !painted {
		return
	}
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
