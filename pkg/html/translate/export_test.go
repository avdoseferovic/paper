package translate

// Test-only shims. Production code always passes explicit limits, widths and
// run contexts, so the default-argument convenience wrappers live here with the
// tests that use them instead of widening the package's real API.

import (
	"maps"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/html/css"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Exported hooks for white-box testing from the external translate_test package.
var (
	Hamilton                 = layout.Hamilton
	ComputeFlexSizesForWidth = computeFlexSizesForWidth
)

// ComputeFlexSizes allocates gridSize integer cells across flex items using the
// default A4-ish content width.
func ComputeFlexSizes(styles []*css.ComputedStyle, gridSize int) []int {
	return computeFlexSizesForWidth(styles, gridSize, defaultFlexContentWidthMM)
}

// parseStylesheet parses CSS text at the default content width with no limits.
func parseStylesheet(text string) *stylesheet {
	ss, _ := parseStylesheetWithLimits(text, defaultContentWidthMM, htmllimits.NoLimits())
	return ss
}

// parseInlineStyle parses a CSS declaration block (e.g. "color:red; font-size:12pt")
// into a single property→value map, ignoring importance.
func parseInlineStyle(decl string) map[string]string {
	normal, important := parseInlineStyleImportance(decl)
	maps.Copy(normal, important)
	return normal
}

// safeDefaultResolver is safeDefaultResolverWithLimits at the default limits.
func safeDefaultResolver(src string) ([]byte, string, error) {
	return safeDefaultResolverWithLimits(src, htmllimits.Default())
}

// baseDirResolver is baseDirResolverWithLimits at the default limits.
func baseDirResolver(dir string) ImageResolver {
	return baseDirResolverWithLimits(dir, htmllimits.Default())
}

// inlineRuns walks the inline children of a block element with the translator's
// resolvers but no cascaded style.
func (tr *translator) inlineRuns(n *dom.Node) []props.RichRun {
	return inlineRunsWithContext(n, runContext{
		handler:       tr.unsupportedHandler,
		inlineImage:   tr.inlineImage,
		inlinePicture: tr.inlinePicture,
		inlineSVG:     tr.inlineSVG,
	})
}
