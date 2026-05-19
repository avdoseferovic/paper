package css

import (
	"strconv"
	"strings"
)

// RGBColor holds a parsed CSS color.
type RGBColor struct {
	R, G, B int
}

// ComputedStyle holds the resolved CSS property values for a DOM element.
type ComputedStyle struct {
	// Font
	FontFamily string
	FontSize   float64 // mm
	FontWeight string  // "normal" | "bold"
	FontStyle  string  // "normal" | "italic"

	// Text
	Color          *RGBColor
	TextAlign      string // "left" | "center" | "right" | "justify"
	TextDecoration string // "none" | "underline" | "line-through"
	LineHeight     float64

	// Box model (mm)
	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64

	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64

	// Border (mm / style / color)
	BorderTopWidth    float64
	BorderRightWidth  float64
	BorderBottomWidth float64
	BorderLeftWidth   float64

	BorderTopStyle    string
	BorderRightStyle  string
	BorderBottomStyle string
	BorderLeftStyle   string

	BorderTopColor    *RGBColor
	BorderRightColor  *RGBColor
	BorderBottomColor *RGBColor
	BorderLeftColor   *RGBColor

	// Background
	BackgroundColor *RGBColor

	// Layout
	Display string // "block" | "inline" | "inline-block" | "none" | "table" | ...
	Width   float64
	Height  float64

	unsupportedHandler func(prop, val string)
}

// NewComputedStyle returns a ComputedStyle with sensible zero-value defaults.
func NewComputedStyle() *ComputedStyle {
	return &ComputedStyle{
		TextAlign:  "left",
		FontWeight: "normal",
		FontStyle:  "normal",
		Display:    "none",
		LineHeight: 1.0,
	}
}

// SetUnsupportedHandler registers a callback invoked for unrecognised CSS properties.
func (s *ComputedStyle) SetUnsupportedHandler(fn func(prop, val string)) {
	s.unsupportedHandler = fn
}

// Apply sets a single CSS property. Parent is used for em resolution.
func (s *ComputedStyle) Apply(prop, val string, parent *ComputedStyle) {
	parentFontSize := 0.0
	if parent != nil {
		parentFontSize = parent.FontSize
	}

	switch prop {
	case "color":
		s.Color = parseColor(val)
	case "background-color":
		s.BackgroundColor = parseColor(val)
	case "font-family":
		s.FontFamily = strings.Trim(val, `'"`)
	case "font-size":
		s.FontSize = ParseLength(val, parentFontSize)
	case "font-weight":
		s.FontWeight = normFontWeight(val)
	case "font-style":
		s.FontStyle = val
	case "text-align":
		s.TextAlign = val
	case "text-decoration":
		s.TextDecoration = val
	case "line-height":
		s.LineHeight = ParseLength(val, s.FontSize)
	case "padding-top":
		s.PaddingTop = ParseLength(val, s.FontSize)
	case "padding-right":
		s.PaddingRight = ParseLength(val, s.FontSize)
	case "padding-bottom":
		s.PaddingBottom = ParseLength(val, s.FontSize)
	case "padding-left":
		s.PaddingLeft = ParseLength(val, s.FontSize)
	case "margin-top":
		s.MarginTop = ParseLength(val, s.FontSize)
	case "margin-right":
		s.MarginRight = ParseLength(val, s.FontSize)
	case "margin-bottom":
		s.MarginBottom = ParseLength(val, s.FontSize)
	case "margin-left":
		s.MarginLeft = ParseLength(val, s.FontSize)
	case "border-top-width":
		s.BorderTopWidth = ParseLength(val, 0)
	case "border-right-width":
		s.BorderRightWidth = ParseLength(val, 0)
	case "border-bottom-width":
		s.BorderBottomWidth = ParseLength(val, 0)
	case "border-left-width":
		s.BorderLeftWidth = ParseLength(val, 0)
	case "border-top-style":
		s.BorderTopStyle = val
	case "border-right-style":
		s.BorderRightStyle = val
	case "border-bottom-style":
		s.BorderBottomStyle = val
	case "border-left-style":
		s.BorderLeftStyle = val
	case "border-top-color":
		s.BorderTopColor = parseColor(val)
	case "border-right-color":
		s.BorderRightColor = parseColor(val)
	case "border-bottom-color":
		s.BorderBottomColor = parseColor(val)
	case "border-left-color":
		s.BorderLeftColor = parseColor(val)
	case "border-color":
		c := parseColor(val)
		s.BorderTopColor, s.BorderRightColor, s.BorderBottomColor, s.BorderLeftColor = c, c, c, c
	case "border-width":
		w := ParseLength(val, 0)
		s.BorderTopWidth, s.BorderRightWidth, s.BorderBottomWidth, s.BorderLeftWidth = w, w, w, w
	case "border-style":
		s.BorderTopStyle, s.BorderRightStyle, s.BorderBottomStyle, s.BorderLeftStyle = val, val, val, val
	case "display":
		s.Display = val
	case "width":
		s.Width = ParseLength(val, 0)
	case "height":
		s.Height = ParseLength(val, 0)
	case "vertical-align":
		// stored implicitly via usage context; no field needed yet
	default:
		if s.unsupportedHandler != nil {
			s.unsupportedHandler(prop, val)
		}
	}
}

func normFontWeight(val string) string {
	switch val {
	case "bold", "bolder", "700", "800", "900":
		return "bold"
	default:
		return "normal"
	}
}

// parseColor parses common CSS color formats: #rgb, #rrggbb, named colors.
func parseColor(val string) *RGBColor {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "#") {
		hex := val[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 {
			r, _ := strconv.ParseInt(hex[0:2], 16, 32)
			g, _ := strconv.ParseInt(hex[2:4], 16, 32)
			b, _ := strconv.ParseInt(hex[4:6], 16, 32)
			return &RGBColor{R: int(r), G: int(g), B: int(b)}
		}
	}
	if c, ok := namedColors[val]; ok {
		return &c
	}
	return nil
}

// namedColors maps common CSS color names to RGB values.
var namedColors = map[string]RGBColor{
	"black":   {0, 0, 0},
	"white":   {255, 255, 255},
	"red":     {255, 0, 0},
	"green":   {0, 128, 0},
	"blue":    {0, 0, 255},
	"yellow":  {255, 255, 0},
	"orange":  {255, 165, 0},
	"gray":    {128, 128, 128},
	"grey":    {128, 128, 128},
	"silver":  {192, 192, 192},
	"navy":    {0, 0, 128},
	"maroon":  {128, 0, 0},
	"purple":  {128, 0, 128},
	"teal":    {0, 128, 128},
	"fuchsia": {255, 0, 255},
	"aqua":    {0, 255, 255},
	"lime":    {0, 255, 0},
}
