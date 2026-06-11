package wasmconvert

import "testing"

func TestBuildComponent_PrimaryTypes_MapToRealComponents(t *testing.T) {
	cases := []struct {
		name string
		comp SpecComp
		want int // expected number of components in the cell
	}{
		{"text", SpecComp{Type: "text", Value: "hi"}, 1},
		{"line", SpecComp{Type: "line"}, 1},
		{"qrcode", SpecComp{Type: "qrcode", Value: "x"}, 1},
		{"barcode no label", SpecComp{Type: "barcode", Value: "x"}, 1},
		{"barcode with label", SpecComp{Type: "barcode", Value: "x", Label: "CAP"}, 2},
		{"signature no label", SpecComp{Type: "signature", Value: "A"}, 1},
		{"signature with label", SpecComp{Type: "signature", Value: "A", Label: "ROLE"}, 2},
		{"checkbox", SpecComp{Type: "checkbox", Value: "ok", Checked: true}, 1},
		{"pagenumber", SpecComp{Type: "pagenumber", Value: "Page {n} of {t}"}, 1},
		{"footer", SpecComp{Type: "footer", Value: "footer"}, 1},
		{"table", SpecComp{Type: "table", Head: []string{"A"}, Rows: [][]string{{"1"}}}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildComponent(tc.comp)
			if len(got) != tc.want {
				t.Fatalf("buildComponent(%s) = %d components, want %d", tc.name, len(got), tc.want)
			}
			for i, c := range got {
				if c == nil {
					t.Fatalf("component %d is nil", i)
				}
			}
		})
	}
}

// A labeled barcode caption must carry a non-zero Top offset so it sits below
// the barcode rather than overlapping it (columns do not flow-stack).
func TestBuildComponent_BarcodeCaption_HasTopOffset(t *testing.T) {
	got := buildComponent(SpecComp{Type: "barcode", Value: "x", Label: "CAP"})
	if len(got) != 2 {
		t.Fatalf("expected barcode + caption, got %d components", len(got))
	}
	// prop_top is an internal structure-map key; this is the only host-observable
	// way to assert the caption sits below (not on top of) the barcode.
	caption := got[1].GetStructure().GetData()
	top, ok := caption.Details["prop_top"]
	if !ok {
		t.Fatal("caption text has no prop_top — it would overlap the barcode")
	}
	if v, _ := top.(float64); v <= 0 {
		t.Fatalf("caption prop_top = %v, want > 0", top)
	}
}
