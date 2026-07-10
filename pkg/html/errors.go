package html

import (
	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

type (
	// AssetError indicates a document-referenced asset failed to load while
	// converting HTML with WithStrictAssets enabled.
	AssetError = translate.AssetError

	// LimitError indicates HTML conversion stopped because a configured
	// resource ceiling was crossed. Its Kind field reports which one.
	LimitError = translate.LimitError

	// LimitKind identifies which HTML conversion resource ceiling was exceeded.
	LimitKind = translate.LimitKind
)

const (
	// LimitElements reports that the configured element/node budget was exceeded.
	LimitElements = translate.LimitElements
	// LimitDepth reports that the configured DOM nesting depth was exceeded.
	LimitDepth = translate.LimitDepth
	// LimitStyleRules reports that the configured stylesheet rule budget was exceeded.
	LimitStyleRules = translate.LimitStyleRules
)

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
