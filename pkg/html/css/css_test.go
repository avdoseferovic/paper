package css_test

import (
	"testing"

	"github.com/johnfercher/maroto/v2/pkg/html/css"
	"github.com/stretchr/testify/assert"
)

// ── ComputedStyle ─────────────────────────────────────────────────────────────

func TestComputedStyle_Defaults(t *testing.T) {
	t.Parallel()
	s := css.NewComputedStyle()
	assert.Equal(t, "left", s.TextAlign)
	assert.Equal(t, "none", s.Display)
	assert.Equal(t, 0.0, s.FontSize)
}

func TestComputedStyle_ApplyProperty(t *testing.T) {
	t.Parallel()

	t.Run("color", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("color", "#ff0000", nil)
		assert.Equal(t, 255, s.Color.R)
		assert.Equal(t, 0, s.Color.G)
	})

	t.Run("font-size px", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("font-size", "16px", nil)
		assert.InDelta(t, 16*0.264583, s.FontSize, 0.01) // px→mm
	})

	t.Run("font-size pt", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("font-size", "12pt", nil)
		assert.InDelta(t, 12*0.352778, s.FontSize, 0.01)
	})

	t.Run("font-size mm", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("font-size", "5mm", nil)
		assert.InDelta(t, 5.0, s.FontSize, 0.01)
	})

	t.Run("font-size em uses parent font size", func(t *testing.T) {
		t.Parallel()
		parent := css.NewComputedStyle()
		parent.FontSize = 10.0 // 10mm parent
		s := css.NewComputedStyle()
		s.Apply("font-size", "1.5em", parent)
		assert.InDelta(t, 15.0, s.FontSize, 0.01)
	})

	t.Run("padding-left mm", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("padding-left", "5mm", nil)
		assert.InDelta(t, 5.0, s.PaddingLeft, 0.01)
	})

	t.Run("display none", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("display", "none", nil)
		assert.Equal(t, "none", s.Display)
	})

	t.Run("unsupported property is silently ignored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		count := 0
		s.SetUnsupportedHandler(func(_, _ string) { count++ })
		s.Apply("box-shadow", "0 0 5px rgba(0,0,0,.2)", nil)
		assert.Equal(t, 1, count)
	})
}

// ── Shorthand expansion ───────────────────────────────────────────────────────

func TestExpandShorthands(t *testing.T) {
	t.Parallel()

	t.Run("border expands to 8 longhands", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"border": "1px solid red"})
		assert.Equal(t, "1px", decls["border-top-width"])
		assert.Equal(t, "solid", decls["border-top-style"])
		assert.Equal(t, "red", decls["border-top-color"])
		assert.Equal(t, "1px", decls["border-right-width"])
		assert.Equal(t, "1px", decls["border-bottom-width"])
		assert.Equal(t, "1px", decls["border-left-width"])
	})

	t.Run("padding 2-value expands correctly", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"padding": "5px 10px"})
		assert.Equal(t, "5px", decls["padding-top"])
		assert.Equal(t, "10px", decls["padding-right"])
		assert.Equal(t, "5px", decls["padding-bottom"])
		assert.Equal(t, "10px", decls["padding-left"])
	})

	t.Run("padding 4-value expands correctly", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"padding": "1px 2px 3px 4px"})
		assert.Equal(t, "1px", decls["padding-top"])
		assert.Equal(t, "2px", decls["padding-right"])
		assert.Equal(t, "3px", decls["padding-bottom"])
		assert.Equal(t, "4px", decls["padding-left"])
	})

	t.Run("margin expands same as padding", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"margin": "8mm"})
		assert.Equal(t, "8mm", decls["margin-top"])
		assert.Equal(t, "8mm", decls["margin-right"])
		assert.Equal(t, "8mm", decls["margin-bottom"])
		assert.Equal(t, "8mm", decls["margin-left"])
	})

	t.Run("border-top shorthand expands per-side", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"border-top": "2px dashed blue"})
		assert.Equal(t, "2px", decls["border-top-width"])
		assert.Equal(t, "dashed", decls["border-top-style"])
		assert.Equal(t, "blue", decls["border-top-color"])
	})

	t.Run("unknown property passes through unchanged", func(t *testing.T) {
		t.Parallel()
		decls := css.ExpandShorthands(map[string]string{"color": "red"})
		assert.Equal(t, "red", decls["color"])
	})
}

// ── Specificity ───────────────────────────────────────────────────────────────

func TestSpecificity(t *testing.T) {
	t.Parallel()
	// Inline (a) > id (b) > class (c) > element (d)
	assert.Greater(t, css.Specificity(1, 0, 0, 0), css.Specificity(0, 9, 9, 9))
	assert.Greater(t, css.Specificity(0, 1, 0, 0), css.Specificity(0, 0, 9, 9))
	assert.Greater(t, css.Specificity(0, 0, 1, 0), css.Specificity(0, 0, 0, 9))
	// Equal specificity: later rule wins (caller's responsibility, not compared here)
	assert.Equal(t, css.Specificity(0, 1, 0, 0), css.Specificity(0, 1, 0, 0))
}

// ── Length parsing ────────────────────────────────────────────────────────────

func TestParseLength(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input    string
		expected float64
		parent   float64
	}{
		{"12pt", 12 * 0.352778, 0},
		{"5mm", 5, 0},
		{"16px", 16 * 0.264583, 0},
		{"1.5em", 15, 10}, // parent=10mm
		{"2cm", 20, 0},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := css.ParseLength(tc.input, tc.parent)
			assert.InDelta(t, tc.expected, got, 0.01)
		})
	}
}
