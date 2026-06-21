package entity

import (
	"fmt"

	"github.com/avdoseferovic/paper/pkg/consts/extension"
)

const trimSizeDefault = 10

// Image is the representation of an image that can be added to the pdf.
type Image struct {
	Bytes      []byte
	Extension  extension.Type
	Dimensions *Dimensions
	// PageCell optionally places the image in physical page coordinates,
	// measured from the page's top-left corner.
	PageCell *Cell
	// ObjectFit controls how a positioned page image is sized inside PageCell.
	ObjectFit string
}

// AppendMap adds the Image fields to the map.
func (i *Image) AppendMap(m map[string]any) map[string]any {
	lenBytes := len(i.Bytes)
	if lenBytes != 0 {
		trimSize := min(lenBytes, trimSizeDefault)
		m["entity_image_bytes"] = fmt.Sprintf("%v", i.Bytes[:trimSize])
	}

	if i.Extension != "" {
		m["entity_extension"] = i.Extension
	}

	if i.Dimensions != nil {
		m = i.Dimensions.AppendMap("background", m)
	}

	if i.PageCell != nil {
		m["background_page_cell_x"] = i.PageCell.X
		m["background_page_cell_y"] = i.PageCell.Y
		m["background_page_cell_width"] = i.PageCell.Width
		m["background_page_cell_height"] = i.PageCell.Height
	}

	if i.ObjectFit != "" {
		m["background_object_fit"] = i.ObjectFit
	}

	return m
}
