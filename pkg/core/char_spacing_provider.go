package core

// CharSpacingProvider is a narrow optional capability interface for providers
// that support per-run character spacing (CSS letter-spacing). Implementations
// MUST always restore character spacing to 0 via defer so state cannot leak
// into subsequent rendering.
//
// Note: the current phpdave11/gofpdf fork does not expose SetCharSpacing —
// only SetWordSpacing. The internal paper provider therefore implements this interface
// as a no-op (fn is invoked without any spacing adjustment). The interface and
// type assertion path are still in place so a future fork swap or upstream
// change can light up the feature without further wiring changes.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if csp, ok := provider.(core.CharSpacingProvider); ok && run.LetterSpacing > 0 {
//	    csp.WithCharSpacing(run.LetterSpacing, func() { ... })
//	} else {
//	    // direct render path
//	}
type CharSpacingProvider interface {
	// WithCharSpacing runs fn with the per-run character spacing temporarily
	// set to mm, restoring 0 via defer (panic-safe). mm < 0 should be clamped
	// to 0 by the implementation.
	WithCharSpacing(mm float64, fn func())
}
