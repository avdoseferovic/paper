package html_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/html"
)

// FuzzFromString drives the whole HTML-to-rows path with arbitrary markup.
// FromString is the public entry point for untrusted documents, and it is where
// the DOM-depth and stylesheet-rule limits are enforced, so the contract is
// that any input either translates or returns an error — never panics and never
// runs away building a document.
func FuzzFromString(f *testing.F) {
	for _, seed := range []string{
		"",
		"<p>hello</p>",
		"<h1>Title</h1><p>Body <strong>bold</strong></p>",
		`<style>p { color: red } @media print { a { color: blue } }</style><p>x</p>`,
		`<table><tr><td colspan="3">a</td></tr></table>`,
		`<div style="display:flex"><div style="flex:1">a</div><div>b</div></div>`,
		`<ul><li>a<ul><li>b<ul><li>c</li></ul></li></ul></li></ul>`,
		"<div><div><div><div><div><div>deep</div></div></div></div></div></div>",
		`<img src="../../../etc/passwd">`,
		`<style>:root{--a:var(--b);--b:var(--a)}</style><p style="color:var(--a)">x</p>`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, markup string) {
		rows, err := html.FromString(t.Context(), markup)
		if err != nil {
			if rows != nil {
				t.Fatalf("FromString() returned both rows and an error: %v", err)
			}
			return
		}
		for i, row := range rows {
			if row == nil {
				t.Fatalf("FromString() returned a nil row at index %d", i)
			}
		}
	})
}
