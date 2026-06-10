package props

import (
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
)

// Watermark default values applied by NormalizeWatermark.
const (
	DefaultWatermarkSize  = 48.0
	DefaultWatermarkAlpha = 0.12
	DefaultWatermarkAngle = 45.0
)

// Watermark configures the translucent diagonal text stamped on every page.
// The watermark is drawn under the page content (right after the background)
// so it never obscures text.
type Watermark struct {
	// Text is the watermark content, e.g. "DRAFT". Set via
	// config.WithWatermark.
	Text string
	// Family of the watermark font, ex: consts.FontFamilyArial.
	Family string
	// Style of the watermark font, ex: fontstyle.Bold.
	Style fontstyle.Type
	// Size is the font size in points (default 48). When the rendered text
	// would exceed the page diagonal, the size is scaled down to fit.
	Size float64
	// Color of the watermark text (default black; the low alpha keeps it
	// subtle).
	Color *Color
	// Alpha is the opacity in [0, 1] (default 0.12).
	Alpha float64
	// Angle is the counter-clockwise rotation in degrees around the page
	// center (default 45).
	Angle float64
}

// ToMap appends watermark attributes to a config attribute map.
func (w *Watermark) ToMap(m map[string]any) map[string]any {
	if m == nil {
		m = make(map[string]any)
	}
	m["config_watermark"] = w.Text
	return m
}

// NormalizeWatermark returns a defaulted, clamped copy of w.
func NormalizeWatermark(w Watermark) Watermark {
	if w.Size <= 0 {
		w.Size = DefaultWatermarkSize
	}
	if w.Alpha <= 0 {
		w.Alpha = DefaultWatermarkAlpha
	}
	if w.Alpha > 1 {
		w.Alpha = 1
	}
	if w.Angle == 0 {
		w.Angle = DefaultWatermarkAngle
	}
	w.Color = CloneColor(w.Color)
	return w
}

// CloneWatermark returns a deep copy of w, or nil when w is nil.
func CloneWatermark(w *Watermark) *Watermark {
	if w == nil {
		return nil
	}
	clone := *w
	clone.Color = CloneColor(w.Color)
	return &clone
}
