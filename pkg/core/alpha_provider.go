package core

// AlphaProvider is a narrow optional capability interface for providers that
// support per-operation alpha (translucency) via gofpdf's SetAlpha. The
// provider MUST save+restore alpha around the fn call (typically via defer)
// so the global alpha state cannot leak into subsequent native rendering.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if ap, ok := provider.(core.AlphaProvider); ok && alpha != nil && *alpha < 1 {
//	    ap.WithAlpha(*alpha, func() { ... })
//	} else {
//	    // direct render — alpha == nil or 1.0 path
//	}
type AlphaProvider interface {
	// WithAlpha runs fn with the global drawing/fill alpha set to a (clamped to
	// [0, 1]), then restores alpha to 1.0 via defer. Panics in fn still restore.
	WithAlpha(a float64, fn func())
}
