package props

// The 2D CSS transform functions a TransformOp can hold.
const (
	TransformRotate    = "rotate"
	TransformScale     = "scale"
	TransformTranslate = "translate"
	TransformSkew      = "skew"
	TransformSkewX     = "skewX"
	TransformSkewY     = "skewY"
)

// TransformOp represents one parsed 2D CSS transform function.
//
// Values uses the first two slots for all supported operations:
// rotate/skew angles are degrees, scale factors are unitless, and translate
// values are millimetres in Paper's normal coordinate system.
type TransformOp struct {
	Type   string
	Values [2]float64
}
