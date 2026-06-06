package cellwriter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"strings"

	gofpdf "github.com/avdoseferovic/paper/internal/paperpdf"
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type backgroundImageStyler struct {
	stylerTemplate
}

func NewBackgroundImageStyler(fpdf gofpdfwrapper.Fpdf) CellWriter {
	return &backgroundImageStyler{
		stylerTemplate: stylerTemplate{
			fpdf: fpdf,
			name: "backgroundImageStyler",
		},
	}
}

func (b *backgroundImageStyler) Apply(width, height float64, config *entity.Config, prop *props.Cell) {
	x, y := b.fpdf.GetXY()
	b.GoToNext(width, height, config, prop)
	if prop == nil || prop.BackgroundImage == nil || len(prop.BackgroundImage.Bytes) == 0 {
		return
	}
	b.drawBackgroundImage(x, y, width, height, prop.BackgroundImage)
}

func (b *backgroundImageStyler) drawBackgroundImage(x, y, width, height float64, image *props.CellBackgroundImage) {
	if width <= 0 || height <= 0 || image == nil {
		return
	}
	digest := sha256.Sum256(image.Bytes)
	name := "cell-bg-" + hex.EncodeToString(digest[:16])
	info := b.fpdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(image.Extension),
		},
		bytes.NewReader(image.Bytes),
	)
	if info == nil {
		return
	}

	w, h := backgroundImageSize(image.Size, info.Width(), info.Height(), width, height)
	ix, iy := backgroundImagePosition(image.Position, x, y, width, height, w, h)
	repeatX, repeatY := backgroundImageRepeat(image.Repeat)

	alpha, blend := b.fpdf.GetAlpha()
	if alpha < 1 {
		b.fpdf.SetAlpha(1, "Normal")
		defer b.fpdf.SetAlpha(alpha, blend)
	}
	b.fpdf.ClipRect(x, y, width, height, false)
	defer b.fpdf.ClipEnd()
	b.drawTiles(name, ix, iy, x, y, width, height, w, h, repeatX, repeatY)
}

func (b *backgroundImageStyler) drawTiles(name string, imageX, imageY, cellX, cellY, cellWidth, cellHeight, imageWidth, imageHeight float64, repeatX, repeatY bool) {
	if imageWidth <= 0 || imageHeight <= 0 {
		return
	}
	startX := imageX
	startY := imageY
	if repeatX {
		startX = tileStart(imageX, cellX, imageWidth)
	}
	if repeatY {
		startY = tileStart(imageY, cellY, imageHeight)
	}
	endX := cellX + cellWidth
	endY := cellY + cellHeight
	for y := startY; y < endY; y += imageHeight {
		for x := startX; x < endX; x += imageWidth {
			b.fpdf.Image(name, x, y, imageWidth, imageHeight, false, "", 0, "")
			if !repeatX {
				break
			}
		}
		if !repeatY {
			break
		}
	}
}

func tileStart(start, min, size float64) float64 {
	if size <= 0 || start <= min {
		return start
	}
	steps := math.Ceil((start - min) / size)
	return start - steps*size
}

func backgroundImageSize(value string, imageWidth, imageHeight, cellWidth, cellHeight float64) (float64, float64) {
	if imageWidth <= 0 || imageHeight <= 0 || cellWidth <= 0 || cellHeight <= 0 {
		return 0, 0
	}
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(tokens) == 0 || tokens[0] == "auto" {
		return imageWidth, imageHeight
	}
	aspect := imageHeight / imageWidth
	switch tokens[0] {
	case "contain":
		scale := math.Min(cellWidth/imageWidth, cellHeight/imageHeight)
		return imageWidth * scale, imageHeight * scale
	case "cover":
		scale := math.Max(cellWidth/imageWidth, cellHeight/imageHeight)
		return imageWidth * scale, imageHeight * scale
	}

	width := imageWidth
	height := imageHeight
	if tokens[0] != "auto" {
		width = parseBackgroundLength(tokens[0], cellWidth)
	}
	if len(tokens) > 1 && tokens[1] != "auto" {
		height = parseBackgroundLength(tokens[1], cellHeight)
	} else if tokens[0] != "auto" {
		height = width * aspect
	}
	if len(tokens) > 1 && tokens[0] == "auto" && tokens[1] != "auto" && aspect > 0 {
		width = height / aspect
	}
	if width <= 0 || height <= 0 {
		return imageWidth, imageHeight
	}
	return width, height
}

func backgroundImagePosition(value string, cellX, cellY, cellWidth, cellHeight, imageWidth, imageHeight float64) (float64, float64) {
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(tokens) == 0 {
		return cellX, cellY
	}
	spaceX := cellWidth - imageWidth
	spaceY := cellHeight - imageHeight
	if len(tokens) == 1 {
		token := tokens[0]
		switch token {
		case "left":
			return cellX, cellY + spaceY/2
		case "right":
			return cellX + spaceX, cellY + spaceY/2
		case "top":
			return cellX + spaceX/2, cellY
		case "bottom":
			return cellX + spaceX/2, cellY + spaceY
		case "center":
			return cellX + spaceX/2, cellY + spaceY/2
		default:
			return cellX + parseBackgroundOffset(token, spaceX), cellY + spaceY/2
		}
	}

	xToken, yToken := normalizeBackgroundPositionTokens(tokens[0], tokens[1])
	return cellX + parseBackgroundOffset(xToken, spaceX), cellY + parseBackgroundOffset(yToken, spaceY)
}

func normalizeBackgroundPositionTokens(first, second string) (string, string) {
	isVertical := func(v string) bool { return v == "top" || v == "bottom" }
	isHorizontal := func(v string) bool { return v == "left" || v == "right" }
	if isVertical(first) || isHorizontal(second) {
		return second, first
	}
	return first, second
}

func parseBackgroundOffset(value string, freeSpace float64) float64 {
	switch value {
	case "left", "top":
		return 0
	case "center":
		return freeSpace / 2
	case "right", "bottom":
		return freeSpace
	}
	if strings.HasSuffix(value, "%") {
		v, ok := parseFloat(strings.TrimSuffix(value, "%"))
		if !ok {
			return 0
		}
		return freeSpace * v / 100
	}
	return parseBackgroundLength(value, freeSpace)
}

func backgroundImageRepeat(value string) (repeatX, repeatY bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "repeat":
		return true, true
	case "no-repeat":
		return false, false
	case "repeat-x":
		return true, false
	case "repeat-y":
		return false, true
	default:
		return true, true
	}
}

func parseBackgroundLength(value string, percentBase float64) float64 {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasSuffix(value, "%"):
		v, ok := parseFloat(strings.TrimSuffix(value, "%"))
		if !ok {
			return 0
		}
		return percentBase * v / 100
	case strings.HasSuffix(value, "mm"):
		v, _ := parseFloat(strings.TrimSuffix(value, "mm"))
		return v
	case strings.HasSuffix(value, "cm"):
		v, _ := parseFloat(strings.TrimSuffix(value, "cm"))
		return v * 10
	case strings.HasSuffix(value, "pt"):
		v, _ := parseFloat(strings.TrimSuffix(value, "pt"))
		return v * 0.352778
	case strings.HasSuffix(value, "px"):
		v, _ := parseFloat(strings.TrimSuffix(value, "px"))
		return v * 0.264583
	default:
		v, _ := parseFloat(value)
		return v
	}
}

func parseFloat(value string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return v, err == nil
}
