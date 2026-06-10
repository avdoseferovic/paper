package svg

import (
	"math"
	"testing"
)

const maxSVGFuzzInputSize = 64 << 10

func FuzzRasterize(f *testing.F) {
	for _, seed := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" fill="red"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="8"><circle cx="4" cy="4" r="3"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 60"><style>.x{font-size:18px;fill:#123456}</style><text x="8" y="32" class="x">Paper</text></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10000 10000"><path d="M0 0L10 10"/></svg>`,
	} {
		f.Add(seed, 0.0, 0.0)
		f.Add(seed, 10.0, 10.0)
	}

	f.Fuzz(func(t *testing.T, src string, widthMM, heightMM float64) {
		if len(src) > maxSVGFuzzInputSize {
			t.Skip("input too large for smoke fuzzing")
		}
		if invalidFuzzDimension(widthMM) || invalidFuzzDimension(heightMM) {
			t.Skip("dimension outside smoke fuzzing bounds")
		}
		_, _, _, _ = RasterizeWithLimit([]byte(src), widthMM, heightMM, 1_000_000)
	})
}

func invalidFuzzDimension(mm float64) bool {
	return math.IsNaN(mm) || math.IsInf(mm, 0) || math.Abs(mm) > 500
}
