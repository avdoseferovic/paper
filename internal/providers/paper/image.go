package paper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	gofpdf "github.com/avdoseferovic/paper/internal/paperpdf"

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
	info := s.pdf.RegisterImageOptionsReader(
		name,
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytes.NewReader(img.Bytes),
	)
	if info == nil {
		return registeredImage{}, false
	}

	registered := registeredImage{
		name: name,
		info: info,
		dimensions: &entity.Dimensions{
			Width:  info.Width(),
			Height: info.Height(),
		},
	}
	s.registered[key] = registered

	return registered, true
}

func (s *Image) addImageToPdf(imageLabel string, info *gofpdf.ImageInfoType, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, flow bool,
) {
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
