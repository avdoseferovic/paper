package core

import "github.com/avdoseferovic/paper/v2/pkg/consts/fontstyle"

// LateFontProvider is a narrow optional capability interface for providers
// that can register a font from raw TTF/OTF bytes AFTER initialization.
// Used by the HTML @font-face handler which only discovers font sources at
// translation time (after the provider has already started building the PDF).
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if lfp, ok := provider.(core.LateFontProvider); ok {
//	    lfp.RegisterFont(family, style, bytes)
//	}
type LateFontProvider interface {
	// RegisterFont makes a TTF/OTF font available under the given family and
	// style for subsequent text rendering. Returns immediately; rendering
	// errors (malformed font data) are surfaced by the next text draw call.
	RegisterFont(family string, style fontstyle.Type, bytes []byte)
}
