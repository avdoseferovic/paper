package translate

import (
	"testing"
)

func declsOf(pairs ...string) []cssDeclaration {
	decls := make([]cssDeclaration, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		decls = append(decls, cssDeclaration{property: pairs[i], value: pairs[i+1]})
	}
	return decls
}

func TestParseCSS_WhenPageRule_ShouldCaptureDeclarations(t *testing.T) {
	t.Parallel()

	rules := parseCSS("@page { size: A4; margin: 10mm }")

	if len(rules) != 1 || rules[0].kind != atRule || rules[0].name != "@page" {
		t.Fatalf("expected one @page at-rule, got %+v", rules)
	}
	if len(rules[0].declarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(rules[0].declarations))
	}
}

func TestPageOptionsFromDecls_WhenNamedSizeAndLandscape_ShouldSetBoth(t *testing.T) {
	t.Parallel()

	opts := pageOptionsFromDecls(declsOf("size", "A5 landscape"))

	if opts == nil || opts.PageSize != "a5" || !opts.Landscape {
		t.Fatalf("expected a5 landscape, got %+v", opts)
	}
}

func TestPageOptionsFromDecls_WhenExplicitDimensions_ShouldSetWidthHeight(t *testing.T) {
	t.Parallel()

	opts := pageOptionsFromDecls(declsOf("size", "200mm 100mm"))

	if opts == nil || opts.Width != 200 || opts.Height != 100 || opts.PageSize != "" {
		t.Fatalf("expected 200x100, got %+v", opts)
	}
}

func TestPageOptionsFromDecls_WhenMarginShorthand_ShouldExpandPerCSSOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value                    string
		top, right, bottom, left float64
	}{
		{"10mm", 10, 10, 10, 10},
		{"10mm 20mm", 10, 20, 10, 20},
		{"10mm 20mm 30mm", 10, 20, 30, 20},
		{"10mm 20mm 30mm 40mm", 10, 20, 30, 40},
	}
	for _, tc := range cases {
		opts := pageOptionsFromDecls(declsOf("margin", tc.value))
		if opts == nil {
			t.Fatalf("margin %q: expected options", tc.value)
		}
		if opts.MarginTop != tc.top || opts.MarginRight != tc.right ||
			opts.MarginBottom != tc.bottom || opts.MarginLeft != tc.left {
			t.Errorf("margin %q = T%v R%v B%v L%v, want T%v R%v B%v L%v", tc.value,
				opts.MarginTop, opts.MarginRight, opts.MarginBottom, opts.MarginLeft,
				tc.top, tc.right, tc.bottom, tc.left)
		}
	}
}

func TestPageOptionsFromDecls_WhenIndividualMarginOverridesShorthand_ShouldWin(t *testing.T) {
	t.Parallel()

	opts := pageOptionsFromDecls(declsOf("margin", "10mm", "margin-left", "25mm"))

	if opts == nil || opts.MarginLeft != 25 || opts.MarginTop != 10 {
		t.Fatalf("expected left=25 top=10, got %+v", opts)
	}
}

func TestPageOptionsFromDecls_WhenNothingUsable_ShouldReturnNil(t *testing.T) {
	t.Parallel()

	if opts := pageOptionsFromDecls(declsOf("bleed", "3mm")); opts != nil {
		t.Fatalf("expected nil for unsupported-only declarations, got %+v", opts)
	}
	if opts := pageOptionsFromDecls(nil); opts != nil {
		t.Fatalf("expected nil for empty declarations, got %+v", opts)
	}
}

func TestParseStylesheet_WhenPseudoPageRule_ShouldRecordSkipped(t *testing.T) {
	t.Parallel()

	ss := parseStylesheet("@page :first { margin: 0 }\n@page { margin: 5mm }")

	if len(ss.skippedPages) != 1 || ss.skippedPages[0] != "@page :first" {
		t.Fatalf("expected one skipped pseudo page, got %+v", ss.skippedPages)
	}
	if len(ss.pageDecls) != 1 {
		t.Fatalf("expected plain @page decls collected, got %+v", ss.pageDecls)
	}
}
