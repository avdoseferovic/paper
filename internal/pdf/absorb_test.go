package pdf

import (
	"errors"
	"testing"
)

func newAbsorbTestPDF(t *testing.T) *PDF {
	t.Helper()

	f := NewCustom(&InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		Size:           SizeType{Wd: 210, Ht: 297},
	})
	f.AddPage()
	return f
}

// TestPagesAbsorbable_DetectsUnspliceableFeaturesAsSoonAsTheyAppear covers the
// check that lets concurrent rendering bail out early: a plain page is
// spliceable, and adding a feature whose PDF names or page references are
// position-dependent must be reported immediately rather than at splice time.
func TestPagesAbsorbable_DetectsUnspliceableFeaturesAsSoonAsTheyAppear(t *testing.T) {
	t.Run("a plain page is spliceable", func(t *testing.T) {
		f := newAbsorbTestPDF(t)
		f.SetFont("arial", "", 12)
		f.Text(10, 10, "plain text")

		if err := f.PagesAbsorbable(); err != nil {
			t.Errorf("expected a plain page to be spliceable, got %v", err)
		}
	})

	t.Run("an outline is reported", func(t *testing.T) {
		f := newAbsorbTestPDF(t)
		f.SetFont("arial", "", 12)
		f.Text(10, 10, "Chapter 1")
		f.Bookmark("Chapter 1", 0, 10)

		if err := f.PagesAbsorbable(); !errors.Is(err, ErrAbsorbUnsupported) {
			t.Errorf("expected ErrAbsorbUnsupported for an outline, got %v", err)
		}
	})

	t.Run("a blend mode is reported", func(t *testing.T) {
		f := newAbsorbTestPDF(t)
		f.SetAlpha(0.5, "Normal")

		if err := f.PagesAbsorbable(); !errors.Is(err, ErrAbsorbUnsupported) {
			t.Errorf("expected ErrAbsorbUnsupported for a blend mode, got %v", err)
		}
	})
}

// TestAbsorbPages_RejectsUnspliceableSource verifies AbsorbPages refuses a
// source document that uses an unspliceable feature, instead of silently
// producing a document with dangling references.
func TestAbsorbPages_RejectsUnspliceableSource(t *testing.T) {
	dst := newAbsorbTestPDF(t)

	src := newAbsorbTestPDF(t)
	src.SetFont("arial", "", 12)
	src.Text(10, 10, "Chapter 1")
	src.Bookmark("Chapter 1", 0, 10)

	if err := dst.AbsorbPages(src); !errors.Is(err, ErrAbsorbUnsupported) {
		t.Errorf("expected ErrAbsorbUnsupported, got %v", err)
	}
}

// TestAbsorbPages_AppendsPagesAndSharesFonts checks the happy path: pages are
// appended in order and a font both documents use is not duplicated.
func TestAbsorbPages_AppendsPagesAndSharesFonts(t *testing.T) {
	dst := newAbsorbTestPDF(t)
	dst.SetFont("arial", "", 12)
	dst.Text(10, 10, "first")

	src := newAbsorbTestPDF(t)
	src.SetFont("arial", "", 12)
	src.Text(10, 10, "second")
	src.AddPage()
	src.Text(10, 20, "third")

	fontsBefore := len(dst.fonts)
	pagesBefore := dst.page

	if err := dst.AbsorbPages(src); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	if got, want := dst.page, pagesBefore+src.page; got != want {
		t.Errorf("page count = %d, want %d", got, want)
	}
	if got := len(dst.fonts); got != fontsBefore {
		t.Errorf("font count = %d, want %d (a shared font must not be duplicated)", got, fontsBefore)
	}
}
