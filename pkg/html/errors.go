package html

import "github.com/avdoseferovic/paper/internal/htmllimits"

var (
	// ErrImageTooLarge is returned when an HTML image exceeds configured byte
	// or pixel limits.
	ErrImageTooLarge = htmllimits.ErrImageTooLarge

	// ErrDOMTooDeep is returned when the parsed DOM exceeds MaxDOMDepth.
	ErrDOMTooDeep = htmllimits.ErrDOMTooDeep

	// ErrDOMTooLarge is returned when the parsed DOM exceeds MaxDOMNodes.
	ErrDOMTooLarge = htmllimits.ErrDOMTooLarge

	// ErrSVGTooLarge is returned when SVG rasterization exceeds MaxSVGPixels.
	ErrSVGTooLarge = htmllimits.ErrSVGTooLarge

	// ErrStyleRulesTooLarge is returned when CSS parsing exceeds MaxStyleRules.
	ErrStyleRulesTooLarge = htmllimits.ErrStyleRulesTooLarge
)
