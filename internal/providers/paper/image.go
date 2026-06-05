package paper

import (
	"bytes"
	"errors"

	"github.com/google/uuid"
	gofpdf "github.com/johnfercher/paper/v2/internal/paperpdf"

	"github.com/johnfercher/paper/v2/internal/providers/paper/gofpdfwrapper"
	"github.com/johnfercher/paper/v2/pkg/consts/extension"
	"github.com/johnfercher/paper/v2/pkg/core"
	"github.com/johnfercher/paper/v2/pkg/core/entity"
	"github.com/johnfercher/paper/v2/pkg/props"
)

var ErrCouldNotRegisterImageOptions = errors.New("could not register image options, maybe path/name is wrong")

type Image struct {
	pdf  gofpdfwrapper.Fpdf
	math core.Math
}

// NewImage create an Image.
func NewImage(pdf gofpdfwrapper.Fpdf, math core.Math) *Image {
	return &Image{
		pdf,
		math,
	}
}

// GetImageDimensions is responsible for loading the image in PDF and returning its dimensions.
func (s *Image) GetImageDimensions(img *entity.Image, extension extension.Type) (*entity.Dimensions, uuid.UUID) {
	imageID, _ := uuid.NewRandom()

	info := s.pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytes.NewReader(img.Bytes),
	)
	if info == nil {
		return nil, imageID
	}

	return &entity.Dimensions{Width: info.Width(), Height: info.Height()}, imageID
}

// Add use a byte array to add image to PDF.
func (s *Image) Add(img *entity.Image, cell *entity.Cell, margins *entity.Margins,
	prop *props.Rect, extension extension.Type, flow bool,
) error {
	imageID, _ := uuid.NewRandom()

	info := s.pdf.RegisterImageOptionsReader(
		imageID.String(),
		gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension),
		},
		bytes.NewReader(img.Bytes),
	)

	if info == nil {
		return ErrCouldNotRegisterImageOptions
	}

	s.addImageToPdf(imageID.String(), info, cell, margins, prop, flow)
	return nil
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
