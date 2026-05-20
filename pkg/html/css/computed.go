package css

import (
	"strconv"
	"strings"
)

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

	// Border radius (mm). BorderRadius is the uniform fallback; per-corner overrides it.
	BorderRadius            float64
	BorderRadiusTopLeft     float64
	BorderRadiusTopRight    float64
	BorderRadiusBottomLeft  float64
	BorderRadiusBottomRight float64

	// Layout
	Display string // "block" | "inline" | "inline-block" | "none" | "flex" | "table" | ...
	Width   float64
	Height  float64

	// Flex container properties
	FlexDirection  string  // "row" | "column" | "row-reverse" | "column-reverse"
	JustifyContent string  // "flex-start" | "center" | "flex-end" | "space-between" | "space-around"
	AlignItems     string  // "flex-start" | "center" | "flex-end" | "stretch"
	RowGap         float64 // mm
	ColumnGap      float64 // mm

	// Flex item properties
	FlexGrow      float64 // default 0; used as proportional weight in layout
	FlexShrink    float64 // parsed/stored; no independent layout effect in v1 (quantizer prevents overflow)
	FlexBasis     float64 // mm; 0 means auto unless FlexBasisAuto or FlexBasisPct set
	FlexBasisAuto bool    // true when flex-basis:auto was explicitly set
	FlexBasisPct  float64 // >0 when flex-basis was a percentage (0–100 scale)

	// List marker style (for ul/ol). Supports standard CSS values plus the
	// "decimal-circle" extension that renders numbers inside filled discs.
	ListStyleType string

	// Opacity multiplies into all descendant color alpha values (0 = invisible, 1 = opaque).
	Opacity float64

	unsupportedHandler func(prop, val string)
}

// NewComputedStyle returns a ComputedStyle with sensible zero-value defaults.
// Display defaults to "" (unset) — callers should treat "" the same as "block".
func NewComputedStyle() *ComputedStyle {
	return &ComputedStyle{
		TextAlign:  "left",
		FontWeight: "normal",
		FontStyle:  "normal",
		Display:    "",
		LineHeight: 1.0,
		Opacity:    1.0,
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
		s.Color = ParseColor(val)
	case "background-color":
		s.BackgroundColor = ParseColor(val)
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
		s.BorderTopColor = ParseColor(val)
	case "border-right-color":
		s.BorderRightColor = ParseColor(val)
	case "border-bottom-color":
		s.BorderBottomColor = ParseColor(val)
	case "border-left-color":
		s.BorderLeftColor = ParseColor(val)
	case "border-color":
		c := ParseColor(val)
		s.BorderTopColor, s.BorderRightColor, s.BorderBottomColor, s.BorderLeftColor = c, c, c, c
	case "border-width":
		w := ParseLength(val, 0)
		s.BorderTopWidth, s.BorderRightWidth, s.BorderBottomWidth, s.BorderLeftWidth = w, w, w, w
	case "border-style":
		s.BorderTopStyle, s.BorderRightStyle, s.BorderBottomStyle, s.BorderLeftStyle = val, val, val, val
	case "display":
		if val == "inline-flex" {
			s.Display = "flex"
		} else {
			s.Display = val
		}
	case "flex-direction":
		s.FlexDirection = val
	case "justify-content":
		s.JustifyContent = val
	case "align-items":
		s.AlignItems = val
	case "flex-grow":
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			s.FlexGrow = v
		}
	case "flex-shrink":
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			s.FlexShrink = v
		}
	case "flex-basis":
		switch val {
		case "auto":
			s.FlexBasisAuto = true
			s.FlexBasis = 0
			s.FlexBasisPct = 0
		default:
			if pct, ok := ParsePercentage(val); ok {
				s.FlexBasisPct = pct * 100
				s.FlexBasis = 0
				s.FlexBasisAuto = false
			} else {
				s.FlexBasis = ParseLength(val, parentFontSize)
				s.FlexBasisAuto = false
				s.FlexBasisPct = 0
			}
		}
	case "gap":
		parts := strings.Fields(val)
		if len(parts) == 1 {
			v := ParseLength(parts[0], parentFontSize)
			s.RowGap = v
			s.ColumnGap = v
		} else if len(parts) >= 2 {
			s.RowGap = ParseLength(parts[0], parentFontSize)
			s.ColumnGap = ParseLength(parts[1], parentFontSize)
		}
	case "row-gap":
		s.RowGap = ParseLength(val, parentFontSize)
	case "column-gap":
		s.ColumnGap = ParseLength(val, parentFontSize)
	case "width":
		s.Width = ParseLength(val, 0)
	case "height":
		s.Height = ParseLength(val, 0)
	case "border-radius":
		s.BorderRadius = ParseLength(val, 0)
	case "border-top-left-radius":
		s.BorderRadiusTopLeft = ParseLength(val, 0)
	case "border-top-right-radius":
		s.BorderRadiusTopRight = ParseLength(val, 0)
	case "border-bottom-left-radius":
		s.BorderRadiusBottomLeft = ParseLength(val, 0)
	case "border-bottom-right-radius":
		s.BorderRadiusBottomRight = ParseLength(val, 0)
	case "opacity":
		v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(val), "%"), 64)
		if err == nil {
			if strings.HasSuffix(strings.TrimSpace(val), "%") {
				v /= 100.0
			}
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			s.Opacity = v
		}
	case "list-style-type":
		s.ListStyleType = strings.TrimSpace(val)
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
