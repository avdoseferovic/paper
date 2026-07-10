package imagecodec

import (
	"bytes"
	"errors"
	"fmt"
	goimage "image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"strings"

	svgraster "github.com/avdoseferovic/paper/internal/svg"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var errInvalidImageDimensions = errors.New("image decode produced invalid dimensions")

type Normalized struct {
	Bytes     []byte
	Extension extension.Type
	Width     int
	Height    int
}

func (n Normalized) HasDimensions() bool {
	return n.Width > 0 && n.Height > 0
}

func NormalizeForPDF(data []byte, ext extension.Type) (Normalized, error) {
	ext = canonicalExtension(ext)
	switch ext {
	case extension.Svg:
		pngBytes, width, height, err := svgraster.Rasterize(data, 0, 0)
		if err != nil {
			return Normalized{}, fmt.Errorf("svg rasterize: %w", err)
		}
		return Normalized{
			Bytes:     pngBytes,
			Extension: extension.Png,
			Width:     width,
			Height:    height,
		}, nil
	case extension.WebP, extension.Tif, extension.Tiff:
		return rasterToPNG(data, ext)
	case extension.Jpg, extension.Jpeg, extension.Png, extension.Gif:
		return Normalized{
			Bytes:     data,
			Extension: ext,
		}, nil
	default:
		return Normalized{
			Bytes:     data,
			Extension: ext,
		}, nil
	}
}

func canonicalExtension(ext extension.Type) extension.Type {
	return extension.Type(strings.ToLower(string(ext)))
}

func rasterToPNG(data []byte, ext extension.Type) (Normalized, error) {
	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return Normalized{}, fmt.Errorf("%s decode: %w", ext, err)
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return Normalized{}, fmt.Errorf("%w: %s %dx%d", errInvalidImageDimensions, ext, width, height)
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return Normalized{}, fmt.Errorf("%s png encode: %w", ext, err)
	}

	return Normalized{
		Bytes:     buf.Bytes(),
		Extension: extension.Png,
		Width:     width,
		Height:    height,
	}, nil
}
