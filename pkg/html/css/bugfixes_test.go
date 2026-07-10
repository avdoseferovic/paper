package css

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestParseLength_RemResolves(t *testing.T) {
	t.Parallel()

	if got := ParseLength("1rem", 4); !almostEqual(got, 4) {
		t.Errorf("1rem with parent 4mm = %v, want 4", got)
	}
	if got := ParseLength("2rem", 0); !almostEqual(got, 2*16*mmPerPx) {
		t.Errorf("2rem with no parent = %v, want default-root fallback", got)
	}
	if got := ParseLength("2em", 4); !almostEqual(got, 8) {
		t.Errorf("2em = %v, want 8", got)
	}
}

func TestParseLength_InchUnit(t *testing.T) {
	t.Parallel()

	if got := ParseLength("1in", 0); !almostEqual(got, 25.4) {
		t.Errorf("1in = %v, want 25.4", got)
	}
}

func TestApplyFontSize_PercentAndKeywords(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.Apply("font-size", "120%", &ComputedStyle{FontSize: 10})
	if !almostEqual(s.FontSize, 12) {
		t.Errorf("120%% of 10mm = %v, want 12", s.FontSize)
	}

	s = NewComputedStyle()
	s.Apply("font-size", "larger", &ComputedStyle{FontSize: 10})
	if !almostEqual(s.FontSize, 12) {
		t.Errorf("larger of 10mm = %v, want 12", s.FontSize)
	}

	// Unparseable keyword must NOT clobber the inherited size with 0.
	s = NewComputedStyle()
	s.FontSize = 5
	s.Apply("font-size", "bogus-keyword", &ComputedStyle{FontSize: 10})
	if s.FontSize != 5 {
		t.Errorf("bogus font-size clobbered inherited size: %v", s.FontSize)
	}
}

func TestApplyLineHeight_UnitsBecomeMultipliers(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.FontSize = 16 * mmPerPx // 16px
	s.Apply("line-height", "24px", &ComputedStyle{FontSize: s.FontSize})
	if !almostEqual(s.LineHeight, 1.5) {
		t.Errorf("line-height 24px on 16px text = %v, want 1.5", s.LineHeight)
	}

	s = NewComputedStyle()
	s.FontSize = 4
	s.Apply("line-height", "150%", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.LineHeight, 1.5) {
		t.Errorf("line-height 150%% = %v, want 1.5", s.LineHeight)
	}

	s = NewComputedStyle()
	s.Apply("line-height", "1.6", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.LineHeight, 1.6) {
		t.Errorf("unitless line-height = %v, want 1.6", s.LineHeight)
	}

	s = NewComputedStyle()
	s.Apply("line-height", "normal", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.LineHeight, 1.0) {
		t.Errorf("line-height normal = %v, want 1.0", s.LineHeight)
	}
}

func TestParseColor_ModernSyntax(t *testing.T) {
	t.Parallel()

	c := ParseColor("rgb(255 0 0)")
	if c == nil || c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("rgb(255 0 0) = %+v, want red", c)
	}

	c = ParseColor("rgb(255 0 0 / 0.5)")
	if c == nil || c.R != 255 {
		t.Errorf("rgb slash alpha = %+v, want red", c)
	}
	if c != nil && !almostEqual(c.A, 0.5) {
		t.Errorf("rgb slash alpha lost alpha: %+v", c)
	}

	c = ParseColor("hsl(120deg, 100%, 25%)")
	if c == nil || c.G == 0 {
		t.Errorf("hsl with deg = %+v, want green-ish", c)
	}
}

func TestExpandShorthands_BorderColorFunctions(t *testing.T) {
	t.Parallel()

	out := ExpandShorthands(map[string]string{"border": "1px solid rgb(255, 0, 0)"})
	if got := out["border-top-color"]; got != "rgb(255, 0, 0)" {
		t.Errorf("border shorthand color = %q, want rgb(255, 0, 0)", got)
	}

	out = ExpandShorthands(map[string]string{"border": "solid red"})
	if got := out["border-top-width"]; got != "medium" {
		t.Errorf("border shorthand default width = %q, want medium", got)
	}
}

func TestBorderWidthKeywords(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.Apply("border-top-width", "medium", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.BorderTopWidth, 3*mmPerPx) {
		t.Errorf("border-width medium = %v, want 3px", s.BorderTopWidth)
	}
	s.Apply("border-bottom-width", "thin", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.BorderBottomWidth, mmPerPx) {
		t.Errorf("border-width thin = %v, want 1px", s.BorderBottomWidth)
	}
}

func TestBorderWidthEmUsesFontSize(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.FontSize = 4
	s.Apply("border-top-width", "1em", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.BorderTopWidth, 4) {
		t.Errorf("border-width 1em = %v, want 4", s.BorderTopWidth)
	}
}

func TestHeightEmUsesFontSize(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.FontSize = 4
	s.Apply("height", "2em", &ComputedStyle{FontSize: 4})
	if !almostEqual(s.Height, 8) {
		t.Errorf("height 2em = %v, want 8", s.Height)
	}
}

func TestExpandShorthands_MarginTooManyValuesIgnored(t *testing.T) {
	t.Parallel()

	out := ExpandShorthands(map[string]string{"margin": "1px 2px 3px 4px 5px"})
	if _, ok := out["margin-top"]; ok {
		t.Errorf("invalid margin shorthand should be dropped, got %v", out)
	}
}

func TestExpandShorthands_FontWithLineHeight(t *testing.T) {
	t.Parallel()

	out := ExpandShorthands(map[string]string{"font": "italic bold 12px/1.5 Arial"})
	if got := out["font-size"]; got != "12px" {
		t.Errorf("font shorthand size = %q, want 12px", got)
	}
	if got := out["line-height"]; got != "1.5" {
		t.Errorf("font shorthand line-height = %q, want 1.5", got)
	}
	if got := out["font-weight"]; got != "bold" {
		t.Errorf("font shorthand weight = %q, want bold", got)
	}
	if got := out["font-style"]; got != "italic" {
		t.Errorf("font shorthand style = %q, want italic", got)
	}
}
