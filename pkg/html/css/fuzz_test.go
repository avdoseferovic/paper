package css_test

import (
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/html/css"
)

const maxCSSFuzzInputSize = 64 << 10

func FuzzParseStylesheet(f *testing.F) {
	for _, seed := range []string{
		`p { color: red; padding: 1mm 2mm; }`,
		`.card { border: 1px solid #123456; box-shadow: 0 2mm 4mm #0004; }`,
		`@media print { h1::before { content: "x"; } }`,
		`div { background: linear-gradient(to right, red, blue); }`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, stylesheet string) {
		if len(stylesheet) > maxCSSFuzzInputSize {
			t.Skip("input too large for smoke fuzzing")
		}
		parseCSSValueShapes(stylesheet)
		for _, declarations := range fuzzDeclarationBlocks(stylesheet) {
			applyFuzzDeclarations(declarations)
		}
	})
}

func parseCSSValueShapes(value string) {
	_ = css.ParseColor(value)
	_ = css.ParseLength(value, 12)
	_ = css.ParseLengthCtx(value, 12, 170)
	_, _ = css.ParsePercentage(value)
	_, _ = css.ParseShadow(value)
	_, _ = css.ParseFilterDropShadow(value)
	_, _ = css.ParseLinearGradient(value)
	_, _ = css.ParseRadialGradient(value)
	_, _ = css.ParseConicGradient(value)
	_, _ = css.ParseCSSURL(value)
	_ = css.ApplyTextTransform(value, value)
}

func fuzzDeclarationBlocks(stylesheet string) []map[string]string {
	blocks := []map[string]string{parseFuzzDeclarations(stylesheet)}
	for block := range strings.SplitSeq(stylesheet, "}") {
		_, declarations, ok := strings.Cut(block, "{")
		if !ok {
			continue
		}
		blocks = append(blocks, parseFuzzDeclarations(declarations))
	}
	return blocks
}

func parseFuzzDeclarations(text string) map[string]string {
	declarations := map[string]string{}
	for _, part := range strings.Split(text, ";") {
		property, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		property = strings.TrimSpace(property)
		if property == "" {
			continue
		}
		declarations[property] = strings.TrimSpace(value)
	}
	return declarations
}

func applyFuzzDeclarations(declarations map[string]string) {
	parent := css.NewComputedStyle()
	style := css.NewComputedStyle()
	for property, value := range css.ExpandShorthands(declarations) {
		style.Apply(property, value, parent)
	}
}
