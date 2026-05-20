// Package linestyle contains all line styles.
package linestyle

const (
	// DefaultLineThickness represents the default line style width in gofpdf.
	DefaultLineThickness float64 = 0.2
)

// Type is a representation of a line style style.
type Type string

const (
	// Solid represents a solid style.
	Solid Type = "solid"
	// Dashed represents a dashed style (long segments, [2,1] mm pattern).
	Dashed Type = "dashed"
	// Dotted represents a dotted style (short equal-length dots, [0.4,0.4] mm pattern).
	Dotted Type = "dotted"
)
