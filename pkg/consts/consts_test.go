package consts_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/consts"
)

// The string values of these constants are serialized into PDFs and golden
// structure files; they are the compatibility contract of the consolidation
// (Go identifiers changed, values must not).
func TestConsts_StringValuesAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"AlignLeft", string(consts.AlignLeft), "L"},
		{"AlignRight", string(consts.AlignRight), "R"},
		{"AlignCenter", string(consts.AlignCenter), "C"},
		{"AlignTop", string(consts.AlignTop), "T"},
		{"AlignBottom", string(consts.AlignBottom), "B"},
		{"AlignMiddle", string(consts.AlignMiddle), "M"},
		{"AlignJustify", string(consts.AlignJustify), "J"},
		{"OrientationVertical", string(consts.OrientationVertical), "vertical"},
		{"OrientationHorizontal", string(consts.OrientationHorizontal), "horizontal"},
		{"LineStyleSolid", string(consts.LineStyleSolid), "solid"},
		{"LineStyleDashed", string(consts.LineStyleDashed), "dashed"},
		{"LineStyleDotted", string(consts.LineStyleDotted), "dotted"},
		{"BreakLineEmptySpace", string(consts.BreakLineEmptySpace), "empty_space_strategy"},
		{"BreakLineDash", string(consts.BreakLineDash), "dash_strategy"},
		{"FontFamilyArial", consts.FontFamilyArial, "arial"},
		{"FontFamilyHelvetica", consts.FontFamilyHelvetica, "helvetica"},
		{"FontFamilySymbol", consts.FontFamilySymbol, "symbol"},
		{"FontFamilyZapBats", consts.FontFamilyZapBats, "zapfdingbats"},
		{"FontFamilyCourier", consts.FontFamilyCourier, "courier"},
		{"BarcodeCode128", string(consts.BarcodeCode128), "code128"},
		{"BarcodeEAN", string(consts.BarcodeEAN), "ean"},
		{"GenerationSequential", string(consts.GenerationSequential), "sequential"},
		{"GenerationConcurrent", string(consts.GenerationConcurrent), "concurrent"},
		{"GenerationSequentialLowMemory", string(consts.GenerationSequentialLowMemory), "sequential_low_memory"},
		{"ProviderPaper", string(consts.ProviderPaper), "paper"},
	}

	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestConsts_DefaultLineThickness(t *testing.T) {
	t.Parallel()

	if consts.DefaultLineThickness != 0.2 {
		t.Errorf("DefaultLineThickness = %v, want 0.2", consts.DefaultLineThickness)
	}
}
