package paper_test

import (
	"testing"

	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
)

func TestFromBytes(t *testing.T) {
	t.Parallel()
	t.Run("when extension is not valid, should return error", func(t *testing.T) {
		t.Parallel()
		// Act
		img, err := gofpdf.FromBytes([]byte{1, 2, 3}, "invalid")

		// Assert
		assert.Nil(t, img)
		assert.NotNil(t, err)
	})
	t.Run("when extension is valid, should return image", func(t *testing.T) {
		t.Parallel()
		// Act
		img, err := gofpdf.FromBytes([]byte{1, 2, 3}, extension.Jpg)

		// Assert
		assert.NotNil(t, img)
		assert.Nil(t, err)
	})
	t.Run("when extension is svg, should return image", func(t *testing.T) {
		t.Parallel()
		// Act
		img, err := gofpdf.FromBytes([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), extension.Svg)

		// Assert
		assert.NotNil(t, img)
		assert.Nil(t, err)
	})
}
