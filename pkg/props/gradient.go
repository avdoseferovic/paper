package props

// GradientKind distinguishes supported CSS gradient families.
type GradientKind int

const (
	GradientLinear GradientKind = iota
	GradientRadial
	GradientConic
)

// GradientStop is a color + position (0.0-1.0) in a gradient.
type GradientStop struct {
	Color    Color
	Position float64
}

// Gradient holds the resolved gradient parameters ready for the renderer.
type Gradient struct {
	Kind     GradientKind
	AngleDeg float64        // linear angle or conic starting angle
	Circle   bool           // radial only
	CX, CY   float64        // radial/conic centre (0–1 fractions)
	Stops    []GradientStop // at least 2, positions in [0,1]
}
