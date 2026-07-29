package cssparse

import (
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	rules, err := Parse(`
/* ignored */
@font-face { font-family: "My Font"; src: url("font,regular.ttf") format("truetype") }
@media print and (min-width: 200px) {
  p, a[href=",one"] { color: red !IMPORTANT; content: "a;b" }
}
@import url("later.css");
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("Parse() rule count = %d, want 3", len(rules))
	}
	if got := rules[0]; got.Kind != AtRule || got.Name != "@font-face" || len(got.Declarations) != 2 {
		t.Fatalf("font face rule = %#v", got)
	}
	if got, want := rules[0].Declarations[1].Value, `url("font,regular.ttf") format("truetype")`; got != want {
		t.Fatalf("font source = %q, want %q", got, want)
	}
	if got := rules[1]; got.Kind != AtRule || got.Name != "@media" || len(got.Rules) != 1 {
		t.Fatalf("media rule = %#v", got)
	}
	if got, want := rules[1].Rules[0].Selectors, []string{"p", `a[href=",one"]`}; !sameStrings(got, want) {
		t.Fatalf("selectors = %#v, want %#v", got, want)
	}
	if declaration := rules[1].Rules[0].Declarations[0]; declaration.Property != "color" || declaration.Value != "red" || !declaration.Important {
		t.Fatalf("important declaration = %#v", declaration)
	}
	if got := rules[2]; got.Kind != AtRule || got.Name != "@import" || got.Prelude != `url("later.css")` {
		t.Fatalf("import rule = %#v", got)
	}
}

func TestParseWithMaxRules(t *testing.T) {
	t.Parallel()

	rules, err := ParseWithMaxRules("p { color: red } a { color: blue }", 1)
	if err == nil {
		t.Fatal("ParseWithMaxRules() error = nil, want limit error")
	}
	if len(rules) != 1 {
		t.Fatalf("ParseWithMaxRules() rules = %d, want the completed first rule", len(rules))
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"p { color: red }",
		"@media print { a[href=\",\"] { content: \";{}\" } }",
		"/* unterminated",
		"a { --value: { nested: [value] } }",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		rules, _ := ParseWithMaxRules(text, 100)
		if len(rules) > 100 {
			t.Fatalf("ParseWithMaxRules() returned %d rules", len(rules))
		}
	})
}

func sameStrings(got, want []string) bool {
	return slices.Equal(got, want)
}
