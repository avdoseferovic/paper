package props

import (
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/breakline"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
)

// RichImage is an inline image embedded in a RichText paragraph.
// Width and Height are measured in millimetres.
type RichImage struct {
	Bytes          []byte
	Extension      extension.Type
	Width          float64
	Height         float64
	Alt            string
	ObjectFit      string
	ObjectPosition string
}

// RichRun is a single styled segment within a RichText paragraph.
type RichRun struct {
	Text          string
	Image         *RichImage
	Family        string
	Style         fontstyle.Type
	Size          float64
	SizeScale     float64 // multiplier applied after Size/default font resolution; 0 = unchanged
	Color         *Color
	Underline     bool
	Strikethrough bool
	Hyperlink     *string
	VerticalAlign string // "baseline" | "sub" | "super"
	Hidden        bool   // preserves layout but skips painting for CSS visibility:hidden

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
	// first shadow is stored here for compatibility. TextShadows stores the full
	// CSS comma-separated list when available.
	TextShadow *Shadow
	// TextShadows, when non-empty, draws every shadow behind the run text in
	// order before the normal text is painted.
	TextShadows []Shadow
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
	*r = NormalizeRichText(*r, font)
}

// NormalizeRichText returns a defaulted copy of r.
func NormalizeRichText(r RichText, _ ...*Font) RichText {
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
	return r
}

// NormalizeRichRun returns a defaulted copy of run.
func NormalizeRichRun(run RichRun, font *Font) RichRun {
	if font != nil {
		normalizedFont := NormalizeFont(*font, "")
		if run.Family == "" {
			run.Family = normalizedFont.Family
		}
		if run.Style == "" {
			run.Style = normalizedFont.Style
		}
		if run.Size == 0 {
			run.Size = normalizedFont.Size
		}
	}
	run.Color = CloneColor(run.Color)
	run.Background = CloneColor(run.Background)
	if run.Image != nil {
		image := *run.Image
		image.Bytes = append([]byte(nil), run.Image.Bytes...)
		run.Image = &image
	}
	if run.Hyperlink != nil {
		hyperlink := *run.Hyperlink
		run.Hyperlink = &hyperlink
	}
	if run.TextShadow != nil {
		shadow := CloneShadow(*run.TextShadow)
		run.TextShadow = &shadow
	}
	if run.TextShadows != nil {
		shadows := make([]Shadow, len(run.TextShadows))
		for i, shadow := range run.TextShadows {
			shadows[i] = CloneShadow(shadow)
		}
		run.TextShadows = shadows
	}
	return run
}

// CloneRichRuns returns an independent copy of runs.
func CloneRichRuns(runs []RichRun) []RichRun {
	if runs == nil {
		return nil
	}
	clone := make([]RichRun, len(runs))
	for i, run := range runs {
		clone[i] = NormalizeRichRun(run, nil)
	}
	return clone
}
