package pdf

import (
	"crypto/sha256"
	"fmt"
)

// blendModeType is one /ExtGState entry: a stroke and fill alpha plus a
// blending mode.
type blendModeType struct {
	strokeStr, fillStr, modeStr string
	id                          string // content hash naming it in content streams
	objNum                      int
}

// gradientType is one /Shading entry.
type gradientType struct {
	tp                int // 2: linear, 3: radial
	clr1Str, clr2Str  string
	x1, y1, x2, y2, r float64
	id                string // content hash naming it in content streams
	objNum            int
}

// generateBlendModeID names a blend mode by a hash of the values it emits,
// mirroring generateFontID. Naming by content rather than by position in
// blendList is what lets two documents that registered the same state merge:
// their content streams already agree on the name, and the definitions
// deduplicate. objNum is excluded because it is assigned at serialization.
func generateBlendModeID(blend blendModeType) string {
	return hashResourceKey(fmt.Sprintf("%s|%s|%s", blend.strokeStr, blend.fillStr, blend.modeStr))
}

// generateGradientID names a gradient by a hash of its shading parameters. See
// generateBlendModeID.
func generateGradientID(gradient gradientType) string {
	return hashResourceKey(fmt.Sprintf("%d|%s|%s|%.5f|%.5f|%.5f|%.5f|%.5f",
		gradient.tp, gradient.clr1Str, gradient.clr2Str,
		gradient.x1, gradient.y1, gradient.x2, gradient.y2, gradient.r))
}

func hashResourceKey(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
