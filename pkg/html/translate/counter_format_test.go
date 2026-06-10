package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestFormatAlphaCounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int
		upper bool
		want  string
	}{
		{name: "first lower", value: 1, upper: false, want: "a"},
		{name: "second lower", value: 2, upper: false, want: "b"},
		{name: "last single lower", value: 26, upper: false, want: "z"},
		{name: "first double lower", value: 27, upper: false, want: "aa"},
		{name: "az boundary lower", value: 52, upper: false, want: "az"},
		{name: "ba lower", value: 53, upper: false, want: "ba"},
		{name: "first upper", value: 1, upper: true, want: "A"},
		{name: "last single upper", value: 26, upper: true, want: "Z"},
		{name: "first double upper", value: 27, upper: true, want: "AA"},
		{name: "zero falls back to decimal", value: 0, upper: false, want: "0"},
		{name: "negative falls back to decimal", value: -5, upper: false, want: "-5"},
		{name: "negative falls back to decimal upper", value: -5, upper: true, want: "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, formatAlphaCounter(tt.value, tt.upper))
		})
	}
}
