package html

import "github.com/avdoseferovic/paper/internal/htmllimits"

// Limits caps resource use while translating untrusted HTML.
type Limits = htmllimits.Limits

// DefaultLimits returns the safe resource limits used by FromString and
// FromReader unless overridden.
func DefaultLimits() Limits {
	return htmllimits.Default()
}

// WithLimits overrides resource limits for untrusted HTML translation. Zero
// fields keep their safe default values.
func WithLimits(l Limits) Option {
	return func(c *config) {
		c.limits = htmllimits.Normalize(l)
		c.limitsSet = true
	}
}

// WithUnsafeNoLimits disables resource caps. Use only for trusted input.
func WithUnsafeNoLimits() Option {
	return func(c *config) {
		c.limits = htmllimits.NoLimits()
		c.limitsSet = true
	}
}
