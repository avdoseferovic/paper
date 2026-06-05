package paper

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/avdoseferovic/paper/v2/internal/merror"
	"github.com/avdoseferovic/paper/v2/pkg/consts/extension"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

var ErrCannotReadImageOptions = errors.New("could not read image options, maybe path/name is wrong")

func (g *provider) AddImageFromFile(file string, cell *entity.Cell, prop *props.Rect) {
	extensionStr := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
	image, err := g.loadImage(file, extensionStr)
	if err != nil {
		g.text.Add("could not load image", cell, merror.DefaultErrorText)
		return
	}

	g.AddImageFromBytes(image.Bytes, cell, prop, extension.Type(extensionStr))
}

func (g *provider) AddImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		g.text.Add("could not parse image bytes", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension, false)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add image to document", cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddBackgroundImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		g.text.Add("could not parse image bytes", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension, true)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add image to document", cell, merror.DefaultErrorText)
	}
	g.fpdf.SetHomeXY()
}

// GetDimensionsByImage is responsible for obtaining the dimensions of an image.
// If the image cannot be loaded, an error is returned.
func (g *provider) GetDimensionsByImage(file string) (*entity.Dimensions, error) {
	extensionStr := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
	img, err := g.loadImage(file, extensionStr)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Type(extensionStr))

	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// GetDimensionsByImageByte is responsible for obtaining the dimensions of an image.
// If the image cannot be loaded, an error is returned.
func (g *provider) GetDimensionsByImageByte(bytes []byte, extension extension.Type) (*entity.Dimensions, error) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension)
	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// loadImage is responsible for loading an image.
func (g *provider) loadImage(file, extensionStr string) (*entity.Image, error) {
	image, err := g.cache.GetImage(file, extension.Type(extensionStr))

	if err == nil {
		return image, nil
	}

	err = g.cache.LoadImage(file, extension.Type(extensionStr))
	if err != nil {
		return nil, err
	}

	return g.cache.GetImage(file, extension.Type(extensionStr))
}
