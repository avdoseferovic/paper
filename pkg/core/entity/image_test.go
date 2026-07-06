package entity_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/internal/assert"

	"github.com/avdoseferovic/paper/pkg/consts/extension"
)

func TestImage_AppendMap(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := fixtureImage()
	m := make(map[string]any)

	// Act
	m = sut.AppendMap(m)

	// Assert
	assert.Equal(t, "[1 2 3]", m["entity_image_bytes"])
	assert.Equal(t, extension.Png, m["entity_extension"])
	assert.Equal(t, 100.0, m["background_dimension_width"])
	assert.Equal(t, 200.0, m["background_dimension_height"])
	assert.Equal(t, 1.0, m["background_page_cell_x"])
	assert.Equal(t, 2.0, m["background_page_cell_y"])
	assert.Equal(t, 3.0, m["background_page_cell_width"])
	assert.Equal(t, 4.0, m["background_page_cell_height"])
	assert.Equal(t, "fill", m["background_object_fit"])
}

func fixtureImage() entity.Image {
	dimensions := fixtureDimensions()
	return entity.Image{
		Bytes:      []byte{1, 2, 3},
		Extension:  extension.Png,
		Dimensions: &dimensions,
		PageCell:   &entity.Cell{X: 1, Y: 2, Width: 3, Height: 4},
		ObjectFit:  "fill",
	}
}
