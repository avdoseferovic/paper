package css

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

// --- ApplyCtx dispatch, custom properties, var() resolution ---

func TestApplyCtx_CustomPropertiesAndVars(t *testing.T) {
	t.Parallel()

	t.Run("custom property declaration is stored in Vars", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.ApplyCtx("--accent", "red", nil, 0)
		require.NotNil(t, s.Vars)
		assert.Equal(t, "red", s.Vars["--accent"])
	})

	t.Run("var reference resolves against Vars before dispatch", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.ApplyCtx("--c", "#ff0000", nil, 0)
		s.ApplyCtx("color", "var(--c)", nil, 0)
		require.NotNil(t, s.Color)
		assert.Equal(t, 255, s.Color.R)
		assert.Equal(t, 0, s.Color.G)
	})

	t.Run("unknown property invokes unsupported handler", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		var gotProp, gotVal string
		s.SetUnsupportedHandler(func(prop, val string) { gotProp, gotVal = prop, val })
		s.ApplyCtx("zz-not-a-property", "whatever", nil, 0)
		assert.Equal(t, "zz-not-a-property", gotProp)
		assert.Equal(t, "whatever", gotVal)
	})

	t.Run("unknown property without handler is a no-op", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		assert.NotPanics(t, func() { s.ApplyCtx("zz-not-a-property", "x", nil, 0) })
	})

	t.Run("parent font size resolves em units", func(t *testing.T) {
		t.Parallel()
		parent := NewComputedStyle()
		parent.FontSize = 10.0
		s := NewComputedStyle()
		s.ApplyCtx("font-size", "2em", parent, 0)
		assert.InDelta(t, 20.0, s.FontSize, 0.001)
	})
}

// --- applyFontProperty ---

func TestApplyFontProperty(t *testing.T) {
	t.Parallel()

	t.Run("font-family strips quotes", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("font-family", `"Helvetica"`, nil)
		assert.Equal(t, "Helvetica", s.FontFamily)
	})

	t.Run("font-weight normalization", func(t *testing.T) {
		t.Parallel()
		cases := []struct{ in, want string }{
			{"bold", "bold"},
			{"bolder", "bold"},
			{"700", "bold"},
			{"800", "bold"},
			{"900", "bold"},
			{"normal", "normal"},
			{"400", "normal"},
			{"lighter", "normal"},
		}
		for _, tc := range cases {
			s := NewComputedStyle()
			s.Apply("font-weight", tc.in, nil)
			assert.Equal(t, tc.want, s.FontWeight, "font-weight %q", tc.in)
		}
	})

	t.Run("font-style text-align text-decoration", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("font-style", "italic", nil)
		s.Apply("text-align", "center", nil)
		s.Apply("text-decoration", "underline", nil)
		assert.Equal(t, "italic", s.FontStyle)
		assert.Equal(t, "center", s.TextAlign)
		assert.Equal(t, "underline", s.TextDecoration)
	})

	t.Run("line-height unitless multiplier", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("line-height", "1.5", nil)
		assert.InDelta(t, 1.5, s.LineHeight, 0.001)
	})
}

// --- applyBoxProperty ---

func TestApplyBoxProperty(t *testing.T) {
	t.Parallel()

	t.Run("padding and margin longhands", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		for prop, val := range map[string]string{
			"padding-top": "1mm", "padding-right": "2mm",
			"padding-bottom": "3mm", "padding-left": "4mm",
			"margin-top": "5mm", "margin-right": "6mm",
			"margin-bottom": "7mm", "margin-left": "8mm",
		} {
			s.Apply(prop, val, nil)
		}
		assert.InDelta(t, 1.0, s.PaddingTop, 0.001)
		assert.InDelta(t, 2.0, s.PaddingRight, 0.001)
		assert.InDelta(t, 3.0, s.PaddingBottom, 0.001)
		assert.InDelta(t, 4.0, s.PaddingLeft, 0.001)
		assert.InDelta(t, 5.0, s.MarginTop, 0.001)
		assert.InDelta(t, 6.0, s.MarginRight, 0.001)
		assert.InDelta(t, 7.0, s.MarginBottom, 0.001)
		assert.InDelta(t, 8.0, s.MarginLeft, 0.001)
	})

	t.Run("display inline-flex maps to flex", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("display", "inline-flex", nil)
		assert.Equal(t, "flex", s.Display)
	})

	t.Run("display other values pass through", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("display", "inline-block", nil)
		assert.Equal(t, "inline-block", s.Display)
	})

	t.Run("visibility normalized to lowercase", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("visibility", "  Hidden ", nil)
		assert.Equal(t, "hidden", s.Visibility)
	})

	t.Run("dimension properties", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.ApplyCtx("width", "50%", nil, 100.0)
		s.Apply("height", "30mm", nil)
		s.ApplyCtx("min-width", "10mm", nil, 0)
		s.ApplyCtx("max-width", "90mm", nil, 0)
		s.Apply("min-height", "5mm", nil)
		s.Apply("max-height", "60mm", nil)
		assert.InDelta(t, 50.0, s.Width, 0.001)
		assert.InDelta(t, 30.0, s.Height, 0.001)
		assert.InDelta(t, 10.0, s.MinWidth, 0.001)
		assert.InDelta(t, 90.0, s.MaxWidth, 0.001)
		assert.InDelta(t, 5.0, s.MinHeight, 0.001)
		assert.InDelta(t, 60.0, s.MaxHeight, 0.001)
	})

	t.Run("object-fit and object-position normalized", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("object-fit", " Cover ", nil)
		s.Apply("object-position", " Top Left ", nil)
		assert.Equal(t, "cover", s.ObjectFit)
		assert.Equal(t, "top left", s.ObjectPosition)
	})
}

// --- applyBorderProperty and parseOutlineShorthand ---

func TestApplyBorderProperty(t *testing.T) {
	t.Parallel()

	t.Run("outline longhands", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("outline-width", "2mm", nil)
		s.Apply("outline-style", " dashed ", nil)
		s.Apply("outline-color", "blue", nil)
		s.Apply("outline-offset", "1mm", nil)
		assert.InDelta(t, 2.0, s.OutlineWidth, 0.001)
		assert.Equal(t, "dashed", s.OutlineStyle)
		require.NotNil(t, s.OutlineColor)
		assert.Equal(t, 255, s.OutlineColor.B)
		assert.InDelta(t, 1.0, s.OutlineOffset, 0.001)
	})

	t.Run("outline shorthand any order", func(t *testing.T) {
		t.Parallel()
		cases := []string{
			"2mm solid red",
			"solid 2mm red",
			"red solid 2mm",
		}
		for _, val := range cases {
			s := NewComputedStyle()
			s.Apply("outline", val, nil)
			assert.InDelta(t, 2.0, s.OutlineWidth, 0.001, "outline %q", val)
			assert.Equal(t, "solid", s.OutlineStyle, "outline %q", val)
			require.NotNil(t, s.OutlineColor, "outline %q", val)
			assert.Equal(t, 255, s.OutlineColor.R, "outline %q", val)
		}
	})

	t.Run("outline shorthand ignores unknown tokens", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("outline", "wibble dotted", nil)
		assert.Equal(t, "dotted", s.OutlineStyle)
		assert.InDelta(t, 0.0, s.OutlineWidth, 0.001)
	})

	t.Run("per-side border longhands", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("border-top-width", "1mm", nil)
		s.Apply("border-right-width", "2mm", nil)
		s.Apply("border-bottom-width", "3mm", nil)
		s.Apply("border-left-width", "4mm", nil)
		s.Apply("border-top-style", "solid", nil)
		s.Apply("border-right-style", "dashed", nil)
		s.Apply("border-bottom-style", "dotted", nil)
		s.Apply("border-left-style", "double", nil)
		s.Apply("border-top-color", "red", nil)
		s.Apply("border-right-color", "lime", nil)
		s.Apply("border-bottom-color", "blue", nil)
		s.Apply("border-left-color", "black", nil)
		assert.InDelta(t, 1.0, s.BorderTopWidth, 0.001)
		assert.InDelta(t, 2.0, s.BorderRightWidth, 0.001)
		assert.InDelta(t, 3.0, s.BorderBottomWidth, 0.001)
		assert.InDelta(t, 4.0, s.BorderLeftWidth, 0.001)
		assert.Equal(t, "solid", s.BorderTopStyle)
		assert.Equal(t, "dashed", s.BorderRightStyle)
		assert.Equal(t, "dotted", s.BorderBottomStyle)
		assert.Equal(t, "double", s.BorderLeftStyle)
		require.NotNil(t, s.BorderTopColor)
		assert.Equal(t, 255, s.BorderTopColor.R)
		require.NotNil(t, s.BorderRightColor)
		assert.Equal(t, 255, s.BorderRightColor.G)
		require.NotNil(t, s.BorderBottomColor)
		assert.Equal(t, 255, s.BorderBottomColor.B)
		require.NotNil(t, s.BorderLeftColor)
		assert.Equal(t, 0, s.BorderLeftColor.R)
	})

	t.Run("border-color border-width border-style set all four sides", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("border-color", "red", nil)
		s.Apply("border-width", "2mm", nil)
		s.Apply("border-style", "solid", nil)
		for _, c := range []*RGBColor{s.BorderTopColor, s.BorderRightColor, s.BorderBottomColor, s.BorderLeftColor} {
			require.NotNil(t, c)
			assert.Equal(t, 255, c.R)
		}
		for _, w := range []float64{s.BorderTopWidth, s.BorderRightWidth, s.BorderBottomWidth, s.BorderLeftWidth} {
			assert.InDelta(t, 2.0, w, 0.001)
		}
		for _, st := range []string{s.BorderTopStyle, s.BorderRightStyle, s.BorderBottomStyle, s.BorderLeftStyle} {
			assert.Equal(t, "solid", st)
		}
	})

	t.Run("border-radius corners", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("border-radius", "1mm", nil)
		s.Apply("border-top-left-radius", "2mm", nil)
		s.Apply("border-top-right-radius", "3mm", nil)
		s.Apply("border-bottom-left-radius", "4mm", nil)
		s.Apply("border-bottom-right-radius", "5mm", nil)
		assert.InDelta(t, 1.0, s.BorderRadius, 0.001)
		assert.InDelta(t, 2.0, s.BorderRadiusTopLeft, 0.001)
		assert.InDelta(t, 3.0, s.BorderRadiusTopRight, 0.001)
		assert.InDelta(t, 4.0, s.BorderRadiusBottomLeft, 0.001)
		assert.InDelta(t, 5.0, s.BorderRadiusBottomRight, 0.001)
	})
}

// --- applyTypographyProperty ---

func TestApplyTypographyProperty(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.Apply("page-break-before", "always", nil)
	s.Apply("break-after", "avoid", nil)
	s.Apply("break-inside", "avoid", nil)
	s.Apply("list-style-type", "decimal", nil)
	s.Apply("vertical-align", " Super ", nil)
	s.Apply("content", ` "hi" `, nil)
	s.Apply("counter-reset", "section 0", nil)
	s.Apply("counter-increment", "section", nil)
	s.Apply("quotes", `"«" "»"`, nil)
	assert.Equal(t, "always", s.PageBreakBefore)
	assert.Equal(t, "avoid", s.PageBreakAfter)
	assert.Equal(t, "avoid", s.BreakInside)
	assert.Equal(t, "decimal", s.ListStyleType)
	assert.Equal(t, "super", s.VerticalAlign)
	assert.Equal(t, `"hi"`, s.Content)
	assert.Equal(t, "section 0", s.CounterReset)
	assert.Equal(t, "section", s.CounterIncrement)
	assert.Equal(t, `"«" "»"`, s.Quotes)
}

// --- applyFlexProperty ---

func TestApplyFlexProperty(t *testing.T) {
	t.Parallel()

	t.Run("container and item keywords", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("flex-direction", "column", nil)
		s.Apply("justify-content", "space-between", nil)
		s.Apply("align-items", "center", nil)
		s.Apply("align-self", " flex-end ", nil)
		s.Apply("flex-wrap", " wrap ", nil)
		assert.Equal(t, "column", s.FlexDirection)
		assert.Equal(t, "space-between", s.JustifyContent)
		assert.Equal(t, "center", s.AlignItems)
		assert.Equal(t, "flex-end", s.AlignSelf)
		assert.Equal(t, "wrap", s.FlexWrap)
	})

	t.Run("order valid and invalid", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("order", " 3 ", nil)
		assert.Equal(t, 3, s.Order)
		s.Apply("order", "abc", nil)
		assert.Equal(t, 3, s.Order) // unchanged on parse failure
	})

	t.Run("flex-grow and flex-shrink invalid values ignored", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("flex-grow", "2", nil)
		s.Apply("flex-shrink", "0.5", nil)
		assert.InDelta(t, 2.0, s.FlexGrow, 0.001)
		assert.InDelta(t, 0.5, s.FlexShrink, 0.001)
		s.Apply("flex-grow", "x", nil)
		s.Apply("flex-shrink", "y", nil)
		assert.InDelta(t, 2.0, s.FlexGrow, 0.001)
		assert.InDelta(t, 0.5, s.FlexShrink, 0.001)
	})

	t.Run("flex-basis auto percent and length", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("flex-basis", "auto", nil)
		assert.True(t, s.FlexBasisAuto)

		s.Apply("flex-basis", "25%", nil)
		assert.False(t, s.FlexBasisAuto)
		assert.InDelta(t, 25.0, s.FlexBasisPct, 0.001)

		s.Apply("flex-basis", "10mm", nil)
		assert.False(t, s.FlexBasisAuto)
		assert.InDelta(t, 0.0, s.FlexBasisPct, 0.001)
		assert.InDelta(t, 10.0, s.FlexBasis, 0.001)
	})

	t.Run("gap single and double values", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("gap", "4mm", nil)
		assert.InDelta(t, 4.0, s.RowGap, 0.001)
		assert.InDelta(t, 4.0, s.ColumnGap, 0.001)

		s.Apply("gap", "2mm 6mm", nil)
		assert.InDelta(t, 2.0, s.RowGap, 0.001)
		assert.InDelta(t, 6.0, s.ColumnGap, 0.001)

		s.Apply("row-gap", "1mm", nil)
		s.Apply("column-gap", "3mm", nil)
		assert.InDelta(t, 1.0, s.RowGap, 0.001)
		assert.InDelta(t, 3.0, s.ColumnGap, 0.001)
	})
}

// --- applyEffectsProperty and applyBackgroundImage ---

func TestApplyEffectsProperty(t *testing.T) {
	t.Parallel()

	t.Run("box-shadow valid and invalid", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("box-shadow", "1mm 2mm 3mm red", nil)
		require.Len(t, s.BoxShadow, 1)
		assert.InDelta(t, 1.0, s.BoxShadow[0].OffsetX, 0.001)

		var unsupported []string
		s2 := NewComputedStyle()
		s2.SetUnsupportedHandler(func(prop, val string) { unsupported = append(unsupported, prop) })
		s2.Apply("box-shadow", "red", nil)
		assert.Contains(t, unsupported, "box-shadow")
	})

	t.Run("text-shadow valid and invalid", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("text-shadow", "1mm 1mm 2mm blue", nil)
		require.Len(t, s.TextShadows, 1)
		require.NotNil(t, s.TextShadow)
		assert.InDelta(t, 1.0, s.TextShadow.OffsetX, 0.001)

		var unsupported []string
		s2 := NewComputedStyle()
		s2.SetUnsupportedHandler(func(prop, val string) { unsupported = append(unsupported, prop) })
		s2.Apply("text-shadow", "notashadow", nil)
		assert.Contains(t, unsupported, "text-shadow")
	})

	t.Run("background size position repeat normalized", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("background-size", " Cover ", nil)
		s.Apply("background-position", " Center ", nil)
		s.Apply("background-repeat", " No-Repeat ", nil)
		assert.Equal(t, "cover", s.BackgroundSize)
		assert.Equal(t, "center", s.BackgroundPosition)
		assert.Equal(t, "no-repeat", s.BackgroundRepeat)
	})

	t.Run("filter drop-shadow appends and caps at four", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("box-shadow", "1mm 1mm, 2mm 2mm, 3mm 3mm", nil)
		require.Len(t, s.BoxShadow, 3)
		s.Apply("filter", "drop-shadow(4mm 4mm) drop-shadow(5mm 5mm)", nil)
		assert.Len(t, s.BoxShadow, 4)
	})

	t.Run("filter invalid invokes handler", func(t *testing.T) {
		t.Parallel()
		var unsupported []string
		s := NewComputedStyle()
		s.SetUnsupportedHandler(func(prop, val string) { unsupported = append(unsupported, prop) })
		s.Apply("filter", "blur(5px)", nil)
		assert.Contains(t, unsupported, "filter")
	})
}

func TestApplyBackgroundImage(t *testing.T) {
	t.Parallel()

	t.Run("none clears image and gradient", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("background-image", "url(a.png)", nil)
		require.Equal(t, "a.png", s.BackgroundImageURL)
		s.Apply("background-image", "none", nil)
		assert.Equal(t, "", s.BackgroundImageURL)
		assert.Nil(t, s.BackgroundGradient)
	})

	t.Run("linear gradient", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("background-image", "linear-gradient(to right, red, blue)", nil)
		require.NotNil(t, s.BackgroundGradient)
		assert.Equal(t, GradientLinear, s.BackgroundGradient.Kind)
		require.NotNil(t, s.BackgroundGradient.Linear)
	})

	t.Run("radial gradient", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("background-image", "radial-gradient(circle, red, blue)", nil)
		require.NotNil(t, s.BackgroundGradient)
		assert.Equal(t, GradientRadial, s.BackgroundGradient.Kind)
		require.NotNil(t, s.BackgroundGradient.Radial)
	})

	t.Run("conic gradient", func(t *testing.T) {
		t.Parallel()
		s := NewComputedStyle()
		s.Apply("background-image", "conic-gradient(red, blue)", nil)
		require.NotNil(t, s.BackgroundGradient)
		assert.Equal(t, GradientConic, s.BackgroundGradient.Kind)
		require.NotNil(t, s.BackgroundGradient.Conic)
	})

	t.Run("invalid gradients invoke handler", func(t *testing.T) {
		t.Parallel()
		cases := []string{
			"linear-gradient(red)",
			"radial-gradient(red)",
			"conic-gradient(red)",
			"url()",
			"cross-fade(red, blue)",
		}
		for _, val := range cases {
			var unsupported []string
			s := NewComputedStyle()
			s.SetUnsupportedHandler(func(prop, v string) { unsupported = append(unsupported, v) })
			s.Apply("background-image", val, nil)
			assert.Contains(t, unsupported, val, "value %q", val)
		}
	})
}

// --- ParseCSSURL ---

func TestParseCSSURL_EdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"url(a.png)", "a.png", true},
		{`url("a.png")`, "a.png", true},
		{`url('a.png')`, "a.png", true},
		{" url( b.png ) ", "b.png", true},
		{"url()", "", false},
		{"url(   )", "", false},
		{`url('')`, "", false},
		{"not-a-url", "", false},
		{"url(a.png", "", false},
	}
	for _, tc := range cases {
		got, ok := ParseCSSURL(tc.in)
		assert.Equal(t, tc.ok, ok, "input %q", tc.in)
		assert.Equal(t, tc.want, got, "input %q", tc.in)
	}
}

// --- applyOpacity ---

func TestApplyOpacity_EdgeCases(t *testing.T) {
	t.Parallel()

	s := NewComputedStyle()
	s.Apply("opacity", "garbage", nil)
	assert.InDelta(t, 1.0, s.Opacity, 0.001, "invalid value leaves opacity unchanged")

	s.Apply("opacity", " 45% ", nil)
	assert.InDelta(t, 0.45, s.Opacity, 0.001)

	s.Apply("opacity", "-2", nil)
	assert.InDelta(t, 0.0, s.Opacity, 0.001)

	s.Apply("opacity", "3", nil)
	assert.InDelta(t, 1.0, s.Opacity, 0.001)
}

// --- Gradients: directions, shapes, positions, stops ---

func TestParseLinearDirection_AllKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		dir  string
		want float64
	}{
		{"to right", 90},
		{"to left", 270},
		{"to bottom", 180},
		{"to top", 0},
		{"to top right", 45},
		{"to right top", 45},
		{"to bottom right", 135},
		{"to right bottom", 135},
		{"to bottom left", 225},
		{"to left bottom", 225},
		{"to top left", 315},
		{"to left top", 315},
		{"30deg", 30},
		{"0.25turn", 90},
		{"3.14159265rad", 180},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			t.Parallel()
			g, err := ParseLinearGradient("linear-gradient(" + tc.dir + ", red, blue)")
			require.NoError(t, err)
			assert.InDelta(t, tc.want, g.AngleDeg, 0.01)
			assert.Len(t, g.Stops, 2)
		})
	}

	t.Run("color first defaults to 180", func(t *testing.T) {
		t.Parallel()
		g, err := ParseLinearGradient("linear-gradient(red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 180.0, g.AngleDeg, 0.01)
	})
}

func TestParseLinearGradient_Errors(t *testing.T) {
	t.Parallel()

	cases := []string{
		"radial-gradient(red, blue)",     // wrong function name
		"linear-gradient(red)",           // fewer than 2 parts
		"linear-gradient(zzz, blue)",     // unknown color (direction fallback path)
		"linear-gradient(xdeg, red)",     // bad angle treated as color, then fails
		"linear-gradient(red 1x%, blue)", // malformed stop position
	}
	for _, in := range cases {
		_, err := ParseLinearGradient(in)
		assert.Error(t, err, "input %q", in)
	}
}

func TestParseRadialGradient_ShapesAndPositions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in     string
		circle bool
		cx, cy float64
	}{
		{"radial-gradient(circle at center, red, blue)", true, 0.5, 0.5},
		{"radial-gradient(circle at top, red, blue)", true, 0.5, 0.0},
		{"radial-gradient(circle at bottom, red, blue)", true, 0.5, 1.0},
		{"radial-gradient(circle at left, red, blue)", true, 0.0, 0.5},
		{"radial-gradient(circle at right, red, blue)", true, 1.0, 0.5},
		{"radial-gradient(circle at top left, red, blue)", true, 0.0, 0.0},
		{"radial-gradient(circle at right top, red, blue)", true, 1.0, 0.0},
		{"radial-gradient(circle at bottom left, red, blue)", true, 0.0, 1.0},
		{"radial-gradient(circle at right bottom, red, blue)", true, 1.0, 1.0},
		{"radial-gradient(circle at 25% 75%, red, blue)", true, 0.5, 0.5}, // unsupported position defaults to centre
		{"radial-gradient(ellipse at left, red, blue)", false, 0.0, 0.5},
		{"radial-gradient(circle, red, blue)", true, 0.5, 0.5},
		{"radial-gradient(red, blue)", true, 0.5, 0.5}, // no shape prelude
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			g, err := ParseRadialGradient(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.circle, g.Circle)
			assert.InDelta(t, tc.cx, g.CX, 0.001)
			assert.InDelta(t, tc.cy, g.CY, 0.001)
		})
	}
}

func TestParseRadialGradient_Errors(t *testing.T) {
	t.Parallel()

	cases := []string{
		"linear-gradient(red, blue)",
		"radial-gradient(red)",
		"radial-gradient(circle, zzz, blue)",
	}
	for _, in := range cases {
		_, err := ParseRadialGradient(in)
		assert.Error(t, err, "input %q", in)
	}
}

func TestParseConicGradient_Prelude(t *testing.T) {
	t.Parallel()

	t.Run("from turn angle", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(from 0.5turn, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 180.0, g.FromDeg, 0.01)
	})

	t.Run("from rad angle", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(from 1.5707963rad, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 90.0, g.FromDeg, 0.01)
	})

	t.Run("from with at position", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(from 90deg at top left, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 90.0, g.FromDeg, 0.01)
		assert.InDelta(t, 0.0, g.CX, 0.001)
		assert.InDelta(t, 0.0, g.CY, 0.001)
	})

	t.Run("at only", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(at bottom right, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 0.0, g.FromDeg, 0.01)
		assert.InDelta(t, 1.0, g.CX, 0.001)
		assert.InDelta(t, 1.0, g.CY, 0.001)
	})

	t.Run("from with unitless angle keeps default", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(from 90, red, blue)")
		require.NoError(t, err)
		assert.InDelta(t, 0.0, g.FromDeg, 0.01)
	})

	t.Run("angle stop positions in degrees", func(t *testing.T) {
		t.Parallel()
		g, err := ParseConicGradient("conic-gradient(red 0deg, blue 180deg)")
		require.NoError(t, err)
		require.Len(t, g.Stops, 2)
		assert.InDelta(t, 0.0, g.Stops[0].Position, 0.001)
		assert.InDelta(t, 0.5, g.Stops[1].Position, 0.001)
	})
}

func TestParseConicGradient_Errors(t *testing.T) {
	t.Parallel()

	cases := []string{
		"linear-gradient(red, blue)",
		"conic-gradient(red)",
		"conic-gradient(zzz, blue)",
	}
	for _, in := range cases {
		_, err := ParseConicGradient(in)
		assert.Error(t, err, "input %q", in)
	}
}

func TestDistributeStops(t *testing.T) {
	t.Parallel()

	t.Run("empty slice is a no-op", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { distributeStops(nil) })
	})

	t.Run("all auto distributes evenly", func(t *testing.T) {
		t.Parallel()
		stops := []GradientStop{{Position: -1}, {Position: -1}, {Position: -1}}
		distributeStops(stops)
		assert.InDelta(t, 0.0, stops[0].Position, 0.001)
		assert.InDelta(t, 0.5, stops[1].Position, 0.001)
		assert.InDelta(t, 1.0, stops[2].Position, 0.001)
	})

	t.Run("interior autos interpolate between anchors", func(t *testing.T) {
		t.Parallel()
		stops := []GradientStop{
			{Position: 0.2}, {Position: -1}, {Position: -1}, {Position: 0.8},
		}
		distributeStops(stops)
		assert.InDelta(t, 0.2, stops[0].Position, 0.001)
		assert.InDelta(t, 0.4, stops[1].Position, 0.001)
		assert.InDelta(t, 0.6, stops[2].Position, 0.001)
		assert.InDelta(t, 0.8, stops[3].Position, 0.001)
	})

	t.Run("explicit positions are untouched", func(t *testing.T) {
		t.Parallel()
		stops := []GradientStop{{Position: 0.1}, {Position: 0.9}}
		distributeStops(stops)
		assert.InDelta(t, 0.1, stops[0].Position, 0.001)
		assert.InDelta(t, 0.9, stops[1].Position, 0.001)
	})
}

func TestParseAngleDeg(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"90deg", 90, true},
		{"0.5turn", 180, true},
		{"3.14159265rad", 180, true},
		{"xdeg", 0, false},
		{"xturn", 0, false},
		{"xrad", 0, false},
		{"90", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseAngleDeg(tc.in)
		assert.Equal(t, tc.ok, ok, "input %q", tc.in)
		if tc.ok {
			assert.InDelta(t, tc.want, got, 0.01, "input %q", tc.in)
		}
	}
}

func TestParseStops_SkipsEmptyParts(t *testing.T) {
	t.Parallel()

	g, err := ParseLinearGradient("linear-gradient(red, , blue)")
	require.NoError(t, err)
	assert.Len(t, g.Stops, 2)
}

func TestParseStops_PositionVariants(t *testing.T) {
	t.Parallel()

	t.Run("bad angle position is an error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseLinearGradient("linear-gradient(red xdeg, blue)")
		assert.Error(t, err)
	})

	t.Run("length position is treated as auto", func(t *testing.T) {
		t.Parallel()
		g, err := ParseLinearGradient("linear-gradient(red 10px, blue)")
		require.NoError(t, err)
		require.Len(t, g.Stops, 2)
		assert.InDelta(t, 0.0, g.Stops[0].Position, 0.001) // auto -> distributed to 0
	})

	t.Run("multi-token color with position", func(t *testing.T) {
		t.Parallel()
		g, err := ParseLinearGradient("linear-gradient(to right, rgb(255, 0, 0) 25%, blue)")
		require.NoError(t, err)
		require.Len(t, g.Stops, 2)
		assert.Equal(t, 255, g.Stops[0].Color.R)
		assert.InDelta(t, 0.25, g.Stops[0].Position, 0.001)
	})
}

// --- Shadow / filter parsing ---

func TestParseShadow_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("truncates to four shadows", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseShadow("1mm 1mm, 2mm 2mm, 3mm 3mm, 4mm 4mm, 5mm 5mm")
		require.NoError(t, err)
		assert.Len(t, shadows, 4)
	})

	t.Run("inset with blur spread and color", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseShadow("inset 1mm 2mm 3mm 4mm red")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		s := shadows[0]
		assert.True(t, s.Inset)
		assert.InDelta(t, 1.0, s.OffsetX, 0.001)
		assert.InDelta(t, 2.0, s.OffsetY, 0.001)
		assert.InDelta(t, 3.0, s.BlurRadius, 0.001)
		assert.InDelta(t, 4.0, s.Spread, 0.001)
		require.NotNil(t, s.Color)
		assert.Equal(t, 255, s.Color.R)
	})

	t.Run("empty entries between commas are skipped", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseShadow("1mm 1mm, ,")
		require.NoError(t, err)
		assert.Len(t, shadows, 1)
	})

	t.Run("single offset is an error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseShadow("1mm red")
		assert.Error(t, err)
	})
}

func TestParseFilterDropShadow_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("none and empty are errors", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFilterDropShadow("none")
		assert.ErrorIs(t, err, errDropShadowMissing)
		_, err = ParseFilterDropShadow("   ")
		assert.ErrorIs(t, err, errDropShadowMissing)
	})

	t.Run("filters without drop-shadow are errors", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFilterDropShadow("blur(5px) brightness(0.5)")
		assert.ErrorIs(t, err, errDropShadowMissing)
	})

	t.Run("value without parens is an error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFilterDropShadow("drop-shadow")
		assert.ErrorIs(t, err, errDropShadowMissing)
	})

	t.Run("unclosed paren is an error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFilterDropShadow("drop-shadow(1mm 2mm")
		assert.ErrorIs(t, err, errFilterInvalidFunc)
	})

	t.Run("bad drop-shadow args is an error", func(t *testing.T) {
		t.Parallel()
		_, err := ParseFilterDropShadow("drop-shadow(red)")
		assert.Error(t, err)
	})

	t.Run("nested function colors are preserved", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseFilterDropShadow("drop-shadow(1mm 2mm rgb(255,0,0))")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		require.NotNil(t, shadows[0].Color)
		assert.Equal(t, 255, shadows[0].Color.R)
	})

	t.Run("quoted args with escapes in other functions are skipped", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseFilterDropShadow(`blur("a\)b") drop-shadow(1mm 2mm)`)
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		assert.InDelta(t, 1.0, shadows[0].OffsetX, 0.001)
	})

	t.Run("caps at four drop shadows", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseFilterDropShadow(
			"drop-shadow(1mm 1mm) drop-shadow(2mm 2mm) drop-shadow(3mm 3mm) drop-shadow(4mm 4mm) drop-shadow(5mm 5mm)")
		require.NoError(t, err)
		assert.Len(t, shadows, 4)
	})

	t.Run("drop-shadow resets spread and inset", func(t *testing.T) {
		t.Parallel()
		shadows, err := ParseFilterDropShadow("drop-shadow(1mm 2mm 3mm)")
		require.NoError(t, err)
		require.Len(t, shadows, 1)
		assert.InDelta(t, 0.0, shadows[0].Spread, 0.001)
		assert.False(t, shadows[0].Inset)
	})
}

// --- Color parsing edge cases ---

func TestParseColor_ClampingAndChannels(t *testing.T) {
	t.Parallel()

	t.Run("rgb channels clamp to 0..255", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("rgb(300, -20, 128)")
		require.NotNil(t, c)
		assert.Equal(t, 255, c.R)
		assert.Equal(t, 0, c.G)
		assert.Equal(t, 128, c.B)
	})

	t.Run("rgb percentage channels", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("rgb(50%, 100%, 0%)")
		require.NotNil(t, c)
		assert.Equal(t, 128, c.R)
		assert.Equal(t, 255, c.G)
		assert.Equal(t, 0, c.B)
	})

	t.Run("rgba alpha clamps to 0..1", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("rgba(0, 0, 0, 5)")
		require.NotNil(t, c)
		assert.InDelta(t, 1.0, c.A, 0.001)

		c = ParseColor("rgba(0, 0, 0, -0.5)")
		require.NotNil(t, c)
		assert.InDelta(t, 0.0, c.A, 0.001)
	})

	t.Run("rgba percentage alpha", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("rgba(0, 0, 0, 50%)")
		require.NotNil(t, c)
		assert.InDelta(t, 0.5, c.A, 0.001)
	})

	t.Run("invalid channel values return nil", func(t *testing.T) {
		t.Parallel()
		for _, in := range []string{
			"rgb(a, b, c)",
			"rgb(x%, 0, 0)",
			"rgb(1, 2)",
			"rgba(1, 2, 3)",
			"rgba(0, 0, 0, zz)",
			"rgba(0, 0, 0, x%)",
		} {
			assert.Nil(t, ParseColor(in), "input %q", in)
		}
	})
}

func TestParseColor_HSLEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("hue wheel coverage", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			in      string
			r, g, b int
		}{
			{"hsl(120, 100%, 50%)", 0, 255, 0},
			{"hsl(180, 100%, 50%)", 0, 255, 255},
			{"hsl(240, 100%, 50%)", 0, 0, 255},
			{"hsl(300, 100%, 50%)", 255, 0, 255},
			{"hsl(-60, 100%, 50%)", 255, 0, 255},
			{"hsl(60, 100%, 25%)", 127, 128, 0}, // R lands on .5 boundary minus float error
			{"hsl(0, 100%, 75%)", 255, 128, 128},
		}
		for _, tc := range cases {
			c := ParseColor(tc.in)
			require.NotNil(t, c, "input %q", tc.in)
			assert.Equal(t, tc.r, c.R, "R for %q", tc.in)
			assert.Equal(t, tc.g, c.G, "G for %q", tc.in)
			assert.Equal(t, tc.b, c.B, "B for %q", tc.in)
		}
	})

	t.Run("saturation and lightness accept bare floats", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("hsl(0, 1.0, 0.5)")
		require.NotNil(t, c)
		assert.Equal(t, 255, c.R)
	})

	t.Run("hsla with alpha", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("hsla(0, 100%, 50%, 0.25)")
		require.NotNil(t, c)
		assert.Equal(t, 255, c.R)
		assert.InDelta(t, 0.25, c.A, 0.001)
	})

	t.Run("invalid hsl values return nil", func(t *testing.T) {
		t.Parallel()
		for _, in := range []string{
			"hsl(0, 50%)",
			"hsl(abc, 50%, 50%)",
			"hsl(0, x%, 50%)",
			"hsl(0, 50%, x%)",
			"hsla(0, 100%, 50%)",
			"hsla(0, 100%, 50%, zz)",
		} {
			assert.Nil(t, ParseColor(in), "input %q", in)
		}
	})
}

func TestParseColor_HexEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("8-digit hex alpha", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("#11223344")
		require.NotNil(t, c)
		assert.Equal(t, 0x11, c.R)
		assert.Equal(t, 0x22, c.G)
		assert.Equal(t, 0x33, c.B)
		assert.InDelta(t, float64(0x44)/255.0, c.A, 0.001)
	})

	t.Run("4-digit hex shorthand", func(t *testing.T) {
		t.Parallel()
		c := ParseColor("#f008")
		require.NotNil(t, c)
		assert.Equal(t, 255, c.R)
		assert.InDelta(t, float64(0x88)/255.0, c.A, 0.001)
	})

	t.Run("invalid hex strings return nil", func(t *testing.T) {
		t.Parallel()
		for _, in := range []string{
			"#12345",    // bad length
			"#zzz",      // bad digits short form
			"#zz0000",   // bad digits long form
			"#1122334g", // bad alpha digits
			"#gg000011", // bad rgb digits in 8-form
		} {
			assert.Nil(t, ParseColor(in), "input %q", in)
		}
	})
}

// --- calc() edge cases ---

func TestCalc_NegativeNumbers(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 5.0, ParseLength("calc(-5mm + 10mm)", 0), 0.001, "leading negative number")
	assert.InDelta(t, -6.0, ParseLength("calc(-2 * 3mm)", 0), 0.001, "negative bare multiplier")
	assert.InDelta(t, 5.0, ParseLength("calc(10mm - 5mm)", 0), 0.001, "minus as operator after number")
	assert.InDelta(t, 5.0, ParseLength("calc((10mm) - 5mm)", 0), 0.001, "minus as operator after rparen")
}

func TestCalc_MalformedExpressions(t *testing.T) {
	t.Parallel()

	cases := []string{
		"calc(-)",          // bare minus
		"calc(- 5mm)",      // minus separated from number
		"calc(-x)",         // minus followed by junk
		"calc(1mm +)",      // missing right operand
		"calc(* 2mm)",      // operator as factor
		"calc((2mm + 3mm)", // missing rparen (still has calc suffix)
		"calc(10mm @ 2)",   // invalid character
		"calc()",           // empty expression
		"calc((*))",        // invalid expression inside parens
		"calc(2mm * )",     // missing right factor
	}
	for _, in := range cases {
		assert.InDelta(t, 0.0, ParseLength(in, 0), 0.001, "input %q", in)
	}
}

func TestCalc_PercentWithoutContext(t *testing.T) {
	t.Parallel()

	// Without a context width, % inside calc() resolves to 0.
	assert.InDelta(t, 10.0, ParseLength("calc(50% + 10mm)", 0), 0.001)
}

func TestParseLengthCtx_NonCalc(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 100.0, ParseLengthCtx("50%", 0, 200.0), 0.001, "bare percent with context")
	assert.InDelta(t, 0.0, ParseLengthCtx("50%", 0, 0), 0.001, "bare percent without context")
	assert.InDelta(t, 12.0, ParseLengthCtx("12mm", 0, 200.0), 0.001, "plain length")
	assert.InDelta(t, 0.0, ParseLengthCtx("calc(@)", 0, 100.0), 0.001, "malformed calc")
}

// --- Shorthand expansion ---

func TestExpandOne_AllShorthands(t *testing.T) {
	t.Parallel()

	t.Run("border-side shorthands", func(t *testing.T) {
		t.Parallel()
		for _, side := range []string{"top", "right", "bottom", "left"} {
			out := expandOne("border-"+side, "2px dashed blue")
			assert.Equal(t, "2px", out["border-"+side+"-width"], "side %s", side)
			assert.Equal(t, "dashed", out["border-"+side+"-style"], "side %s", side)
			assert.Equal(t, "blue", out["border-"+side+"-color"], "side %s", side)
		}
	})

	t.Run("border expands to twelve longhands", func(t *testing.T) {
		t.Parallel()
		out := expandOne("border", "1px solid red")
		assert.Len(t, out, 12)
		assert.Equal(t, "1px", out["border-top-width"])
		assert.Equal(t, "solid", out["border-left-style"])
		assert.Equal(t, "red", out["border-bottom-color"])
	})

	t.Run("border triple defaults", func(t *testing.T) {
		t.Parallel()
		out := expandOne("border-top", "solid")
		assert.Equal(t, "medium", out["border-top-width"])
		assert.Equal(t, "solid", out["border-top-style"])
		assert.Equal(t, "currentColor", out["border-top-color"])

		out = expandOne("border-top", "2px red")
		assert.Equal(t, "none", out["border-top-style"])
	})

	t.Run("padding and margin box expansion", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			val                      string
			top, right, bottom, left string
		}{
			{"1mm", "1mm", "1mm", "1mm", "1mm"},
			{"1mm 2mm", "1mm", "2mm", "1mm", "2mm"},
			{"1mm 2mm 3mm", "1mm", "2mm", "3mm", "2mm"},
			{"1mm 2mm 3mm 4mm", "1mm", "2mm", "3mm", "4mm"},
			{"1mm 2mm 3mm 4mm 5mm", "0", "0", "0", "0"},
		}
		for _, tc := range cases {
			out := expandOne("padding", tc.val)
			assert.Equal(t, tc.top, out["padding-top"], "val %q", tc.val)
			assert.Equal(t, tc.right, out["padding-right"], "val %q", tc.val)
			assert.Equal(t, tc.bottom, out["padding-bottom"], "val %q", tc.val)
			assert.Equal(t, tc.left, out["padding-left"], "val %q", tc.val)
		}
		out := expandOne("margin", "7mm")
		assert.Equal(t, "7mm", out["margin-top"])
	})

	t.Run("border-radius five values passes through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("border-radius", "1mm 2mm 3mm 4mm 5mm")
		assert.Equal(t, "1mm 2mm 3mm 4mm 5mm", out["border-radius"])
	})

	t.Run("font shorthand", func(t *testing.T) {
		t.Parallel()
		out := expandOne("font", "12pt Times New Roman")
		assert.Equal(t, "12pt", out["font-size"])
		assert.Equal(t, "Times New Roman", out["font-family"])
	})

	t.Run("border-radius single value", func(t *testing.T) {
		t.Parallel()
		out := expandOne("border-radius", "3mm")
		assert.Equal(t, "3mm", out["border-top-left-radius"])
		assert.Equal(t, "3mm", out["border-bottom-right-radius"])
	})

	t.Run("unknown property passes through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("color", "red")
		assert.Equal(t, map[string]string{"color": "red"}, out)
	})
}

func TestExpandFlex_AllForms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		val                 string
		grow, shrink, basis string
	}{
		{"none", "0", "0", "auto"},
		{"auto", "1", "1", "auto"},
		{"initial", "0", "1", "auto"},
		{"2", "2", "1", "0"},
		{"2 auto", "2", "1", "auto"},
		{"2 30%", "2", "1", "30%"},
		{"2 3", "2", "3", "0"},
		{"2 3 10mm", "2", "3", "10mm"},
	}
	for _, tc := range cases {
		out := expandOne("flex", tc.val)
		assert.Equal(t, tc.grow, out["flex-grow"], "flex %q", tc.val)
		assert.Equal(t, tc.shrink, out["flex-shrink"], "flex %q", tc.val)
		assert.Equal(t, tc.basis, out["flex-basis"], "flex %q", tc.val)
	}

	t.Run("too many parts passes through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("flex", "1 2 3 4")
		assert.Equal(t, "1 2 3 4", out["flex"])
	})
}

func TestExpandBackground_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("multi-layer with top-level comma passes through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "red, blue")
		assert.Equal(t, "red, blue", out["background"])
	})

	t.Run("empty value passes through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "  ")
		assert.Equal(t, "", out["background"])
	})

	t.Run("full shorthand with position slash size", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "url(a.png) center / cover no-repeat red")
		assert.Equal(t, "url(a.png)", out["background-image"])
		assert.Equal(t, "center", out["background-position"])
		assert.Equal(t, "cover", out["background-size"])
		assert.Equal(t, "no-repeat", out["background-repeat"])
		assert.Equal(t, "red", out["background-color"])
	})

	t.Run("none sets background-image none", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "none")
		assert.Equal(t, "none", out["background-image"])
	})

	t.Run("box and attachment tokens are accepted but dropped", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "red padding-box border-box content-box scroll fixed local")
		assert.Equal(t, "red", out["background-color"])
		assert.NotContains(t, out, "background-position")
	})

	t.Run("only dropped tokens passes whole value through", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "scroll")
		assert.Equal(t, "scroll", out["background"])
	})

	t.Run("gradient image token", func(t *testing.T) {
		t.Parallel()
		out := expandOne("background", "linear-gradient(to right, red, blue)")
		assert.Equal(t, "linear-gradient(to right, red, blue)", out["background-image"])
	})
}

// --- var() resolution edge cases ---

func TestResolveVars_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("unmatched paren writes remainder verbatim", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("var(--x", map[string]string{"--x": "red"})
		assert.Equal(t, "var(--x", got)
	})

	t.Run("malformed name without dashes uses fallback", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("var(x, blue)", map[string]string{"--x": "red"})
		assert.Equal(t, "blue", got)
	})

	t.Run("malformed name without fallback resolves empty", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("var(x)", nil)
		assert.Equal(t, "", got)
	})

	t.Run("fallback containing commas is preserved", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("var(--missing, 1px, solid)", nil)
		assert.Equal(t, "1px, solid", got)
	})

	t.Run("nil scope with fallback", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("var(--missing, 4mm)", nil)
		assert.Equal(t, "4mm", got)
	})

	t.Run("text after the last var reference is kept", func(t *testing.T) {
		t.Parallel()
		got := ResolveVars("1px var(--s) red", map[string]string{"--s": "solid"})
		assert.Equal(t, "1px solid red", got)
	})
}

// --- Length parsing edge cases ---

func TestParseLength_InvalidNumbers(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.0, ParseLength("abcpx", 0), 0.001)
	assert.InDelta(t, 0.0, ParseLength("12xyz", 0), 0.001)
	assert.InDelta(t, 10.0, ParseLength("1cm", 0), 0.001)
	assert.InDelta(t, 0.0, ParseLength("", 0), 0.001)
}

func TestParsePercentage_Invalid(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"%", "abc%", "50", ""} {
		_, ok := ParsePercentage(in)
		assert.False(t, ok, "input %q", in)
	}
	v, ok := ParsePercentage(" 25% ")
	assert.True(t, ok)
	assert.InDelta(t, 0.25, v, 0.001)
}
