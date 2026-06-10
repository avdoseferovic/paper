package translate

import (
	"github.com/avdoseferovic/paper/internal/htmllimits"
)

// Option configures translator behaviour.
type Option func(*translator)

// WithGridSize overrides the default 12-column grid size used for flex quantization.
func WithGridSize(n int) Option {
	return func(tr *translator) {
		if n > 0 {
			tr.gridSize = n
		}
	}
}

// WithContentWidth sets the content width in mm, used for gap-to-col approximation.
func WithContentWidth(mm float64) Option {
	return func(tr *translator) {
		if mm > 0 {
			tr.contentWidthMM = mm
		}
	}
}

// WithLimits configures resource limits for untrusted HTML translation.
func WithLimits(l htmllimits.Limits) Option {
	return func(tr *translator) {
		tr.limits = htmllimits.Normalize(l)
	}
}

// WithStylesheetBaseDir scopes the default stylesheet resolver to a single
// directory. Local-file reads outside this directory are refused.
func WithStylesheetBaseDir(dir string) Option {
	return func(tr *translator) { tr.stylesheetResolver = stylesheetBaseDirResolver(dir) }
}

// WithImageResolver lets callers plug in a custom <img src=…> loader.
func WithImageResolver(fn ImageResolver) Option {
	return func(tr *translator) {
		tr.imageResolver = fn
	}
}

// WithImageBaseDir scopes the default resolver to a single directory.
// Local file reads outside this directory are refused (path-traversal safe).
func WithImageBaseDir(dir string) Option {
	return func(tr *translator) {
		tr.imageBaseDir = dir
	}
}

// WithUnsupportedHandler registers a callback for unsupported tags/props.
func WithUnsupportedHandler(fn func(thing, value string)) Option {
	return func(tr *translator) {
		tr.unsupportedHandler = fn
	}
}

// WithOutlineFromHeadings marks h1-h6 headings with a props.Outline so they
// appear in the PDF document outline (h1 = level 0 ... h6 = level 5).
func WithOutlineFromHeadings() Option {
	return func(tr *translator) {
		tr.outlineFromHeadings = true
	}
}

// headingOutlineLevel maps an h1-h6 tag to its outline nesting level.
func headingOutlineLevel(tag string) (int, bool) {
	switch tag {
	case "h1":
		return 0, true
	case "h2":
		return 1, true
	case "h3":
		return 2, true
	case "h4":
		return 3, true
	case "h5":
		return 4, true
	case "h6":
		return 5, true
	default:
		return 0, false
	}
}
