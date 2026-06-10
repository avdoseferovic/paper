package core

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// WatermarkProvider is a narrow optional capability interface for providers
// that can stamp a translucent rotated text watermark onto the current page.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if wp, ok := provider.(core.WatermarkProvider); ok { ... }
type WatermarkProvider interface {
	// AddWatermark draws prop.Text centered in cell, rotated by prop.Angle
	// around the cell center, with prop.Alpha opacity. The font size is
	// scaled down when the text would exceed the cell diagonal.
	AddWatermark(cell *entity.Cell, prop *props.Watermark)
}
