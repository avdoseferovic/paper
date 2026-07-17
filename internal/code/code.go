package code

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/png"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

var (
	ErrCannotEncodePNG          = errors.New("cannot encode png")
	ErrCannotScaleBarcode       = errors.New("cannot scale barcode")
	ErrCannotEncodeQRcode       = errors.New("cannot encode qr code")
	ErrCannotEncodeDataMatrix   = errors.New("cannot encode data matrix")
	errInvalidBarcodeDimensions = errors.New("target dimensions must be positive")
	errBarcodeScaleWouldDiscard = errors.New("target dimensions cannot discard barcode modules")
)

type Code struct{}

// New create a Code (Singleton).
func New() *Code {
	return &Code{}
}

// GenDataMatrix is responsible to generate a data matrix byte array.
func (c *Code) GenDataMatrix(code string) (*entity.Image, error) {
	dataMatrix, err := encodeDataMatrix(code)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodeDataMatrix, err)
	}

	return c.getImage(dataMatrix)
}

// GenQr is responsible to generate a qr code byte array.
func (c *Code) GenQr(code string) (*entity.Image, error) {
	qrCode, err := encodeQR(code)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodeQRcode, err)
	}

	return c.getImage(qrCode)
}

// GenBar is responsible to generate a barcode byte array.
func (c *Code) GenBar(code string, _ *entity.Cell, prop *props.Barcode) (*entity.Image, error) {
	barcodeGen := getBarcodeClosure(prop.Type)

	barCode, err := barcodeGen(code)
	if err != nil {
		return nil, err
	}

	width := float64(barCode.Bounds().Dx())
	heightPercentFromWidth := prop.Proportion.Height / prop.Proportion.Width
	height := int(width * heightPercentFromWidth)

	scaledBarCode, err := scaleBarcode(barCode, int(width), height)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotScaleBarcode, err)
	}

	return c.getImage(scaledBarCode)
}

func getBarcodeClosure(
	barcodeType consts.BarcodeType,
) func(code string) (image.Image, error) {
	switch barcodeType {
	case consts.BarcodeEAN:
		return encodeEAN
	case consts.BarcodeCode128:
		return encodeCode128
	default:
		return encodeCode128
	}
}

func scaleBarcode(source image.Image, width, height int) (image.Image, error) {
	bounds := source.Bounds()
	if width <= 0 || height <= 0 || bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, errInvalidBarcodeDimensions
	}
	if width < bounds.Dx() || height < bounds.Dy() {
		return nil, errBarcodeScaleWouldDiscard
	}

	scaled := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		sourceY := bounds.Min.Y + y*bounds.Dy()/height
		for x := range width {
			sourceX := bounds.Min.X + x*bounds.Dx()/width
			scaled.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	return scaled, nil
}

func (c *Code) getImage(img image.Image) (*entity.Image, error) {
	var buf bytes.Buffer

	dst := image.NewPaletted(img.Bounds(), palette.Plan9)
	drawer := draw.Drawer(draw.Src)
	drawer.Draw(dst, dst.Bounds(), img, img.Bounds().Min)

	err := png.Encode(&buf, dst)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotEncodePNG, err)
	}

	imgEntity := &entity.Image{
		Bytes:     buf.Bytes(),
		Extension: extension.Png,
		Dimensions: &entity.Dimensions{
			Width:  float64(dst.Bounds().Dx()),
			Height: float64(dst.Bounds().Dy()),
		},
	}

	return imgEntity, nil
}
