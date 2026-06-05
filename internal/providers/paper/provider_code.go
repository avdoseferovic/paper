package paper

import (
	"github.com/avdoseferovic/paper/v2/internal/merror"
	"github.com/avdoseferovic/paper/v2/pkg/consts/barcode"
	"github.com/avdoseferovic/paper/v2/pkg/consts/extension"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func (g *provider) AddMatrixCode(code string, cell *entity.Cell, prop *props.Rect) {
	img, err := g.loadCode(code, "matrix-code-", g.code.GenDataMatrix)
	if err != nil {
		message := "could not generate matrixcode"
		g.recordRenderIssue("matrixcode.generate", message, err)
		g.text.Add(message, cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension.Png, false)
	if err != nil {
		message := "could not add matrixcode to document"
		g.recordRenderIssue("matrixcode.add", message, err)
		g.fpdf.ClearError()
		g.text.Add(message, cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddQrCode(code string, cell *entity.Cell, prop *props.Rect) {
	img, err := g.loadCode(code, "qr-code-", g.code.GenQr)
	if err != nil {
		message := "could not generate qrcode"
		g.recordRenderIssue("qrcode.generate", message, err)
		g.text.Add(message, cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension.Png, false)
	if err != nil {
		message := "could not add qrcode to document"
		g.recordRenderIssue("qrcode.add", message, err)
		g.fpdf.ClearError()
		g.text.Add(message, cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddBarCode(code string, cell *entity.Cell, prop *props.Barcode) {
	image, err := g.cache.GetImage(g.getBarcodeImageName("bar-code-"+code, prop), extension.Png)
	if err != nil {
		image, err = g.code.GenBar(code, cell, prop)
	}
	if err != nil {
		message := "could not generate barcode"
		g.recordRenderIssue("barcode.generate", message, err)
		g.text.Add(message, cell, merror.DefaultErrorText)
		return
	}

	g.cache.AddImage(g.getBarcodeImageName("bar-code-"+code, prop), image)
	err = g.image.Add(image, cell, g.cfg.Margins, prop.ToRectProp(), extension.Png, false)
	if err != nil {
		message := "could not add barcode to document"
		g.recordRenderIssue("barcode.add", message, err)
		g.fpdf.ClearError()
		g.text.Add(message, cell, merror.DefaultErrorText)
	}
}

// GetDimensionsByMatrixCode is responsible for obtaining the dimensions of an MatrixCode.
// If the image cannot be loaded, an error is returned.
func (g *provider) GetDimensionsByMatrixCode(code string) (*entity.Dimensions, error) {
	img, err := g.loadCode(code, "matrix-code-", g.code.GenDataMatrix)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Png)

	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// GetDimensionsByQrCode is responsible for obtaining the dimensions of an QrCode.
// If the image cannot be loaded, an error is returned.
func (g *provider) GetDimensionsByQrCode(code string) (*entity.Dimensions, error) {
	img, err := g.loadCode(code, "qr-code-", g.code.GenQr)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Png)
	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

func (g *provider) getBarcodeImageName(code string, prop *props.Barcode) string {
	if prop == nil {
		return code + string(barcode.Code128)
	}

	return code + string(prop.Type)
}

// loadCode is responsible for loading generated codes from cache or generating them.
func (g *provider) loadCode(code, codeType string, generate func(code string) (*entity.Image, error)) (*entity.Image, error) {
	image, err := g.cache.GetImage(codeType+code, extension.Png)
	if err != nil {
		image, err = generate(code)
	} else {
		return image, nil
	}
	if err != nil {
		return nil, err
	}
	g.cache.AddImage(codeType+code, image)

	return image, nil
}
