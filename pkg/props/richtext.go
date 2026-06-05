package props

import (
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/breakline"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
)

// RichRun is a single styled segment within a RichText paragraph.
// It deliberately has no image field — inline images are split into separate rows by the HTML translator.
type RichRun struct {
	Text          string
	Family        string
	Style         fontstyle.Type
	Size          float64
	Color         *Color
	Underline     bool
	Strikethrough bool
	Hyperlink     *string
	VerticalAlign string // "baseline" | "sub" | "super"

	// LetterSpacing is extra character spacing in mm (0 = default).
	LetterSpacing float64
	// Background, when non-nil, paints a filled rectangle behind the run before
	// drawing the text. Used by HTML <mark>, <kbd>, inline <code>.
	Background *Color
	// LocalAnchor, when non-empty, makes the run an internal PDF link target
	// to the named destination (registered via id="…" on a block element).
	// Takes precedence over Hyperlink.
	LocalAnchor string

	// TextShadow, when non-nil, draws a shadow behind the run text. Only the
	// first shadow is rendered (CSS multi-shadow on text is a known limitation).
	TextShadow *Shadow
}

// RichText holds paragraph-level properties for a RichText component.
type RichText struct {
	Top               float64
	Bottom            float64
	Left              float64
	Right             float64
	Align             align.Type
	LineHeight        float64
	BreakLineStrategy breakline.Strategy
	FirstLineIndent   float64
	WhiteSpace        string

	// AnchorResolver, when non-nil, is called with a LocalAnchor name to
	// obtain the PDF link ID for per-run internal anchor rectangles. It is set
	// at render time by richtext.RichText.Render when the provider implements
	// core.LinkProvider and the component was built with an anchor registry.
	AnchorResolver func(name string) int
}

// MakeValid fills in default values for RichText paragraph props.
func (r *RichText) MakeValid(font *Font) {
	if r.Align == "" {
		r.Align = align.Left
	}
	if r.LineHeight == 0 {
		r.LineHeight = 1.0
	}
	if r.BreakLineStrategy == "" {
		r.BreakLineStrategy = breakline.EmptySpaceStrategy
	}
	if r.WhiteSpace == "" {
		r.WhiteSpace = "normal"
	}
}
