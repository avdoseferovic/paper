package props

import "fmt"

var (
	// WhiteColor is a Color with all values in 255.
	WhiteColor = Color{Red: 255, Green: 255, Blue: 255}
	// BlackColor is a Color with all values in 0.
	BlackColor = Color{Red: 0, Green: 0, Blue: 0}
	// RedColor is a Color with only Red in 255.
	RedColor = Color{Red: 255, Green: 0, Blue: 0}
	// GreenColor is a Color with only Green in 255.
	GreenColor = Color{Red: 0, Green: 255, Blue: 0}
	// BlueColor is a Color with only Blue in 255.
	BlueColor = Color{Red: 0, Green: 0, Blue: 255}
)

// White returns an independent white Color value.
func White() Color {
	return WhiteColor
}

// Black returns an independent black Color value.
func Black() Color {
	return BlackColor
}

// Red returns an independent red Color value.
func Red() Color {
	return RedColor
}

// Green returns an independent green Color value.
func Green() Color {
	return GreenColor
}

// Blue returns an independent blue Color value.
func Blue() Color {
	return BlueColor
}

// CloneColor returns an independent copy of c.
func CloneColor(c *Color) *Color {
	if c == nil {
		return nil
	}
	clone := *c
	if c.Alpha != nil {
		alpha := *c.Alpha
		clone.Alpha = &alpha
	}
	return &clone
}

// Color represents a color in the RGB (Red, Green, Blue) space,
// is possible mix values, when all values are 0 the result color is black
// when all values are 255 the result color is white.
//
// Alpha controls translucency on render paths that honor it (gofpdf SetAlpha
// via core.AlphaProvider). It is a pointer so existing zero-value literals
// (Color{Red,Green,Blue}) remain opaque and byte-identical to prior output.
// nil = opaque; non-nil values in [0, 1] activate the alpha pipeline.
// Render paths that read this field:
//   - cellwriter/fillcolorstyler.go (background fills)
//   - cellwriter/bordercolorstyler.go (border strokes)
//   - cellwriter/borderradius.go (rounded fill+stroke)
//   - components/richtext (text color)
type Color struct {
	// Red is the amount of red
	Red int
	// Green is the amount of red
	Green int
	// Blue is the amount of red
	Blue int
	// Alpha is the translucency in [0, 1]; nil = fully opaque (default).
	Alpha *float64
}

// ToString returns a string representation of the Color, including Alpha
// when it has been set (non-nil pointer). Opaque colours use the legacy
// RGB(...) form for backward compatibility with snapshot tests.
func (c *Color) ToString() string {
	if c == nil {
		return ""
	}
	if c.Alpha != nil {
		return fmt.Sprintf("RGBA(%d, %d, %d, %.3f)", c.Red, c.Green, c.Blue, *c.Alpha)
	}
	return fmt.Sprintf("RGB(%d, %d, %d)", c.Red, c.Green, c.Blue)
}
