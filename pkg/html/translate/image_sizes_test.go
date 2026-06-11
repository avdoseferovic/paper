package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
)

func TestSplitSizesList(t *testing.T) {
	t.Parallel()

	t.Run("splits on top-level commas", func(t *testing.T) {
		t.Parallel()
		got := splitSizesList("(min-width: 100mm) 50vw, 100vw")
		assert.Equal(t, []string{"(min-width: 100mm) 50vw", "100vw"}, got)
	})

	t.Run("ignores commas inside parens", func(t *testing.T) {
		t.Parallel()
		got := splitSizesList("(min-width: 100mm) calc(100vw - 10mm), 50vw")
		assert.Equal(t, []string{"(min-width: 100mm) calc(100vw - 10mm)", "50vw"}, got)
	})

	t.Run("ignores commas inside quotes", func(t *testing.T) {
		t.Parallel()
		got := splitSizesList(`"a,b" 10mm, 20mm`)
		assert.Equal(t, []string{`"a,b" 10mm`, "20mm"}, got)
	})

	t.Run("empty tail dropped", func(t *testing.T) {
		t.Parallel()
		got := splitSizesList("50vw, ")
		assert.Equal(t, []string{"50vw"}, got)
	})

	t.Run("empty input yields nothing", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, splitSizesList(""))
	})
}

func TestLastSourceSizeLengthStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "bare length starts at zero", value: "50vw", want: 0},
		{name: "media condition then length", value: "(min-width: 10mm) 5vw", want: 18},
		{name: "space inside parens ignored", value: "(min-width: 10mm)5vw", want: 0},
		{name: "space inside quotes ignored", value: `"a b"5vw`, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, lastSourceSizeLengthStart(tt.value))
		})
	}
}

func TestParseSourceSize(t *testing.T) {
	t.Parallel()

	const contentWidth = 170.0

	t.Run("bare vw resolves against content width", func(t *testing.T) {
		t.Parallel()
		got, ok := parseSourceSize("50vw", contentWidth)
		require.True(t, ok)
		assert.InDelta(t, 85.0, got, 0.001)
	})

	t.Run("matching media condition uses its length", func(t *testing.T) {
		t.Parallel()
		got, ok := parseSourceSize("(min-width: 10mm) 100mm", contentWidth)
		require.True(t, ok)
		assert.InDelta(t, 100.0, got, 0.001)
	})

	t.Run("non-matching media condition is skipped", func(t *testing.T) {
		t.Parallel()
		_, ok := parseSourceSize("(min-width: 5000mm) 100mm", contentWidth)
		assert.False(t, ok)
	})

	t.Run("auto is rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := parseSourceSize("auto", contentWidth)
		assert.False(t, ok)
	})

	t.Run("empty is rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := parseSourceSize("", contentWidth)
		assert.False(t, ok)
	})

	t.Run("malformed vw is rejected", func(t *testing.T) {
		t.Parallel()
		_, ok := parseSourceSize("abcvw", contentWidth)
		assert.False(t, ok)
	})
}

func TestSourceSizeMM(t *testing.T) {
	t.Parallel()

	t.Run("first matching entry wins", func(t *testing.T) {
		t.Parallel()
		got := sourceSizeMM("(min-width: 5000mm) 10mm, 80mm", 170)
		assert.InDelta(t, 80.0, got, 0.001)
	})

	t.Run("no matching entries yields zero", func(t *testing.T) {
		t.Parallel()
		got := sourceSizeMM("(min-width: 5000mm) 10mm", 170)
		assert.Equal(t, 0.0, got)
	})

	t.Run("zero content width falls back to default", func(t *testing.T) {
		t.Parallel()
		got := sourceSizeMM("100vw", 0)
		assert.Greater(t, got, 0.0)
	})
}

func TestParseSourceSizeLength(t *testing.T) {
	t.Parallel()

	t.Run("percent resolves against content width", func(t *testing.T) {
		t.Parallel()
		got, ok := parseSourceSizeLength("50%", 170)
		require.True(t, ok)
		assert.InDelta(t, 85.0, got, 0.001)
	})

	t.Run("calc is supported", func(t *testing.T) {
		t.Parallel()
		got, ok := parseSourceSizeLength("calc(100% - 70mm)", 170)
		require.True(t, ok)
		assert.InDelta(t, 100.0, got, 0.001)
	})

	t.Run("plain length parses", func(t *testing.T) {
		t.Parallel()
		got, ok := parseSourceSizeLength("25mm", 170)
		require.True(t, ok)
		assert.InDelta(t, 25.0, got, 0.001)
	})
}
