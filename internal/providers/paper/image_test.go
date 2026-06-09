package paper_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	gofpdf2 "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/internal/math"
	gofpdf "github.com/avdoseferovic/paper/internal/paperpdf"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/stretchr/testify/mock"

	"github.com/avdoseferovic/paper/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	t.Parallel()
	image := gofpdf2.NewImage(mocks.NewFpdf(t), mocks.NewMath(t))

	assert.NotNil(t, image)
	assert.Equal(t, "*paper.Image", fmt.Sprintf("%T", image))
}

func TestImage_Add(t *testing.T) {
	t.Parallel()
	t.Run("when RegisterImageOptionsReader return nil, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).Return(nil)

		image := gofpdf2.NewImage(pdf, mocks.NewMath(t))

		// Act
		err := image.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.NotNil(t, err)
	})
	t.Run("when prop is not center, should work properly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).Return(&gofpdf.ImageInfoType{})
		pdf.EXPECT().Image(mock.Anything, 30.0, 35.0, 98.0, mock.Anything, true, "", 0, "")

		m := math.New()

		image := gofpdf2.NewImage(pdf, m)

		// Act
		err := image.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.Nil(t, err)
	})
	t.Run("when prop is center, should work properly", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		rect.Center = true
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).Return(&gofpdf.ImageInfoType{})
		pdf.EXPECT().Image(mock.Anything, 21.0, mock.Anything, 98.0, mock.Anything, true, "", 0, "")

		m := math.New()

		image := gofpdf2.NewImage(pdf, m)

		// Act
		err := image.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.Nil(t, err)
	})
	t.Run("when image is measured and reused, should register image only once", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := fixture.CellEntity()
		margins := fixture.MarginsEntity()
		rect := fixture.RectProp()
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}
		registeredNames := make([]string, 0, 2)

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).
			Return(&gofpdf.ImageInfoType{}).
			Once()
		pdf.EXPECT().Image(mock.Anything, 30.0, 35.0, 98.0, mock.Anything, true, "", 0, "").
			Run(func(imageName string, _ float64, _ float64, _ float64, _ float64, _ bool, _ string, _ int, _ string) {
				registeredNames = append(registeredNames, imageName)
			}).
			Twice()

		m := math.New()

		image := gofpdf2.NewImage(pdf, m)

		// Act
		dimensions := image.GetImageDimensions(&img, img.Extension)
		err1 := image.Add(&img, &cell, &margins, &rect, img.Extension, true)
		err2 := image.Add(&img, &cell, &margins, &rect, img.Extension, true)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.Len(t, registeredNames, 2)
		assert.Equal(t, registeredNames[0], registeredNames[1])
	})
	t.Run("when extension is svg, should rasterise and register as png", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{
			Bytes: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 16">
  <rect width="32" height="16" fill="#24807c"/>
  <text x="3" y="12" fill="#ffffff" font-size="8">SVG</text>
</svg>`),
			Extension: extension.Svg,
		}
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(extension.Png),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, mock.MatchedBy(func(reader io.Reader) bool {
			data, err := io.ReadAll(reader)
			return err == nil && bytes.HasPrefix(data, []byte("\x89PNG"))
		})).Return(&gofpdf.ImageInfoType{}).Once()

		image := gofpdf2.NewImage(pdf, mocks.NewMath(t))

		// Act
		dimensions := image.GetImageDimensions(img, img.Extension)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Equal(t, 32.0, dimensions.Width)
		assert.Equal(t, 16.0, dimensions.Height)
	})
}

func TestImage_GetImageDimensions(t *testing.T) {
	t.Parallel()
	t.Run("when RegisterImageOptionsReader return nil, should return nil", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).Return(nil)

		image := gofpdf2.NewImage(pdf, mocks.NewMath(t))

		// Act
		dimensions := image.GetImageDimensions(&img, img.Extension)

		// Assert
		assert.Nil(t, dimensions)
	})

	t.Run("when RegisterImageOptionsReader return info, should return dimensions", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := fixture.ImageEntity()
		options := gofpdf.ImageOptions{
			ReadDpi:   false,
			ImageType: string(img.Extension),
		}

		pdf := mocks.NewFpdf(t)
		pdf.EXPECT().RegisterImageOptionsReader(mock.Anything, options, bytes.NewReader(img.Bytes)).Return(&gofpdf.ImageInfoType{})

		image := gofpdf2.NewImage(pdf, mocks.NewMath(t))

		// Act
		dimensions := image.GetImageDimensions(&img, img.Extension)

		// Assert
		assert.NotNil(t, dimensions)
	})
}
