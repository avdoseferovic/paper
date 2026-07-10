// Package image provides standalone image decoding helpers that can be adapted
// into Paper image components.
package image

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	imagecomp "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var (
	ErrCannotReadFile       = errors.New("image: cannot read file")
	ErrInvalidImage         = errors.New("image: invalid image")
	ErrInvalidDimensions    = errors.New("image: invalid dimensions")
	ErrUnexpectedImageType  = errors.New("image: unexpected image type")
	ErrUnsupportedExtension = errors.New("image: unsupported extension")
)

type Image struct {
	data      []byte
	extension extension.Type
	width     int
	height    int
}

func NewJPEG(data []byte) (*Image, error) {
	return newImage(data, extension.Jpg)
}

func LoadJPEG(path string) (*Image, error) {
	return loadImage(path, extension.Jpg)
}

func NewPNG(data []byte) (*Image, error) {
	return newImage(data, extension.Png)
}

func LoadPNG(path string) (*Image, error) {
	return loadImage(path, extension.Png)
}

func NewGIF(data []byte) (*Image, error) {
	return newImage(data, extension.Gif)
}

func LoadGIF(path string) (*Image, error) {
	return loadImage(path, extension.Gif)
}

func NewWebP(data []byte) (*Image, error) {
	return newImage(data, extension.WebP)
}

func LoadWebP(path string) (*Image, error) {
	return loadImage(path, extension.WebP)
}

func NewTIFF(data []byte) (*Image, error) {
	return newImage(data, extension.Tiff)
}

func LoadTIFF(path string) (*Image, error) {
	return loadImage(path, extension.Tiff)
}

func (img *Image) Bytes() []byte {
	if img == nil {
		return nil
	}
	return append([]byte(nil), img.data...)
}

func (img *Image) Extension() extension.Type {
	if img == nil {
		return ""
	}
	return img.extension
}

func (img *Image) Width() int {
	if img == nil {
		return 0
	}
	return img.width
}

func (img *Image) Height() int {
	if img == nil {
		return 0
	}
	return img.height
}

func (img *Image) Dimensions() *entity.Dimensions {
	if img == nil {
		return nil
	}
	return &entity.Dimensions{
		Width:  float64(img.width),
		Height: float64(img.height),
	}
}

func (img *Image) Component(rectProps ...props.Rect) core.Component {
	if img == nil {
		return imagecomp.NewFromBytes(nil, "", rectProps...)
	}
	return imagecomp.NewFromBytes(img.Bytes(), img.extension, rectProps...)
}

func (img *Image) Col(size int, rectProps ...props.Rect) core.Col {
	if img == nil {
		return imagecomp.NewFromBytesCol(size, nil, "", rectProps...)
	}
	return imagecomp.NewFromBytesCol(size, img.Bytes(), img.extension, rectProps...)
}

func (img *Image) Row(height float64, rectProps ...props.Rect) core.Row {
	if img == nil {
		return imagecomp.NewFromBytesRow(height, nil, "", rectProps...)
	}
	return imagecomp.NewFromBytesRow(height, img.Bytes(), img.extension, rectProps...)
}

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
