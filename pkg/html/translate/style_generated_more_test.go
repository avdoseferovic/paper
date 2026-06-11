package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestCSSShadowToProps(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, cssShadowToProps(nil))
	})

	t.Run("maps all fields with color", func(t *testing.T) {
		t.Parallel()
		got := cssShadowToProps(&css.Shadow{
			OffsetX:    1,
			OffsetY:    2,
			BlurRadius: 3,
			Spread:     4,
			Inset:      true,
			Color:      &css.RGBColor{R: 10, G: 20, B: 30},
		})
		require.NotNil(t, got)
		assert.Equal(t, 1.0, got.OffsetX)
		assert.Equal(t, 2.0, got.OffsetY)
		assert.Equal(t, 3.0, got.BlurRadius)
		assert.Equal(t, 4.0, got.Spread)
		assert.True(t, got.Inset)
		require.NotNil(t, got.Color)
		assert.Equal(t, 10, got.Color.Red)
		assert.Equal(t, 20, got.Color.Green)
		assert.Equal(t, 30, got.Color.Blue)
	})

	t.Run("missing color stays nil", func(t *testing.T) {
		t.Parallel()
		got := cssShadowToProps(&css.Shadow{OffsetX: 1})
		require.NotNil(t, got)
		assert.Nil(t, got.Color)
	})
}

func TestIsDisplayNone(t *testing.T) {
	t.Parallel()

	parseFirst := func(t *testing.T, src, tag string) *dom.Node {
		t.Helper()
		doc, err := dom.Parse(src)
		require.NoError(t, err)
		n := findNode(doc, tag)
		require.NotNil(t, n)
		return n
	}

	t.Run("nil node is visible", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isDisplayNone(nil))
	})

	t.Run("hidden attribute hides", func(t *testing.T) {
		t.Parallel()
		n := parseFirst(t, `<html><body><p hidden="hidden">x</p></body></html>`, "p")
		assert.True(t, isDisplayNone(n))
	})

	t.Run("display:none hides", func(t *testing.T) {
		t.Parallel()
		n := parseFirst(t, `<html><body><p style="display:none">x</p></body></html>`, "p")
		assert.True(t, isDisplayNone(n))
	})

	t.Run("display: none with space hides", func(t *testing.T) {
		t.Parallel()
		n := parseFirst(t, `<html><body><p style="display: none">x</p></body></html>`, "p")
		assert.True(t, isDisplayNone(n))
	})

	t.Run("plain node is visible", func(t *testing.T) {
		t.Parallel()
		n := parseFirst(t, `<html><body><p>x</p></body></html>`, "p")
		assert.False(t, isDisplayNone(n))
	})
}

func TestFormatCounterValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int
		style string
		want  string
	}{
		{name: "empty style is decimal", value: 7, style: "", want: "7"},
		{name: "decimal", value: 42, style: "decimal", want: "42"},
		{name: "decimal-leading-zero pads single digit", value: 5, style: "decimal-leading-zero", want: "05"},
		{name: "decimal-leading-zero pads negative single digit", value: -5, style: "decimal-leading-zero", want: "-05"},
		{name: "decimal-leading-zero leaves two digits", value: 15, style: "decimal-leading-zero", want: "15"},
		{name: "lower-alpha first", value: 1, style: "lower-alpha", want: "a"},
		{name: "lower-alpha wraps to aa", value: 27, style: "lower-latin", want: "aa"},
		{name: "upper-alpha", value: 2, style: "upper-alpha", want: "B"},
		{name: "alpha non-positive falls back to digits", value: 0, style: "lower-alpha", want: "0"},
		{name: "lower-roman", value: 4, style: "lower-roman", want: "iv"},
		{name: "upper-roman", value: 1999, style: "upper-roman", want: "MCMXCIX"},
		{name: "roman zero falls back to digits", value: 0, style: "lower-roman", want: "0"},
		{name: "roman over 3999 falls back to digits", value: 4000, style: "upper-roman", want: "4000"},
		{name: "unknown style falls back to decimal", value: 9, style: "fancy", want: "9"},
		{name: "quoted style is trimmed", value: 3, style: `"lower-roman"`, want: "iii"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatCounterValue(tt.value, tt.style))
		})
	}
}

func TestCounterState_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()
	var c *counterState

	assert.Equal(t, 0, c.value("x"))
	assert.Equal(t, []int{0}, c.allValues("x"))
	assert.Nil(t, c.enter(&css.ComputedStyle{CounterReset: "sec 1"}))
	c.exit([]string{"sec"}) // must not panic
}

func TestCounterState_AllValues(t *testing.T) {
	t.Parallel()

	t.Run("missing counter yields single zero", func(t *testing.T) {
		t.Parallel()
		c := newCounterState()
		assert.Equal(t, []int{0}, c.allValues("missing"))
	})

	t.Run("nested resets yield full stack copy", func(t *testing.T) {
		t.Parallel()
		c := newCounterState()
		c.reset("sec 1")
		c.reset("sec 2")
		got := c.allValues("sec")
		assert.Equal(t, []int{1, 2}, got)

		// Mutating the copy must not affect internal state.
		got[0] = 99
		assert.Equal(t, []int{1, 2}, c.allValues("sec"))
	})
}

func TestGeneratedCountersText(t *testing.T) {
	t.Parallel()

	nested := func() *counterState {
		c := newCounterState()
		c.reset("sec 1")
		c.reset("sec 2")
		c.reset("sec 3")
		return c
	}

	t.Run("joins nested values with separator", func(t *testing.T) {
		t.Parallel()
		got, ok := generatedCountersText(`sec, "."`, nested())
		require.True(t, ok)
		assert.Equal(t, "1.2.3", got)
	})

	t.Run("applies counter style", func(t *testing.T) {
		t.Parallel()
		got, ok := generatedCountersText(`sec, "-", lower-roman`, nested())
		require.True(t, ok)
		assert.Equal(t, "i-ii-iii", got)
	})

	t.Run("missing counter renders zero", func(t *testing.T) {
		t.Parallel()
		got, ok := generatedCountersText(`other, "."`, newCounterState())
		require.True(t, ok)
		assert.Equal(t, "0", got)
	})

	t.Run("single argument fails", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedCountersText(`sec`, newCounterState())
		assert.False(t, ok)
	})

	t.Run("empty name fails", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedCountersText(`"", "."`, newCounterState())
		assert.False(t, ok)
	})

	t.Run("unterminated separator string fails", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedCountersText(`sec, "unterminated`, newCounterState())
		assert.False(t, ok)
	})
}

func TestGeneratedContentRuns_CountersFunction(t *testing.T) {
	t.Parallel()

	c := newCounterState()
	c.reset("sec 1")
	c.reset("sec 4")
	ctx := runContext{counters: c, quotes: newQuoteState()}

	t.Run("counters plus string literal in one run", func(t *testing.T) {
		t.Parallel()
		runs, ok := generatedContentRuns(`counters(sec, ".") ":"`, nil, ctx)
		require.True(t, ok)
		require.Len(t, runs, 1)
		assert.Equal(t, "1.4:", runs[0].Text)
	})

	t.Run("malformed counters rejects the whole value", func(t *testing.T) {
		t.Parallel()
		runs, ok := generatedContentRuns(`counters(sec)`, nil, ctx)
		assert.False(t, ok)
		assert.Nil(t, runs)
	})

	t.Run("unclosed counters function rejects", func(t *testing.T) {
		t.Parallel()
		_, ok := generatedContentRuns(`counters(sec, "."`, nil, ctx)
		assert.False(t, ok)
	})
}

func TestGeneratedContentRuns_QuoteSuppressionKeywords(t *testing.T) {
	t.Parallel()

	t.Run("no-open-quote advances depth silently", func(t *testing.T) {
		t.Parallel()
		ctx := runContext{counters: newCounterState(), quotes: newQuoteState()}
		// no-open-quote bumps depth, so close-quote pops back to the outer pair.
		runs, ok := generatedContentRuns(`no-open-quote "x" close-quote`, nil, ctx)
		require.True(t, ok)
		require.Len(t, runs, 1)
		assert.Equal(t, `x"`, runs[0].Text)
	})

	t.Run("no-close-quote keeps depth without output", func(t *testing.T) {
		t.Parallel()
		ctx := runContext{counters: newCounterState(), quotes: newQuoteState()}
		// open-quote at depth 0 → " ; no-close-quote at depth 1 → pops to 0,
		// so the following open-quote uses the primary pair again.
		runs, ok := generatedContentRuns(`open-quote no-close-quote open-quote`, nil, ctx)
		require.True(t, ok)
		require.Len(t, runs, 1)
		assert.Equal(t, `""`, runs[0].Text)
	})
}

func TestQuoteState_NoOpenNoClose(t *testing.T) {
	t.Parallel()

	t.Run("noOpen increments depth", func(t *testing.T) {
		t.Parallel()
		q := newQuoteState()
		q.noOpen()
		assert.Equal(t, 1, q.depth)
	})

	t.Run("noClose decrements but never below zero", func(t *testing.T) {
		t.Parallel()
		q := newQuoteState()
		q.noClose()
		assert.Equal(t, 0, q.depth)
		q.noOpen()
		q.noClose()
		assert.Equal(t, 0, q.depth)
	})

	t.Run("nil receiver is safe", func(t *testing.T) {
		t.Parallel()
		var q *quoteState
		q.noOpen()
		q.noClose()
	})
}
