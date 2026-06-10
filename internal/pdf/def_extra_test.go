package pdf

import "testing"

func TestPointTypeXY(t *testing.T) {
	x, y := PointType{X: 3, Y: 7}.XY()
	if x != 3 || y != 7 {
		t.Fatalf("XY = %v, %v", x, y)
	}
}

func TestPointConversionsRoundTrip(t *testing.T) {
	f := NewCustom(&InitType{UnitStr: "mm"})
	// PointConvert and PointToUnitConvert are aliases.
	if f.PointConvert(72) != f.PointToUnitConvert(72) {
		t.Fatal("PointConvert and PointToUnitConvert disagree")
	}
	// UnitToPointConvert is the inverse of PointConvert.
	pts := 36.0
	units := f.PointConvert(pts)
	if got := f.UnitToPointConvert(units); !floatNear(got, pts, 1e-9) {
		t.Fatalf("round trip = %v, want %v", got, pts)
	}
}

func TestPointConvertPointUnit(t *testing.T) {
	// With unit "pt", k == 1 so conversions are identity.
	f := NewCustom(&InitType{UnitStr: "pt"})
	if got := f.PointConvert(15); got != 15 {
		t.Fatalf("PointConvert in pt = %v", got)
	}
	if got := f.UnitToPointConvert(15); got != 15 {
		t.Fatalf("UnitToPointConvert in pt = %v", got)
	}
}

func TestImageInfoExtentWidthHeight(t *testing.T) {
	info := &ImageInfoType{w: 144, h: 72, scale: 1, dpi: 72}
	// At 72 dpi and scale 1, pixels map 1:1 to points.
	if got := info.Width(); got != 144 {
		t.Errorf("Width = %v", got)
	}
	if got := info.Height(); got != 72 {
		t.Errorf("Height = %v", got)
	}
	w, h := info.Extent()
	if w != 144 || h != 72 {
		t.Errorf("Extent = %v, %v", w, h)
	}
}

func TestImageInfoSetDpiChangesExtent(t *testing.T) {
	info := &ImageInfoType{w: 144, h: 144, scale: 1, dpi: 72}
	info.SetDpi(144)
	// Doubling dpi halves the rendered extent.
	if got := info.Width(); got != 72 {
		t.Fatalf("Width after SetDpi = %v, want 72", got)
	}
}

func TestImageInfoGobRoundTrip(t *testing.T) {
	original := &ImageInfoType{
		data:  []byte{1, 2, 3},
		smask: []byte{4, 5},
		n:     2,
		w:     10,
		h:     20,
		cs:    "DeviceRGB",
		pal:   []byte{9},
		bpc:   8,
		f:     "FlateDecode",
		dp:    "/Predictor 15",
		trns:  []int{1, 2},
		scale: 1,
		dpi:   72,
	}
	encoded, err := original.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode: %v", err)
	}
	var decoded ImageInfoType
	if err := decoded.GobDecode(encoded); err != nil {
		t.Fatalf("GobDecode: %v", err)
	}
	if decoded.w != 10 || decoded.h != 20 || decoded.cs != "DeviceRGB" || decoded.bpc != 8 {
		t.Fatalf("decoded fields mismatch: %+v", decoded)
	}
	// GobDecode computes the checksum id; it must be populated.
	if decoded.i == "" {
		t.Fatal("expected checksum id to be set after decode")
	}
}

func TestImageInfoGobDecodeRejectsGarbage(t *testing.T) {
	var info ImageInfoType
	if err := info.GobDecode([]byte("not gob data")); err == nil {
		t.Fatal("expected error decoding garbage")
	}
}

func floatNear(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}
