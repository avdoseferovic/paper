package listnum

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestRoman(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		n     int
		upper bool
		want  string
	}{
		"one":              {1, true, "I"},
		"four uses IV":     {4, true, "IV"},
		"nine uses IX":     {9, false, "ix"},
		"fourteen":         {14, true, "XIV"},
		"forty":            {40, false, "xl"},
		"1990":             {1990, true, "MCMXC"},
		"3999 is all ones": {3999, true, "MMMCMXCIX"},
		"above 3999":       {4000, true, "MMMM"},
		"zero":             {0, true, ""},
		"negative":         {-3, true, ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Roman(tc.n, tc.upper))
		})
	}
}

func TestAlpha(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		n     int
		upper bool
		want  string
	}{
		"first":         {1, false, "a"},
		"last single":   {26, false, "z"},
		"wraps to aa":   {27, false, "aa"},
		"28 is ab":      {28, false, "ab"},
		"52 is az":      {52, false, "az"},
		"53 is ba":      {53, false, "ba"},
		"uppercase":     {27, true, "AA"},
		"zero":          {0, false, ""},
		"negative":      {-1, false, ""},
		"three letters": {703, false, "aaa"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Alpha(tc.n, tc.upper))
		})
	}
}
