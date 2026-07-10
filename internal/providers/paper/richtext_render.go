package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"

	gofpdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/consts"
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

	// Inline boxes are painted in a first pass, grouped per run per line, so a
	// run that tokenises into several words renders ONE continuous background
	// or outlined rounded "pill" instead of per-word boxes.
	s.renderRunBackgrounds(tokens, lineWidths, resolved, cell, prop, lineHeight, lineMultiplier, left, top)

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
		y += richRunRenderBaselineOffset(r, lineHeight)

		if r.Hidden {
			continue
		}
		if t.isImage(r) {
			s.renderTokenImage(r, x, y)
			s.renderTokenLinks(r, prop, x, y, t.width, r.Image.Height)
			continue
		}
		if t.isFixedInlineBox(r) {
			s.renderTokenLinks(r, prop, x, y, t.width, richRunInlineBoxLineHeight(r, lineHeight))
			continue
		}
		s.renderTokenShadows(r, origColor, x, y, t.translated)
		s.renderTokenText(r, x, y, t.translated)
		s.renderTokenLinks(r, prop, x, y, t.width, lineHeight)
	}
}

// renderRunBackgrounds paints each run's inline box once per line, spanning the
// run's contiguous tokens on that line. Runs with BgRadius/BgPadX/BgPadY or
// asymmetric horizontal padding render as rounded, padded pills; plain
// backgrounds render as tight rects.
func (s *Text) renderRunBackgrounds(
	tokens []rtToken,
	lineWidths map[int]float64,
	resolved []resolvedRun,
	cell *entity.Cell,
	prop *props.RichText,
	lineHeight, lineMultiplier, left, top float64,
) {
	width := cell.Width - prop.Left - prop.Right
	curKey, curRun, curLine := 0, -1, -1
	var gx1, gx2 float64
	flush := func() {
		if curRun < 0 {
			return
		}
		r := resolved[curRun]
		yTop := cell.Y + prop.Top + float64(curLine)*lineHeight*lineMultiplier + top
		yTop += richRunBaselineOffset(r, lineHeight)
		s.drawRunBackground(r, gx1, yTop, gx2-gx1, lineHeight)
		curKey, curRun = 0, -1
	}
	for _, t := range tokens {
		if t.isBreak || t.skip {
			continue
		}
		r := resolved[t.runIdx]
		if !r.hasInlineBox() || r.Hidden {
			flush()
			continue
		}
		x := cell.X + prop.Left + alignmentOffset(prop.Align, width, lineWidths[t.lineY]) + t.x + left
		key := richTextInlineBoxKey(t, resolved)
		if key != curKey || t.lineY != curLine {
			flush()
			curKey, curRun, curLine = key, t.runIdx, t.lineY
			gx1, gx2 = x, x+t.width
		} else {
			if preferInlineBoxGeometryRun(resolved[curRun], r) {
				curRun = t.runIdx
			}
			gx2 = x + t.width
		}
	}
	flush()
}

func preferInlineBoxGeometryRun(current, candidate resolvedRun) bool {
	if current.VerticalOffset == 0 && current.VerticalAlign == "" {
		return false
	}
	return candidate.VerticalOffset == 0 && candidate.VerticalAlign == ""
}

func (s *Text) drawRunBackground(r resolvedRun, x, yTop, w, lineHeight float64) {
	if !r.hasInlineBox() || w <= 0 {
		return
	}
	padLeft := r.inlinePadLeft()
	padRight := r.inlinePadRight()
	borderWidth := r.inlineBorderWidth()
	bx := x - padLeft - borderWidth
	boxLineHeight := richRunInlineBoxLineHeight(r, lineHeight)
	bw := w + padLeft + padRight + 2*borderWidth
	bh := boxLineHeight + 2*(r.BgPadY+borderWidth)
	// Center the pill on the text's optical middle. The glyph baseline sits at
	// yTop+lineHeight (matching renderTokenText); the cap-box middle is ~0.35em
	// above it. Anchoring the pill here keeps the label vertically centered
	// instead of riding high with empty space below.
	baseline := yTop + lineHeight
	capMiddle := baseline - r.Size*0.352778*0.36
	by := capMiddle - bh/2

	radius := r.BgRadius
	if radius > bh/2 {
		radius = bh / 2
	}
	if radius > bw/2 {
		radius = bw / 2
	}
	s.drawRunBoxShadows(r, bx, by, bw, bh, radius)

	hasFill := r.Background != nil
	hasStroke := r.BorderColor != nil && r.BorderWidth > 0
	if !hasFill && !hasStroke {
		return
	}

	drawMode := "D"
	translucent := false
	if r.Background != nil {
		drawMode = "F"
		setPDFFillColor(s.pdf, r.Background)
		translucent = r.Background.Alpha != nil && *r.Background.Alpha < 1
	}
	if r.BorderColor != nil && r.BorderWidth > 0 {
		if r.Background != nil {
			drawMode = "FD"
		}
		s.pdf.SetDrawColor(r.BorderColor.Red, r.BorderColor.Green, r.BorderColor.Blue)
		s.pdf.SetLineWidth(r.BorderWidth)
	}
	if translucent {
		s.pdf.SetAlpha(*r.Background.Alpha, "Normal")
	}
	if r.BgRadius > 0 {
		if isCompactFullRoundInlineBox(bw, bh, radius) {
			s.pdf.Ellipse(bx+bw/2, by+bh/2, bw/2, bh/2, 0, drawMode)
		} else {
			s.pdf.RoundedRect(bx, by, bw, bh, radius, "1234", drawMode)
		}
	} else {
		s.pdf.Rect(bx, by, bw, bh, drawMode)
	}
	if translucent {
		s.pdf.SetAlpha(1, "Normal")
	}
	if r.Background != nil {
		s.pdf.SetFillColor(255, 255, 255)
	}
	if r.BorderColor != nil && r.BorderWidth > 0 {
		s.pdf.SetDrawColor(0, 0, 0)
		s.pdf.SetLineWidth(consts.DefaultLineThickness)
	}
}

func richRunInlineBoxLineHeight(r resolvedRun, lineHeight float64) float64 {
	boxLineHeight := lineHeight
	if r.InlineBoxHeight > boxLineHeight {
		boxLineHeight = r.InlineBoxHeight
	}
	if r.LineHeight <= 0 {
		return boxLineHeight
	}
	base := lineHeight
	if fontSizeHeight := r.Size * 0.352778; fontSizeHeight > base {
		base = fontSizeHeight
	}
	return base * r.LineHeight
}

func isCompactFullRoundInlineBox(w, h, radius float64) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	shortSide := min(w, h)
	longSide := max(w, h)
	if longSide/shortSide > 1.25 {
		return false
	}
	return nearlyEqualInlineRadius(radius, shortSide/2)
}

func nearlyEqualInlineRadius(a, b float64) bool {
	const epsilon = 0.000001
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= epsilon
}

func (s *Text) drawRunBoxShadows(r resolvedRun, x, y, w, h, radius float64) {
	if len(r.BoxShadows) == 0 || w <= 0 || h <= 0 {
		return
	}
	for _, shadow := range r.BoxShadows {
		if shadow.Inset {
			continue
		}
		s.drawRunBoxShadow(shadow, x, y, w, h, radius)
	}
	s.pdf.SetAlpha(1, "Normal")
	s.pdf.SetFillColor(255, 255, 255)
}

func (s *Text) drawRunBoxShadow(shadow props.Shadow, x, y, w, h, radius float64) {
	cr, cg, cb := 0, 0, 0
	if shadow.Color != nil {
		cr, cg, cb = shadow.Color.Red, shadow.Color.Green, shadow.Color.Blue
	}
	alpha := 1.0
	if shadow.Color != nil && shadow.Color.Alpha != nil {
		alpha = *shadow.Color.Alpha
	}
	rx := x + shadow.OffsetX - shadow.Spread
	ry := y + shadow.OffsetY - shadow.Spread
	rw := w + 2*shadow.Spread
	rh := h + 2*shadow.Spread
	if rw <= 0 || rh <= 0 {
		return
	}
	if shadow.BlurRadius <= 0 {
		s.drawRunShadowBox(rx, ry, rw, rh, radius+shadow.Spread, cr, cg, cb, alpha)
		return
	}
	alphas := [3]float64{0.3, 0.5, 0.8}
	step := shadow.BlurRadius / float64(len(alphas))
	for i, alphaFactor := range alphas {
		expand := shadow.BlurRadius - float64(i)*step
		s.drawRunShadowBox(
			rx-expand,
			ry-expand,
			rw+2*expand,
			rh+2*expand,
			radius+shadow.Spread+expand,
			cr,
			cg,
			cb,
			alpha*alphaFactor,
		)
	}
}

func (s *Text) drawRunShadowBox(x, y, w, h, radius float64, r, g, b int, alpha float64) {
	if w <= 0 || h <= 0 || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	s.pdf.SetFillColor(r, g, b)
	s.pdf.SetAlpha(alpha, "Normal")
	if radius > 0 {
		if radius > h/2 {
			radius = h / 2
		}
		if radius > w/2 {
			radius = w / 2
		}
		s.pdf.RoundedRect(x, y, w, h, radius, "1234", "F")
		return
	}
	s.pdf.Rect(x, y, w, h, "F")
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
	if r.VerticalOffset != 0 {
		return r.VerticalOffset
	}
	switch strings.ToLower(r.VerticalAlign) {
	case "sub":
		return lineHeight * 0.2
	case "super", "sup":
		return -lineHeight * 0.35
	default:
		return 0
	}
}

func richRunRenderBaselineOffset(r resolvedRun, lineHeight float64) float64 {
	if r.Image != nil && r.VerticalOffset != 0 {
		return -r.VerticalOffset
	}
	return richRunBaselineOffset(r, lineHeight)
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
		setPDFTextColor(s.pdf, sc)
		s.pdf.Text(x+shadow.OffsetX, y+shadow.OffsetY, translated)
		painted = true
	}
	if !painted {
		return
	}
	// Restore run colour before drawing normal text.
	if r.Color != nil {
		setPDFTextColor(s.pdf, r.Color)
	} else {
		setPDFTextColor(s.pdf, origColor)
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
