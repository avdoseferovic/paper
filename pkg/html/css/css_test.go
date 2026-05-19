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
	assert.Equal(t, "", s.Display) // unset by default; treat as block
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

	t.Run("flex:1 expands to grow/shrink/basis", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "1"})
		assert.Equal(t, "1", d["flex-grow"])
		assert.Equal(t, "1", d["flex-shrink"])
		assert.Equal(t, "0", d["flex-basis"])
	})

	t.Run("flex:auto", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "auto"})
		assert.Equal(t, "1", d["flex-grow"])
		assert.Equal(t, "1", d["flex-shrink"])
		assert.Equal(t, "auto", d["flex-basis"])
	})

	t.Run("flex:none", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "none"})
		assert.Equal(t, "0", d["flex-grow"])
		assert.Equal(t, "0", d["flex-shrink"])
		assert.Equal(t, "auto", d["flex-basis"])
	})

	t.Run("flex:initial", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "initial"})
		assert.Equal(t, "0", d["flex-grow"])
		assert.Equal(t, "1", d["flex-shrink"])
		assert.Equal(t, "auto", d["flex-basis"])
	})

	t.Run("flex:2 50mm (grow + basis)", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "2 50mm"})
		assert.Equal(t, "2", d["flex-grow"])
		assert.Equal(t, "1", d["flex-shrink"])
		assert.Equal(t, "50mm", d["flex-basis"])
	})

	t.Run("flex:3 2 (two numbers)", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "3 2"})
		assert.Equal(t, "3", d["flex-grow"])
		assert.Equal(t, "2", d["flex-shrink"])
		assert.Equal(t, "0", d["flex-basis"])
	})

	t.Run("flex:1 0 100% (all three)", func(t *testing.T) {
		t.Parallel()
		d := css.ExpandShorthands(map[string]string{"flex": "1 0 100%"})
		assert.Equal(t, "1", d["flex-grow"])
		assert.Equal(t, "0", d["flex-shrink"])
		assert.Equal(t, "100%", d["flex-basis"])
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

// ── Flex properties ───────────────────────────────────────────────────────────

func TestComputedStyle_FlexFields(t *testing.T) {
	t.Parallel()

	t.Run("display:flex normalises to flex", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("display", "flex", nil)
		assert.Equal(t, "flex", s.Display)
	})

	t.Run("display:inline-flex normalises to flex", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("display", "inline-flex", nil)
		assert.Equal(t, "flex", s.Display)
	})

	t.Run("flex-direction stored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-direction", "column", nil)
		assert.Equal(t, "column", s.FlexDirection)
	})

	t.Run("justify-content stored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("justify-content", "center", nil)
		assert.Equal(t, "center", s.JustifyContent)
	})

	t.Run("align-items stored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("align-items", "flex-end", nil)
		assert.Equal(t, "flex-end", s.AlignItems)
	})

	t.Run("flex-grow stored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-grow", "2", nil)
		assert.InDelta(t, 2.0, s.FlexGrow, 0.001)
	})

	t.Run("flex-shrink stored", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-shrink", "0", nil)
		assert.InDelta(t, 0.0, s.FlexShrink, 0.001)
	})

	t.Run("flex-basis auto sets FlexBasisAuto", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-basis", "auto", nil)
		assert.True(t, s.FlexBasisAuto)
		assert.InDelta(t, 0.0, s.FlexBasis, 0.001)
		assert.InDelta(t, 0.0, s.FlexBasisPct, 0.001)
	})

	t.Run("flex-basis mm sets FlexBasis", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-basis", "50mm", nil)
		assert.InDelta(t, 50.0, s.FlexBasis, 0.001)
		assert.False(t, s.FlexBasisAuto)
		assert.InDelta(t, 0.0, s.FlexBasisPct, 0.001)
	})

	t.Run("flex-basis percent sets FlexBasisPct", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("flex-basis", "25%", nil)
		assert.InDelta(t, 25.0, s.FlexBasisPct, 0.001)
		assert.InDelta(t, 0.0, s.FlexBasis, 0.001)
		assert.False(t, s.FlexBasisAuto)
	})

	t.Run("gap single value sets both row and column gap", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("gap", "5mm", nil)
		assert.InDelta(t, 5.0, s.RowGap, 0.001)
		assert.InDelta(t, 5.0, s.ColumnGap, 0.001)
	})

	t.Run("gap two values sets row and column separately", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("gap", "5mm 10mm", nil)
		assert.InDelta(t, 5.0, s.RowGap, 0.001)
		assert.InDelta(t, 10.0, s.ColumnGap, 0.001)
	})

	t.Run("row-gap sets RowGap", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("row-gap", "3mm", nil)
		assert.InDelta(t, 3.0, s.RowGap, 0.001)
	})

	t.Run("column-gap sets ColumnGap", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("column-gap", "8mm", nil)
		assert.InDelta(t, 8.0, s.ColumnGap, 0.001)
	})
}

func TestParsePercentage(t *testing.T) {
	t.Parallel()

	t.Run("parses percentage string", func(t *testing.T) {
		t.Parallel()
		pct, ok := css.ParsePercentage("25%")
		assert.True(t, ok)
		assert.InDelta(t, 0.25, pct, 0.0001)
	})

	t.Run("parses 100%", func(t *testing.T) {
		t.Parallel()
		pct, ok := css.ParsePercentage("100%")
		assert.True(t, ok)
		assert.InDelta(t, 1.0, pct, 0.0001)
	})

	t.Run("returns false for non-percentage", func(t *testing.T) {
		t.Parallel()
		_, ok := css.ParsePercentage("50px")
		assert.False(t, ok)
	})

	t.Run("returns false for empty string", func(t *testing.T) {
		t.Parallel()
		_, ok := css.ParsePercentage("")
		assert.False(t, ok)
	})
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

func TestBorderRadius_Apply(t *testing.T) {
	t.Parallel()

	t.Run("uniform border-radius", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("border-radius", "4mm", nil)
		assert.InDelta(t, 4.0, s.BorderRadius, 0.01)
	})

	t.Run("border-top-left-radius", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("border-top-left-radius", "3mm", nil)
		assert.InDelta(t, 3.0, s.BorderRadiusTopLeft, 0.01)
		assert.Equal(t, 0.0, s.BorderRadius)
	})

	t.Run("all four per-corner longhands", func(t *testing.T) {
		t.Parallel()
		s := css.NewComputedStyle()
		s.Apply("border-top-left-radius", "1mm", nil)
		s.Apply("border-top-right-radius", "2mm", nil)
		s.Apply("border-bottom-right-radius", "3mm", nil)
		s.Apply("border-bottom-left-radius", "4mm", nil)
		assert.InDelta(t, 1.0, s.BorderRadiusTopLeft, 0.01)
		assert.InDelta(t, 2.0, s.BorderRadiusTopRight, 0.01)
		assert.InDelta(t, 3.0, s.BorderRadiusBottomRight, 0.01)
		assert.InDelta(t, 4.0, s.BorderRadiusBottomLeft, 0.01)
	})
}

func TestExpandBorderRadius(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		tl   string
		tr   string
		br   string
		bl   string
	}{
		{"one value", "4mm", "4mm", "4mm", "4mm", "4mm"},
		{"two values", "4mm 8mm", "4mm", "8mm", "4mm", "8mm"},
		{"three values", "4mm 8mm 2mm", "4mm", "8mm", "2mm", "8mm"},
		{"four values", "1mm 2mm 3mm 4mm", "1mm", "2mm", "3mm", "4mm"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := css.ExpandShorthands(map[string]string{"border-radius": tc.in})
			assert.Equal(t, tc.tl, out["border-top-left-radius"])
			assert.Equal(t, tc.tr, out["border-top-right-radius"])
			assert.Equal(t, tc.br, out["border-bottom-right-radius"])
			assert.Equal(t, tc.bl, out["border-bottom-left-radius"])
		})
	}
}
