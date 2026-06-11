package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestCSSGradientToProps(t *testing.T) {
	t.Parallel()

	stops := []css.GradientStop{
		{Color: css.RGBColor{R: 255}, Position: 0},
		{Color: css.RGBColor{B: 255}, Position: 1},
	}

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, cssGradientToProps(nil))
	})

	t.Run("linear maps angle and stops", func(t *testing.T) {
		t.Parallel()
		got := cssGradientToProps(&css.Gradient{
			Kind:   css.GradientLinear,
			Linear: &css.LinearGradient{AngleDeg: 45, Stops: stops},
		})
		require.NotNil(t, got)
		assert.Equal(t, props.GradientLinear, got.Kind)
		assert.Equal(t, 45.0, got.AngleDeg)
		require.Len(t, got.Stops, 2)
		assert.Equal(t, 255, got.Stops[0].Color.Red)
		assert.Equal(t, 255, got.Stops[1].Color.Blue)
	})

	t.Run("radial maps centre and circle flag", func(t *testing.T) {
		t.Parallel()
		got := cssGradientToProps(&css.Gradient{
			Kind:   css.GradientRadial,
			Radial: &css.RadialGradient{Circle: true, CX: 0.25, CY: 0.75, Stops: stops},
		})
		require.NotNil(t, got)
		assert.Equal(t, props.GradientRadial, got.Kind)
		assert.True(t, got.Circle)
		assert.Equal(t, 0.25, got.CX)
		assert.Equal(t, 0.75, got.CY)
		assert.Len(t, got.Stops, 2)
	})

	t.Run("conic maps from-angle and centre", func(t *testing.T) {
		t.Parallel()
		got := cssGradientToProps(&css.Gradient{
			Kind:  css.GradientConic,
			Conic: &css.ConicGradient{FromDeg: 90, CX: 0.5, CY: 0.5, Stops: stops},
		})
		require.NotNil(t, got)
		assert.Equal(t, props.GradientConic, got.Kind)
		assert.Equal(t, 90.0, got.AngleDeg)
		assert.Equal(t, 0.5, got.CX)
		assert.Len(t, got.Stops, 2)
	})

	t.Run("kind without payload yields empty gradient", func(t *testing.T) {
		t.Parallel()
		got := cssGradientToProps(&css.Gradient{Kind: css.GradientLinear})
		require.NotNil(t, got)
		assert.Equal(t, props.GradientLinear, got.Kind)
		assert.Empty(t, got.Stops)
	})
}

func TestMergeCSSFontStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing fontstyle.Type
		weight   string
		style    string
		want     fontstyle.Type
	}{
		{name: "bold weight", existing: fontstyle.Normal, weight: "bold", style: "", want: fontstyle.Bold},
		{name: "italic style", existing: fontstyle.Normal, weight: "", style: "italic", want: fontstyle.Italic},
		{name: "oblique counts as italic", existing: fontstyle.Normal, weight: "", style: "oblique", want: fontstyle.Italic},
		{name: "bold plus italic", existing: fontstyle.Normal, weight: "bold", style: "italic", want: fontstyle.BoldItalic},
		{name: "existing bold gains italic", existing: fontstyle.Bold, weight: "", style: "italic", want: fontstyle.BoldItalic},
		{name: "existing italic gains bold", existing: fontstyle.Italic, weight: "bold", style: "", want: fontstyle.BoldItalic},
		{name: "nothing keeps existing", existing: fontstyle.Normal, weight: "", style: "normal", want: fontstyle.Normal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, mergeCSSFontStyle(tt.existing, tt.weight, tt.style))
		})
	}
}

func TestReadCSSContentString(t *testing.T) {
	t.Parallel()

	t.Run("simple double-quoted string", func(t *testing.T) {
		t.Parallel()
		text, rest, ok := readCSSContentString(`"hello" tail`)
		require.True(t, ok)
		assert.Equal(t, "hello", text)
		assert.Equal(t, " tail", rest)
	})

	t.Run("escaped quote stays literal", func(t *testing.T) {
		t.Parallel()
		text, _, ok := readCSSContentString(`"a\"b"`)
		require.True(t, ok)
		assert.Equal(t, `a"b`, text)
	})

	t.Run("escaped a becomes newline", func(t *testing.T) {
		t.Parallel()
		text, _, ok := readCSSContentString(`"x\ay"`)
		require.True(t, ok)
		assert.Equal(t, "x\ny", text)
	})

	t.Run("unterminated string fails", func(t *testing.T) {
		t.Parallel()
		_, _, ok := readCSSContentString(`"never ends`)
		assert.False(t, ok)
	})

	t.Run("empty input fails", func(t *testing.T) {
		t.Parallel()
		_, _, ok := readCSSContentString("")
		assert.False(t, ok)
	})
}

func TestSplitCSSFunctionArgs(t *testing.T) {
	t.Parallel()

	t.Run("splits on top-level commas", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"a", "b", "c"}, splitCSSFunctionArgs("a, b, c"))
	})

	t.Run("ignores commas inside parens", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"rgb(1,2,3)", "x"}, splitCSSFunctionArgs("rgb(1,2,3), x"))
	})

	t.Run("ignores commas inside quotes", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{`"a,b"`, "c"}, splitCSSFunctionArgs(`"a,b", c`))
	})

	t.Run("handles escaped quote inside string", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{`"a\",b"`, "c"}, splitCSSFunctionArgs(`"a\",b", c`))
	})

	t.Run("drops trailing empty parts", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, []string{"a"}, splitCSSFunctionArgs("a,"))
	})
}

func TestCSSStringArg(t *testing.T) {
	t.Parallel()

	t.Run("quoted string", func(t *testing.T) {
		t.Parallel()
		got, ok := cssStringArg(`"value"`)
		require.True(t, ok)
		assert.Equal(t, "value", got)
	})

	t.Run("quoted string with trailing junk fails", func(t *testing.T) {
		t.Parallel()
		_, ok := cssStringArg(`"value" extra`)
		assert.False(t, ok)
	})

	t.Run("unquoted value passes through", func(t *testing.T) {
		t.Parallel()
		got, ok := cssStringArg("plain")
		require.True(t, ok)
		assert.Equal(t, "plain", got)
	})

	t.Run("empty fails", func(t *testing.T) {
		t.Parallel()
		_, ok := cssStringArg("  ")
		assert.False(t, ok)
	})
}

func TestReadCSSFunction(t *testing.T) {
	t.Parallel()

	t.Run("extracts args and rest", func(t *testing.T) {
		t.Parallel()
		args, rest, ok := readCSSFunction(`counter(sec, decimal) tail`, "counter")
		require.True(t, ok)
		assert.Equal(t, "sec, decimal", args)
		assert.Equal(t, " tail", rest)
	})

	t.Run("name mismatch fails", func(t *testing.T) {
		t.Parallel()
		_, _, ok := readCSSFunction(`counters(sec, ".")`, "counter")
		assert.False(t, ok)
	})

	t.Run("nested parens balanced", func(t *testing.T) {
		t.Parallel()
		args, _, ok := readCSSFunction(`url(calc(1 + 2))`, "url")
		require.True(t, ok)
		assert.Equal(t, "calc(1 + 2)", args)
	})

	t.Run("closing paren inside quotes ignored", func(t *testing.T) {
		t.Parallel()
		args, _, ok := readCSSFunction(`url("a)b")`, "url")
		require.True(t, ok)
		assert.Equal(t, `"a)b"`, args)
	})

	t.Run("escaped quote inside string handled", func(t *testing.T) {
		t.Parallel()
		args, _, ok := readCSSFunction(`url("a\")b")`, "url")
		require.True(t, ok)
		assert.Equal(t, `"a\")b"`, args)
	})

	t.Run("unclosed function fails", func(t *testing.T) {
		t.Parallel()
		_, _, ok := readCSSFunction(`url("x"`, "url")
		assert.False(t, ok)
	})
}

func TestGeneratedContentRuns_CounterFunction(t *testing.T) {
	t.Parallel()

	newCtx := func() runContext {
		c := newCounterState()
		c.reset("sec 7")
		return runContext{counters: c, quotes: newQuoteState()}
	}

	t.Run("counter with style argument", func(t *testing.T) {
		t.Parallel()
		runs, ok := generatedContentRuns(`counter(sec, upper-roman)`, nil, newCtx())
		require.True(t, ok)
		require.Len(t, runs, 1)
		assert.Equal(t, "VII", runs[0].Text)
	})

	t.Run("counter default decimal", func(t *testing.T) {
		t.Parallel()
		runs, ok := generatedContentRuns(`counter(sec)`, nil, newCtx())
		require.True(t, ok)
		require.Len(t, runs, 1)
		assert.Equal(t, "7", runs[0].Text)
	})

	t.Run("empty counter name rejects", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedContentRuns(`counter("")`, nil, newCtx())
		assert.False(t, ok)
	})

	t.Run("unclosed counter rejects", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedContentRuns(`counter(sec`, nil, newCtx())
		assert.False(t, ok)
	})
}
