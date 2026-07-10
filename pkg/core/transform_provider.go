package core

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// TransformProvider is an optional capability for providers that can wrap
// rendering in a temporary 2D transform context. Coordinates and translate
// distances are margin-relative millimetres, matching entity.Cell.
type TransformProvider interface {
	// WithTransform applies ops around the origin point relative to cell's
	// top-left corner, renders fn, then restores the provider transform state.
	WithTransform(cell *entity.Cell, originX, originY float64, ops []props.TransformOp, fn func())
}
