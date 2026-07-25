// Package htmllimits holds the size limits applied when rendering HTML, such as
// maximum image and stylesheet sizes. Limits guard against inputs large enough
// to exhaust memory.
package htmllimits

import "errors"

const disabled = -1

// Errors reported when HTML input goes past one of the configured limits.
var (
	ErrImageTooLarge      = errors.New("html: image exceeds configured limits")
	ErrDOMTooDeep         = errors.New("html: DOM exceeds configured depth limit")
	ErrDOMTooLarge        = errors.New("html: DOM exceeds configured node limit")
	ErrSVGTooLarge        = errors.New("html: SVG raster dimensions exceed configured pixel limit")
	ErrStyleRulesTooLarge = errors.New("html: stylesheet exceeds configured rule limit")
)

// Limits caps how much work one HTML document may cost. A zero field means
// "use the default"; a negative field turns that limit off.
type Limits struct {
	MaxImagePixels int64
	MaxImageBytes  int64
	MaxDOMDepth    int
	MaxDOMNodes    int
	MaxSVGPixels   int64
	MaxStyleRules  int
}

// Default returns the limits used when the caller does not set any.
func Default() Limits {
	return Limits{
		MaxImagePixels: 50_000_000,
		MaxImageBytes:  32 << 20,
		MaxDOMDepth:    256,
		MaxDOMNodes:    200_000,
		MaxSVGPixels:   50_000_000,
		MaxStyleRules:  50_000,
	}
}

// NoLimits returns limits with every check turned off.
func NoLimits() Limits {
	return Limits{
		MaxImagePixels: disabled,
		MaxImageBytes:  disabled,
		MaxDOMDepth:    disabled,
		MaxDOMNodes:    disabled,
		MaxSVGPixels:   disabled,
		MaxStyleRules:  disabled,
	}
}

// Normalize fills in each unset (zero) field of l with the default for that
// limit and returns the result.
func Normalize(l Limits) Limits {
	d := Default()
	if l.MaxImagePixels != 0 {
		d.MaxImagePixels = l.MaxImagePixels
	}
	if l.MaxImageBytes != 0 {
		d.MaxImageBytes = l.MaxImageBytes
	}
	if l.MaxDOMDepth != 0 {
		d.MaxDOMDepth = l.MaxDOMDepth
	}
	if l.MaxDOMNodes != 0 {
		d.MaxDOMNodes = l.MaxDOMNodes
	}
	if l.MaxSVGPixels != 0 {
		d.MaxSVGPixels = l.MaxSVGPixels
	}
	if l.MaxStyleRules != 0 {
		d.MaxStyleRules = l.MaxStyleRules
	}
	return d
}

// Int64Exceeded reports whether value is over limit. A limit of zero or less is
// treated as no limit.
func Int64Exceeded(limit, value int64) bool {
	return limit > 0 && value > limit
}

// IntExceeded reports whether value is over limit. A limit of zero or less is
// treated as no limit.
func IntExceeded(limit, value int) bool {
	return limit > 0 && value > limit
}
