package pdf

import (
	"math"
	"testing"
)

func TestInitDataDefaultA4Size(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})

	if f.Error() != nil {
		t.Fatalf("new pdf: %v", f.Error())
	}
	assertClose(t, f.defPageSize.Wd, 210.0015555555555)
	assertClose(t, f.defPageSize.Ht, 297.0000833333333)
}

func TestInitDataCustomSizeOverride(t *testing.T) {
	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
		Size:           SizeType{Wd: 123, Ht: 45},
	})

	if f.Error() != nil {
		t.Fatalf("new pdf: %v", f.Error())
	}
	if f.defPageSize != (SizeType{Wd: 123, Ht: 45}) {
		t.Fatalf("expected custom default size, got %+v", f.defPageSize)
	}
}

func TestInitDataStandardPageSizesAreCloned(t *testing.T) {
	first := cloneStandardPageSizes()
	first["a4"] = SizeType{Wd: 1, Ht: 1}

	second := cloneStandardPageSizes()

	if second["a4"] == first["a4"] {
		t.Fatal("expected standard page size clones to be independent")
	}
}

func TestInitDataCoreFontSetIsCloned(t *testing.T) {
	first := cloneCoreFontSet()
	delete(first, "helvetica")

	second := cloneCoreFontSet()

	if !second["helvetica"] {
		t.Fatal("expected core font set clones to be independent")
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("expected %f, got %f", want, got)
	}
}
