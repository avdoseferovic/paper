package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormFontWeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "bold keyword", val: "bold", want: "bold"},
		{name: "bolder keyword", val: "bolder", want: "bold"},
		{name: "numeric 700", val: "700", want: "bold"},
		{name: "numeric 800", val: "800", want: "bold"},
		{name: "numeric 900", val: "900", want: "bold"},
		{name: "normal keyword", val: "normal", want: "normal"},
		{name: "lighter keyword", val: "lighter", want: "normal"},
		{name: "numeric 400 below bold threshold", val: "400", want: "normal"},
		{name: "numeric 600 below bold threshold", val: "600", want: "normal"},
		{name: "empty", val: "", want: "normal"},
		{name: "unknown", val: "heavy", want: "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, normFontWeight(tt.val))
		})
	}
}

func TestExpandFont(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want map[string]string
	}{
		{
			name: "size and single-word family",
			val:  "12px Arial",
			want: map[string]string{"font-size": "12px", "font-family": "Arial"},
		},
		{
			name: "size and multi-word family",
			val:  "12pt Times New Roman",
			want: map[string]string{"font-size": "12pt", "font-family": "Times New Roman"},
		},
		{
			name: "size only without family",
			val:  "16px",
			want: map[string]string{"font-size": "16px"},
		},
		{
			name: "no length token passes through unchanged",
			val:  "bold Arial",
			want: map[string]string{"font": "bold Arial"},
		},
		{
			name: "empty passes through unchanged",
			val:  "",
			want: map[string]string{"font": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, expandFont(tt.val))
		})
	}
}
