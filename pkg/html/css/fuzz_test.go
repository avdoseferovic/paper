package css_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/css"
)

// FuzzParseValues drives the CSS value grammars that sit behind the rule
// tokenizer. internal/cssparse.FuzzParse stops at the declaration boundary, so
// every parser below is otherwise only exercised with well-formed input even
// though all of it comes from untrusted document content.
func FuzzParseValues(f *testing.F) {
	for _, seed := range []string{
		"",
		"red",
		"#abc",
		"#aabbccdd",
		"rgba(1,2,3,0.5)",
		"rgb(300, -1, 999)",
		"10px",
		"calc(100% - 10px)",
		"calc(((((1)))))",
		"1e400mm",
		"0 0 5px rgba(0,0,0,.5)",
		"inset 1px 1px 1px 1px black",
		"linear-gradient(90deg, red 0%, blue 100%)",
		"radial-gradient(circle at 50% 50%, red, blue)",
		"conic-gradient(from 0deg, red, blue)",
		"url('a.png')",
		"var(--a)",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		// Each of these must tolerate arbitrary input without panicking.
		_ = css.ParseColor(value)
		_ = css.ParseLength(value, 12)
		_ = css.ParseLengthCtx(value, 12, 100)
		_, _ = css.ParseShadow(value)
		_, _ = css.ParseLinearGradient(value)

		// A self-referential custom property is the natural expansion bomb here;
		// ResolveVars must terminate rather than recurse forever.
		scope := map[string]string{"--a": value, "--b": "var(--a)", "--c": "var(--b) var(--c)"}
		_ = css.ResolveVars(value, scope)

		expanded := css.ExpandShorthands(map[string]string{"margin": value, "background": value})
		for prop := range expanded {
			if prop == "" {
				t.Fatalf("ExpandShorthands(%q) produced an empty property name", value)
			}
		}
	})
}
