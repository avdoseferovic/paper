package consts

// DefaultLineThickness is the default thickness, in mm, applied to lines and
// borders when no explicit thickness is set.
const DefaultLineThickness float64 = 0.2

// LineStyle is the representation of a line style.
type LineStyle string

const (
	// LineStyleSolid represents a continuous line.
	LineStyleSolid LineStyle = "solid"
	// LineStyleDashed represents a dashed line.
	LineStyleDashed LineStyle = "dashed"
	// LineStyleDotted represents a dotted line.
	LineStyleDotted LineStyle = "dotted"
)
