package htmllimits

import "errors"

const disabled = -1

var (
	ErrImageTooLarge      = errors.New("html: image exceeds configured limits")
	ErrDOMTooDeep         = errors.New("html: DOM exceeds configured depth limit")
	ErrDOMTooLarge        = errors.New("html: DOM exceeds configured node limit")
	ErrSVGTooLarge        = errors.New("html: SVG raster dimensions exceed configured pixel limit")
	ErrStyleRulesTooLarge = errors.New("html: stylesheet exceeds configured rule limit")
)

type Limits struct {
	MaxImagePixels int64
	MaxImageBytes  int64
	MaxDOMDepth    int
	MaxDOMNodes    int
	MaxSVGPixels   int64
	MaxStyleRules  int
}

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

func Int64Exceeded(limit, value int64) bool {
	return limit > 0 && value > limit
}

func IntExceeded(limit, value int) bool {
	return limit > 0 && value > limit
}
