package css

import (
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
	BackgroundColor    *RGBColor
	BackgroundGradient *Gradient // set when background-image is a gradient
	BackgroundImageURL string    // set when background-image is url(...)
	BackgroundSize     string
	BackgroundPosition string
	BackgroundRepeat   string
	BoxShadow          []Shadow // parsed box-shadow value (up to 4)
	TextShadow         *Shadow  // text-shadow compatibility field; points at first TextShadows entry
	TextShadows        []Shadow // parsed text-shadow value (up to 4)

	// Outline (mm / style / color / offset) — drawn outside the cell box.
	OutlineWidth  float64
	OutlineStyle  string // "solid" | "dashed" | "dotted"
	OutlineColor  *RGBColor
	OutlineOffset float64 // mm; positive pushes outline further out

	// Border radius (mm). BorderRadius is the uniform fallback; per-corner overrides it.
	BorderRadius            float64
	BorderRadiusTopLeft     float64
	BorderRadiusTopRight    float64
	BorderRadiusBottomLeft  float64
	BorderRadiusBottomRight float64

	// Layout
	Display        string // "block" | "inline" | "inline-block" | "none" | "flex" | "table" | ...
	Width          float64
	Height         float64
	MinWidth       float64
	MaxWidth       float64
	MinHeight      float64
	MaxHeight      float64
	ObjectFit      string
	ObjectPosition string

	// Flex container properties
	FlexDirection  string // "row" | "column" | "row-reverse" | "column-reverse"
	FlexWrap       string // "nowrap" | "wrap" | "wrap-reverse"
	JustifyContent string // "flex-start" | "center" | "flex-end" | "space-between" | "space-around"
	AlignItems     string // "flex-start" | "center" | "flex-end" | "stretch"

	// Flex item properties (cross-axis)
	AlignSelf string  // "auto" | "flex-start" | "flex-end" | "center" | "stretch"
	RowGap    float64 // mm
	ColumnGap float64 // mm

	// Flex item properties
	FlexGrow      float64 // default 0; used as proportional weight in layout
	FlexShrink    float64 // parsed/stored; no independent layout effect (quantizer prevents overflow)
	FlexBasis     float64 // mm; 0 means auto unless FlexBasisAuto or FlexBasisPct set
	FlexBasisAuto bool    // true when flex-basis:auto was explicitly set
	FlexBasisPct  float64 // >0 when flex-basis was a percentage (0–100 scale)
	Order         int     // CSS order property; lower = earlier; 0 is default

	// List marker style (for ul/ol). Supports standard CSS values plus the
	// "decimal-circle" extension that renders numbers inside filled discs.
	ListStyleType string

	// Opacity multiplies into all descendant color alpha values (0 = invisible, 1 = opaque).
	Opacity float64

	// Typography
	LetterSpacing float64 // mm; applied via SetCharSpacing
	TextTransform string  // "none" | "uppercase" | "lowercase" | "capitalize"
	TextIndent    float64 // mm; first-line indent
	WhiteSpace    string  // "normal" | "nowrap" | "pre" | "pre-wrap" | "pre-line"

	// Page break hints.
	PageBreakBefore string // "always" | "avoid" | "auto"
	PageBreakAfter  string // "always" | "avoid" | "auto"
	BreakInside     string // "avoid" | "auto"

	// CSS custom properties (--name: value). Stored as a flat map per element;
	// cascade inheritance is handled by callers (computeNodeStyle) which copy
	// from the parent's Vars map before applying child rules.
	Vars map[string]string

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
// CSS custom properties (--*) are stored on s.Vars; standard properties get
// var(...) references resolved against s.Vars (inheriting parent's vars
// implicitly via computeNodeStyle's pre-population).
// Apply resolves val for prop and stores the result. Callers that know the
// parent content width should use ApplyCtx so that calc() and % resolve
// correctly against the context width.
func (s *ComputedStyle) Apply(prop, val string, parent *ComputedStyle) {
	s.ApplyCtx(prop, val, parent, 0)
}

// ApplyCtx is Apply with an explicit context width (parent content width in mm).
// Width-relative properties (width, padding, margin, text-indent) resolve
// percentages and calc(…%) against ctxWidth.
func (s *ComputedStyle) ApplyCtx(prop, val string, parent *ComputedStyle, ctxWidth float64) {
	parentFontSize := 0.0
	if parent != nil {
		parentFontSize = parent.FontSize
	}

	// CSS custom property declaration: store without further processing.
	if strings.HasPrefix(prop, "--") {
		if s.Vars == nil {
			s.Vars = map[string]string{}
		}
		s.Vars[prop] = val
		return
	}

	// Resolve any var() references in val before dispatching to handlers.
	if strings.Contains(val, "var(") {
		val = ResolveVars(val, s.Vars)
	}

	ctx := computedPropertyContext{
		prop:           prop,
		val:            val,
		parentFontSize: parentFontSize,
		ctxWidth:       ctxWidth,
	}
	if s.applyEffectsProperty(ctx) ||
		s.applyFontProperty(ctx) ||
		s.applyBoxProperty(ctx) ||
		s.applyBorderProperty(ctx) ||
		s.applyFlexProperty(ctx) ||
		s.applyTypographyProperty(ctx) {
		return
	}

	if s.unsupportedHandler != nil {
		s.unsupportedHandler(prop, val)
	}
}

type computedPropertyContext struct {
	prop           string
	val            string
	parentFontSize float64
	ctxWidth       float64
}
