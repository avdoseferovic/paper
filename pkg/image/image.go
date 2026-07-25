// Package image provides standalone image decoding helpers that can be adapted
// into Paper image components.
package image

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	_ "image/gif"  // registered with image.DecodeConfig so this format can be inspected
	_ "image/jpeg" // registered with image.DecodeConfig so this format can be inspected
	_ "image/png"  // registered with image.DecodeConfig so this format can be inspected
	"os"

	_ "golang.org/x/image/tiff" // registered with image.DecodeConfig so this format can be inspected
	_ "golang.org/x/image/webp" // registered with image.DecodeConfig so this format can be inspected

	imagecomp "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Errors reported when an image cannot be loaded or does not match the
// extension it was loaded as.
var (
	ErrCannotReadFile       = errors.New("image: cannot read file")
	ErrInvalidImage         = errors.New("image: invalid image")
	ErrInvalidDimensions    = errors.New("image: invalid dimensions")
	ErrUnexpectedImageType  = errors.New("image: unexpected image type")
	ErrUnsupportedExtension = errors.New("image: unsupported extension")
)

// Image is a decoded image together with its format and pixel size. Use the
// Component, Col, Row, and AutoRow methods to place it in a document.
type Image struct {
	data      []byte
	extension extension.Type
	width     int
	height    int
}

// NewJPEG decodes JPEG data. It returns an error if the bytes are not a valid
// JPEG image.
func NewJPEG(data []byte) (*Image, error) {
	return newImage(data, extension.Jpg)
}

// LoadJPEG reads the file at path and decodes it as JPEG.
func LoadJPEG(path string) (*Image, error) {
	return loadImage(path, extension.Jpg)
}

// NewPNG decodes PNG data. It returns an error if the bytes are not a valid
// PNG image.
func NewPNG(data []byte) (*Image, error) {
	return newImage(data, extension.Png)
}

// LoadPNG reads the file at path and decodes it as PNG.
func LoadPNG(path string) (*Image, error) {
	return loadImage(path, extension.Png)
}

// NewGIF decodes GIF data. It returns an error if the bytes are not a valid
// GIF image.
func NewGIF(data []byte) (*Image, error) {
	return newImage(data, extension.Gif)
}

// LoadGIF reads the file at path and decodes it as GIF.
func LoadGIF(path string) (*Image, error) {
	return loadImage(path, extension.Gif)
}

// NewWebP decodes WebP data. It returns an error if the bytes are not a valid
// WebP image.
func NewWebP(data []byte) (*Image, error) {
	return newImage(data, extension.WebP)
}

// LoadWebP reads the file at path and decodes it as WebP.
func LoadWebP(path string) (*Image, error) {
	return loadImage(path, extension.WebP)
}

// NewTIFF decodes TIFF data. It returns an error if the bytes are not a valid
// TIFF image.
func NewTIFF(data []byte) (*Image, error) {
	return newImage(data, extension.Tiff)
}

// LoadTIFF reads the file at path and decodes it as TIFF.
func LoadTIFF(path string) (*Image, error) {
	return loadImage(path, extension.Tiff)
}

// Bytes returns a copy of the encoded image data.
func (img *Image) Bytes() []byte {
	if img == nil {
		return nil
	}
	return append([]byte(nil), img.data...)
}

// Extension returns the image format.
func (img *Image) Extension() extension.Type {
	if img == nil {
		return ""
	}
	return img.extension
}

// Width returns the image width in pixels.
func (img *Image) Width() int {
	if img == nil {
		return 0
	}
	return img.width
}

// Height returns the image height in pixels.
func (img *Image) Height() int {
	if img == nil {
		return 0
	}
	return img.height
}

// Dimensions returns the pixel width and height as an entity.Dimensions.
func (img *Image) Dimensions() *entity.Dimensions {
	if img == nil {
		return nil
	}
	return &entity.Dimensions{
		Width:  float64(img.width),
		Height: float64(img.height),
	}
}

// Component returns the image as a component that can be added to a column.
func (img *Image) Component(rectProps ...props.Rect) core.Component {
	if img == nil {
		return imagecomp.NewFromBytes(nil, "", rectProps...)
	}
	return imagecomp.NewFromBytes(img.Bytes(), img.extension, rectProps...)
}

// Col returns the image as a column spanning size grid units.
func (img *Image) Col(size int, rectProps ...props.Rect) core.Col {
	if img == nil {
		return imagecomp.NewFromBytesCol(size, nil, "", rectProps...)
	}
	return imagecomp.NewFromBytesCol(size, img.Bytes(), img.extension, rectProps...)
}

// Row returns the image as a row of the given height in millimeters.
func (img *Image) Row(height float64, rectProps ...props.Rect) core.Row {
	if img == nil {
		return imagecomp.NewFromBytesRow(height, nil, "", rectProps...)
	}
	return imagecomp.NewFromBytesRow(height, img.Bytes(), img.extension, rectProps...)
}

// AutoRow returns the image as a row whose height follows the image itself.
func (img *Image) AutoRow(rectProps ...props.Rect) core.Row {
	if img == nil {
		return imagecomp.NewAutoFromBytesRow(nil, "", rectProps...)
	}
	return imagecomp.NewAutoFromBytesRow(img.Bytes(), img.extension, rectProps...)
}

func loadImage(path string, ext extension.Type) (*Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotReadFile, err)
	}
	return newImage(data, ext)
}

func newImage(data []byte, ext extension.Type) (*Image, error) {
	if !ext.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedExtension, ext)
	}
	cfg, format, err := goimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidImage, ext, err)
	}
	if !formatMatchesExtension(format, ext) {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedImageType, ext, format)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("%w: %dx%d", ErrInvalidDimensions, cfg.Width, cfg.Height)
	}
	return &Image{
		data:      append([]byte(nil), data...),
		extension: ext,
		width:     cfg.Width,
		height:    cfg.Height,
	}, nil
}

func formatMatchesExtension(format string, ext extension.Type) bool {
	switch ext {
	case extension.Jpg, extension.Jpeg:
		return format == "jpeg"
	case extension.Png:
		return format == "png"
	case extension.Gif:
		return format == "gif"
	case extension.WebP:
		return format == "webp"
	case extension.Tif, extension.Tiff:
		return format == "tiff"
	case extension.Svg:
		return false
	default:
		return false
	}
}
