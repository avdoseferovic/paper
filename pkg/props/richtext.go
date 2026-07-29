package props

import (
	"cmp"
	"slices"

	"github.com/avdoseferovic/paper/pkg/consts"
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
	Text           string
	ForceBreak     bool
	Image          *RichImage
	WhiteSpace     string // per-run CSS white-space override; currently "nowrap" affects line breaking
	Family         string
	Style          fontstyle.Type
	Size           float64
	SizeScale      float64 // multiplier applied after Size/default font resolution; 0 = unchanged
	LineHeight     float64 // per-run CSS line-height multiplier for inline box painting; 0 = default metrics
	Color          *Color
	Underline      bool
	Strikethrough  bool
	Hyperlink      *string
	VerticalAlign  string // "baseline" | "sub" | "super"
	VerticalOffset float64
	Hidden         bool // preserves layout but skips painting for CSS visibility:hidden

	// LetterSpacing is extra character spacing in mm (0 = default).
	LetterSpacing float64
	// Background, when non-nil, paints a filled rectangle behind the run before
	// drawing the text. Used by HTML <mark>, <kbd>, inline <code>.
	Background *Color
	// BgRadius is the corner radius (mm) for the run background — renders the
	// background as a rounded "pill"/badge. 0 = sharp rectangle.
	BgRadius float64
	// BgPadX is extra horizontal padding (mm) added to each side of the run
	// background, so a badge/pill has breathing room around its text. 0 = none.
	BgPadX float64
	// BgPadLeft/BgPadRight override BgPadX for asymmetric inline boxes. When both
	// are zero, BgPadX is used as the backward-compatible symmetric fallback.
	BgPadLeft  float64
	BgPadRight float64
	// BgPadY is extra vertical padding (mm) added above and below the run
	// background. It affects only the painted background box; line layout is
	// controlled by RichText.LineHeight.
	BgPadY float64
	// BorderColor, when non-nil with BorderWidth > 0, paints a uniform border
	// around the same inline box used for Background/BgRadius/BgPadX/BgPadY.
	BorderColor *Color
	// BorderWidth is the uniform inline border thickness in millimetres.
	BorderWidth float64
	// InlineMarginLeft reserves horizontal space before this inline box. It is
	// layout-only and does not expand the painted background/border.
	InlineMarginLeft float64
	// InlineMarginRight reserves horizontal space after this inline box. It is
	// layout-only and does not expand the painted background/border.
	InlineMarginRight float64
	// InlineBoxID groups adjacent runs that belong to the same styled inline
	// element so backgrounds/borders are painted once around the whole element
	// even when nested spans split the text into multiple runs.
	InlineBoxID int
	// BoxShadows, when non-empty, paints CSS box-shadow style shadows behind the
	// same inline box used for Background/BgRadius/BgPadX/BgPadY.
	BoxShadows []Shadow
	// InlineBoxWidth/InlineBoxHeight reserve a fixed-size inline box for empty
	// styled elements such as checkbox markers. The dimensions are content-box
	// sizes in millimetres; border and padding are added by the inline box
	// painter just like text-backed runs.
	InlineBoxWidth  float64
	InlineBoxHeight float64
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
	Align             consts.Align
	LineHeight        float64
	BreakLineStrategy consts.BreakLineStrategy
	FirstLineIndent   float64
	WhiteSpace        string

	// AnchorResolver, when non-nil, is called with a LocalAnchor name to
	// obtain the PDF link ID for per-run internal anchor rectangles. It is set
	// at render time by richtext.RichText.Render when the provider implements
	// core.LinkProvider and the component was built with an anchor registry.
	AnchorResolver func(name string) int

	// Outline, when set, adds this paragraph as an entry in the PDF document
	// outline (the bookmark sidebar). When Title is empty, the concatenated
	// run text is used.
	Outline *Outline
}

// MakeValid fills in default values for RichText paragraph props.
func (r *RichText) MakeValid(font *Font) {
	*r = NormalizeRichText(*r, font)
}

// NormalizeRichText returns a defaulted copy of r.
func NormalizeRichText(r RichText, _ ...*Font) RichText {
	r.Align = cmp.Or(r.Align, consts.AlignLeft)
	r.LineHeight = cmp.Or(r.LineHeight, 1.0)
	r.BreakLineStrategy = cmp.Or(r.BreakLineStrategy, consts.BreakLineEmptySpace)
	r.WhiteSpace = cmp.Or(r.WhiteSpace, "normal")
	return r
}

// NormalizeRichRun returns a defaulted copy of run.
func NormalizeRichRun(run RichRun, font *Font) RichRun {
	if font != nil {
		normalizedFont := NormalizeFont(*font, "")
		run.Family = cmp.Or(run.Family, normalizedFont.Family)
		run.Style = cmp.Or(run.Style, normalizedFont.Style)
		run.Size = cmp.Or(run.Size, normalizedFont.Size)
	}
	run.Color = CloneColor(run.Color)
	run.Background = CloneColor(run.Background)
	run.BorderColor = CloneColor(run.BorderColor)
	run.BoxShadows = cloneShadows(run.BoxShadows)
	if run.Image != nil {
		image := *run.Image
		image.Bytes = slices.Clone(run.Image.Bytes)
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
