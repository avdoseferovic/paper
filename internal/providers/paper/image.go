package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	gofpdf "github.com/avdoseferovic/paper/internal/paperpdf"
	svgraster "github.com/avdoseferovic/paper/internal/svg"

	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

var ErrCouldNotRegisterImageOptions = errors.New("could not register image options, maybe path/name is wrong")

type Image struct {
	pdf        gofpdfwrapper.Fpdf
	math       core.Math
	registered map[imageCacheKey]registeredImage
}

type imageCacheKey struct {
	extension extension.Type
	digest    [sha256.Size]byte
}

type registeredImage struct {
	name       string
	info       *gofpdf.ImageInfoType
	dimensions *entity.Dimensions
}

// NewImage create an Image.
func NewImage(pdf gofpdfwrapper.Fpdf, math core.Math) *Image {
	return &Image{
		pdf:        pdf,
		math:       math,
		registered: make(map[imageCacheKey]registeredImage),
	}
}

// GetImageDimensions is responsible for loading the image in PDF and returning its dimensions.
func (s *Image) GetImageDimensions(img *entity.Image, extension extension.Type) *entity.Dimensions {
	registered, ok := s.registerImage(img, extension)
	if !ok {
		return nil
	}

	return &entity.Dimensions{
		Width:  registered.dimensions.Width,
		Height: registered.dimensions.Height,
	}
}

// Add use a byte array to add image to PDF.
func (s *Image) Add(img *entity.Image, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, extension extension.Type, flow bool,
) error {
	registered, ok := s.registerImage(img, extension)
	if !ok {
		return ErrCouldNotRegisterImageOptions
	}

	s.addImageToPdf(registered.name, registered.info, cell, margins, prop, flow)
	return nil
}

func (s *Image) registerImage(img *entity.Image, extension extension.Type) (registeredImage, bool) {
	key := imageCacheKey{
		extension: extension,
		digest:    sha256.Sum256(img.Bytes),
	}

	if s.registered == nil {
		s.registered = make(map[imageCacheKey]registeredImage)
	}

	if registered, ok := s.registered[key]; ok {
		return registered, true
	}

	name := "image-" + hex.EncodeToString(key.digest[:16])
	imageBytes, registerExtension, dimensions, err := normalizeImageForRegistration(img.Bytes, extension)
	if err != nil {
		return registeredImage{}, false
	}
	info := s.pdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(registerExtension),
		},
		bytes.NewReader(imageBytes),
	)
	if info == nil {
		return registeredImage{}, false
	}
	if dimensions == nil {
		dimensions = &entity.Dimensions{
			Width:  info.Width(),
			Height: info.Height(),
		}
	}

	registered := registeredImage{
		name:       name,
		info:       info,
		dimensions: dimensions,
	}
	s.registered[key] = registered

	return registered, true
}

func normalizeImageForRegistration(bytes []byte, ext extension.Type) ([]byte, extension.Type, *entity.Dimensions, error) {
	if ext != extension.Svg {
		return bytes, ext, nil, nil
	}
	pngBytes, width, height, err := svgraster.Rasterize(bytes, 0, 0)
	if err != nil {
		return nil, "", nil, fmt.Errorf("svg rasterize: %w", err)
	}
	return pngBytes, extension.Png, &entity.Dimensions{
		Width:  float64(width),
		Height: float64(height),
	}, nil
}

func (s *Image) addImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, flow bool,
) {
	if usesObjectBox(prop) {
		s.addObjectImageToPdf(imageLabel, info, cell, margins, prop, flow)
		return
	}
	dimensions := s.math.Resize(&entity.Dimensions{
		Width:  info.Width(),
		Height: info.Height(),
	}, cell.GetDimensions(), prop.Percent, prop.JustReferenceWidth)

	rectCell := &entity.Cell{X: prop.Left, Y: prop.Top, Width: dimensions.Width, Height: dimensions.Height}

	if prop.Center {
		rectCell = s.math.GetInnerCenterCell(dimensions, cell.GetDimensions())
	}

	s.pdf.Image(imageLabel, cell.X+rectCell.X+margins.Left, cell.Y+rectCell.Y+margins.Top,
		rectCell.Width, rectCell.Height, flow, "", 0, "")
}

func usesObjectBox(prop *props.Rect) bool {
	return prop != nil && (strings.TrimSpace(prop.ObjectFit) != "" || strings.TrimSpace(prop.ObjectPosition) != "")
}

func (s *Image) addObjectImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, flow bool,
) {
	boxX := cell.X + prop.Left + margins.Left
	boxY := cell.Y + prop.Top + margins.Top
	boxWidth := cell.Width - prop.Left
	boxHeight := cell.Height - prop.Top
	if boxWidth <= 0 || boxHeight <= 0 {
		return
	}
	rect := objectImageRect(prop.ObjectFit, prop.ObjectPosition, info.Width(), info.Height(), boxX, boxY, boxWidth, boxHeight)
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	s.pdf.ClipRect(boxX, boxY, boxWidth, boxHeight, false)
	s.pdf.Image(imageLabel, rect.X, rect.Y, rect.Width, rect.Height, flow, "", 0, "")
	s.pdf.ClipEnd()
}

func objectImageRect(fit, position string, imageWidth, imageHeight, boxX, boxY, boxWidth, boxHeight float64) entity.Cell {
	if imageWidth <= 0 || imageHeight <= 0 || boxWidth <= 0 || boxHeight <= 0 {
		return entity.Cell{X: boxX, Y: boxY, Width: boxWidth, Height: boxHeight}
	}
	width, height := objectImageSize(fit, imageWidth, imageHeight, boxWidth, boxHeight)
	x, y := objectImagePosition(position, boxX, boxY, boxWidth, boxHeight, width, height)
	return entity.Cell{X: x, Y: y, Width: width, Height: height}
}

func objectImageSize(fit string, imageWidth, imageHeight, boxWidth, boxHeight float64) (float64, float64) {
	fit = strings.ToLower(strings.TrimSpace(fit))
	if fit == "" {
		fit = "contain"
	}
	switch fit {
	case "fill":
		return boxWidth, boxHeight
	case "cover":
		scale := math.Max(boxWidth/imageWidth, boxHeight/imageHeight)
		return imageWidth * scale, imageHeight * scale
	case "none":
		return imageWidth, imageHeight
	case "scale-down":
		containW, containH := objectImageSize("contain", imageWidth, imageHeight, boxWidth, boxHeight)
		if imageWidth <= boxWidth && imageHeight <= boxHeight {
			return imageWidth, imageHeight
		}
		return containW, containH
	default:
		scale := math.Min(boxWidth/imageWidth, boxHeight/imageHeight)
		return imageWidth * scale, imageHeight * scale
	}
}

func objectImagePosition(value string, boxX, boxY, boxWidth, boxHeight, imageWidth, imageHeight float64) (float64, float64) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "center"
	}
	tokens := strings.Fields(value)
	spaceX := boxWidth - imageWidth
	spaceY := boxHeight - imageHeight
	switch len(tokens) {
	case 0:
		return boxX + spaceX/2, boxY + spaceY/2
	case 1:
		token := tokens[0]
		if isVerticalObjectPosition(token) {
			return boxX + spaceX/2, boxY + objectPositionOffset(token, spaceY)
		}
		return boxX + objectPositionOffset(token, spaceX), boxY + spaceY/2
	default:
		xToken, yToken := normalizeObjectPositionTokens(tokens[0], tokens[1])
		return boxX + objectPositionOffset(xToken, spaceX), boxY + objectPositionOffset(yToken, spaceY)
	}
}

func normalizeObjectPositionTokens(first, second string) (string, string) {
	if isVerticalObjectPosition(first) && !isVerticalObjectPosition(second) {
		return second, first
	}
	return first, second
}

func isVerticalObjectPosition(token string) bool {
	return token == "top" || token == "bottom"
}

func objectPositionOffset(value string, freeSpace float64) float64 {
	switch value {
	case "left", "top":
		return 0
	case "right", "bottom":
		return freeSpace
	case "center":
		return freeSpace / 2
	}
	if pctValue, ok := strings.CutSuffix(value, "%"); ok {
		pct, err := strconv.ParseFloat(pctValue, 64)
		if err == nil {
			return freeSpace * pct / 100
		}
	}
	if length, ok := parseObjectPositionLength(value); ok {
		return length
	}
	return freeSpace / 2
}

func parseObjectPositionLength(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	units := map[string]float64{
		"mm": 1,
		"cm": 10,
		"in": 25.4,
		"pt": 0.352778,
		"px": 0.264583,
	}
	for suffix, factor := range units {
		if trimmed, ok := strings.CutSuffix(value, suffix); ok {
			n, err := strconv.ParseFloat(trimmed, 64)
			if err != nil {
				return 0, false
			}
			return n * factor, true
		}
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
