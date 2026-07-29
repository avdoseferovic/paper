package pdf

import (
	"bytes"
	"errors"
	"strings"
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

// TestPagesAbsorbable_AcceptsPageLevelFeatures covers the check that lets
// concurrent rendering bail out early. Everything a page can draw is
// spliceable, because it is named by content or rewritten at splice time; only
// document-catalog features, which are written once for the whole document, are
// reported.
func TestPagesAbsorbable_AcceptsPageLevelFeatures(t *testing.T) {
	t.Parallel()

	spliceable := map[string]func(f *PDF){
		"a plain page": func(f *PDF) {
			f.SetFont("arial", "", 12)
			f.Text(10, 10, "plain text")
		},
		"an outline": func(f *PDF) {
			f.SetFont("arial", "", 12)
			f.Text(10, 10, "Chapter 1")
			f.Bookmark("Chapter 1", 0, 10)
		},
		"a blend mode": func(f *PDF) {
			f.SetAlpha(0.5, "Normal")
		},
		"a gradient": func(f *PDF) {
			f.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)
		},
		"an external link": func(f *PDF) {
			f.LinkString(10, 10, 20, 5, "https://example.com")
		},
		"an internal link": func(f *PDF) {
			link := f.AddLink()
			f.SetLink(link, 20, 1)
			f.Link(10, 10, 20, 5, link)
		},
	}

	for name, draw := range spliceable {
		t.Run(name+" is spliceable", func(t *testing.T) {
			t.Parallel()
			f := newAbsorbTestPDF(t)
			draw(f)

			if err := f.PagesAbsorbable(); err != nil {
				t.Errorf("expected %s to be spliceable, got %v", name, err)
			}
		})
	}

	// Defence in depth: paper routes documents configured with a catalog
	// feature to single-document generation before a mode is chosen, so this
	// is unreachable through the public API. It is what turns a newly added
	// position-dependent feature into a sequential fallback rather than a
	// silently corrupt document.
	t.Run("a document-catalog feature is reported", func(t *testing.T) {
		f := newAbsorbTestPDF(t)
		f.SetPageAnnotations(PageAnnotation{PageIndex: 0, Subtype: "Text", Contents: "note"})

		if err := f.PagesAbsorbable(); !errors.Is(err, ErrAbsorbUnsupported) {
			t.Errorf("expected ErrAbsorbUnsupported for a page annotation, got %v", err)
		}
	})
}

// TestAbsorbPages_RejectsUnspliceableSource verifies AbsorbPages refuses a
// source document that uses an unspliceable feature, instead of silently
// producing a document that has lost it.
func TestAbsorbPages_RejectsUnspliceableSource(t *testing.T) {
	t.Parallel()

	dst := newAbsorbTestPDF(t)

	src := newAbsorbTestPDF(t)
	src.SetPageAnnotations(PageAnnotation{PageIndex: 0, Subtype: "Text", Contents: "note"})

	if err := dst.AbsorbPages(src); !errors.Is(err, ErrAbsorbUnsupported) {
		t.Errorf("expected ErrAbsorbUnsupported, got %v", err)
	}
}

// TestAbsorbPages_AppendsPagesAndSharesFonts checks the happy path: pages are
// appended in order and a font both documents use is not duplicated.
func TestAbsorbPages_AppendsPagesAndSharesFonts(t *testing.T) {
	t.Parallel()

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

// TestAbsorbPages_CarriesExternalLinks verifies that a URL link drawn in the
// source lands on the page it was drawn on, with its rectangle intact. External
// links hold no page reference, so nothing needs rewriting — but the entries do
// need carrying across, which the previous implementation did not do.
func TestAbsorbPages_CarriesExternalLinks(t *testing.T) {
	t.Parallel()

	dst := newAbsorbTestPDF(t)

	src := newAbsorbTestPDF(t)
	src.LinkString(10, 20, 30, 5, "https://example.com/first")
	src.AddPage()
	src.LinkString(15, 25, 35, 6, "https://example.com/second")

	if err := dst.AbsorbPages(src); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	if got := len(dst.pageLinks[1]); got != 0 {
		t.Errorf("destination's own page has %d links, want 0", got)
	}
	assertSingleExternalLink(t, dst.pageLinks, 2, "https://example.com/first")
	assertSingleExternalLink(t, dst.pageLinks, 3, "https://example.com/second")
}

func assertSingleExternalLink(t *testing.T, pageLinks [][]linkType, page int, want string) {
	t.Helper()

	if page >= len(pageLinks) {
		t.Fatalf("page %d has no link slice", page)
	}
	links := pageLinks[page]
	if len(links) != 1 {
		t.Fatalf("page %d has %d links, want 1", page, len(links))
	}
	if links[0].linkStr != want {
		t.Errorf("page %d link target = %q, want %q", page, links[0].linkStr, want)
	}
	if links[0].link != 0 {
		t.Errorf("page %d link has internal target %d, want 0", page, links[0].link)
	}
	if links[0].wd <= 0 || links[0].ht <= 0 {
		t.Errorf("page %d link rectangle = %.2fx%.2f, want a non-empty box", page, links[0].wd, links[0].ht)
	}
}

// TestAbsorbPages_ShiftsOutlinePages verifies bookmarks point at the page they
// were placed on after that page has moved to its position in the merged
// document.
func TestAbsorbPages_ShiftsOutlinePages(t *testing.T) {
	t.Parallel()

	dst := newAbsorbTestPDF(t)
	dst.SetFont("arial", "", 12)
	dst.Bookmark("Chapter 1", 0, 10)
	dst.AddPage()
	dst.Bookmark("Chapter 2", 0, 10)

	src := newAbsorbTestPDF(t)
	src.SetFont("arial", "", 12)
	src.Bookmark("Chapter 3", 0, 10)
	src.AddPage()
	src.Bookmark("Chapter 4", 0, 10)

	if err := dst.AbsorbPages(src); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	wantPages := []int{1, 2, 3, 4}
	if len(dst.outlines) != len(wantPages) {
		t.Fatalf("outline count = %d, want %d", len(dst.outlines), len(wantPages))
	}
	for i, want := range wantPages {
		if got := dst.outlines[i].p; got != want {
			t.Errorf("outline %d (%q) points at page %d, want %d", i, dst.outlines[i].text, got, want)
		}
	}
}

// TestAbsorbPages_RewritesInternalLinkIndices verifies the two-level
// indirection survives splicing: the clickable area must still find its
// destination in the merged link table, and the destination must point at the
// page's new position.
func TestAbsorbPages_RewritesInternalLinkIndices(t *testing.T) {
	t.Parallel()

	dst := newAbsorbTestPDF(t)
	dstLink := dst.AddLink()
	dst.SetLink(dstLink, 30, 1)
	dst.Link(10, 10, 20, 5, dstLink)

	src := newAbsorbTestPDF(t)
	src.AddPage()
	srcLink := src.AddLink()
	src.SetLink(srcLink, 40, 2) // second page of the source
	src.Link(10, 10, 20, 5, srcLink)

	if err := dst.AbsorbPages(src); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	// The source's clickable area was drawn on its second page, which is now
	// the third page of the merged document.
	if got := len(dst.pageLinks[3]); got != 1 {
		t.Fatalf("page 3 has %d links, want 1", got)
	}
	absorbed := dst.pageLinks[3][0].link
	if absorbed == srcLink {
		t.Errorf("link index %d was not rewritten; it still collides with the destination's own link", absorbed)
	}
	if got, want := dst.links[absorbed].page, 3; got != want {
		t.Errorf("absorbed link targets page %d, want %d", got, want)
	}
	if got, want := dst.links[dstLink].page, 1; got != want {
		t.Errorf("destination's own link targets page %d, want %d", got, want)
	}
}

// TestAbsorbPages_ReconcilesNamedLinksAcrossDocuments is the cross-chunk case:
// the clickable area and the page it jumps to were rendered by different
// workers, so neither document can resolve the link on its own. The shared name
// is what lets the two reservations collapse into one resolved destination.
func TestAbsorbPages_ReconcilesNamedLinksAcrossDocuments(t *testing.T) {
	t.Parallel()

	// The worker that drew the clickable area only reserved the destination.
	source := newAbsorbTestPDF(t)
	sourceLink := source.AddNamedLink("chapter-2")
	source.Link(10, 10, 20, 5, sourceLink)

	// A different worker rendered the page the link jumps to.
	target := newAbsorbTestPDF(t)
	targetLink := target.AddNamedLink("chapter-2")
	target.SetLink(targetLink, 25, 1)

	if err := source.AbsorbPages(target); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	link := source.pageLinks[1][0].link
	if got, want := source.links[link].page, 2; got != want {
		t.Errorf("cross-document link targets page %d, want %d", got, want)
	}
	if got, want := source.links[link].y, 25.0; got != want {
		t.Errorf("cross-document link targets y=%.2f, want %.2f", got, want)
	}
	if got := len(source.links); got != 2 {
		t.Errorf("link table holds %d entries (plus placeholder), want 1: the two reservations must collapse", got-1)
	}
}

// TestAbsorbPages_ReconcilesNamedLinksWhenTargetAbsorbedFirst covers the other
// absorption order: the destination is already resolved when the reservation
// arrives, so the reservation must not overwrite it.
func TestAbsorbPages_ReconcilesNamedLinksWhenTargetAbsorbedFirst(t *testing.T) {
	t.Parallel()

	target := newAbsorbTestPDF(t)
	targetLink := target.AddNamedLink("chapter-2")
	target.SetLink(targetLink, 25, 1)

	source := newAbsorbTestPDF(t)
	sourceLink := source.AddNamedLink("chapter-2")
	source.Link(10, 10, 20, 5, sourceLink)

	if err := target.AbsorbPages(source); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	link := target.pageLinks[2][0].link
	if got, want := target.links[link].page, 1; got != want {
		t.Errorf("cross-document link targets page %d, want %d", got, want)
	}
}

// TestAddNamedLink_ReusesTheSameReservation pins the identity contract the
// splice reconciliation relies on.
func TestAddNamedLink_ReusesTheSameReservation(t *testing.T) {
	t.Parallel()

	f := newAbsorbTestPDF(t)

	first := f.AddNamedLink("intro")
	second := f.AddNamedLink("intro")
	other := f.AddNamedLink("outro")

	if first != second {
		t.Errorf("reserving %q twice returned %d and %d, want the same identifier", "intro", first, second)
	}
	if first == other {
		t.Errorf("distinct names both returned identifier %d", first)
	}
}

// TestAbsorbPages_UnionsGradientsAndBlendModes verifies the content-hash naming
// makes graphics state merge like fonts do: shared definitions collapse and
// distinct ones are all carried across.
func TestAbsorbPages_UnionsGradientsAndBlendModes(t *testing.T) {
	t.Parallel()

	dst := newAbsorbTestPDF(t)
	dst.SetAlpha(0.5, "Normal")
	dst.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)

	src := newAbsorbTestPDF(t)
	src.SetAlpha(0.5, "Normal")                                             // same state as dst
	src.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)    // same gradient as dst
	src.SetAlpha(0.25, "Multiply")                                          // new state
	src.RadialGradient(10, 10, 50, 50, 0, 255, 0, 0, 0, 255, 0, 0, 1, 1, 1) // new gradient

	if err := dst.AbsorbPages(src); err != nil {
		t.Fatalf("absorb: %v", err)
	}

	// One placeholder at index 0, plus the shared definition, plus the new one.
	if got, want := len(dst.blendList), 3; got != want {
		t.Errorf("blend mode count = %d, want %d", got-1, want-1)
	}
	if got, want := len(dst.gradientList), 3; got != want {
		t.Errorf("gradient count = %d, want %d", got-1, want-1)
	}

	// The shared gradient must reach the file once, not once per document.
	var out bytes.Buffer
	if err := dst.Output(&out); err != nil {
		t.Fatalf("output: %v", err)
	}
	if got, want := bytes.Count(out.Bytes(), []byte("/ShadingType")), 2; got != want {
		t.Errorf("wrote %d shading objects, want %d", got, want)
	}
	if got, want := bytes.Count(out.Bytes(), []byte("/Type /ExtGState")), 2; got != want {
		t.Errorf("wrote %d graphics state objects, want %d", got, want)
	}
}

// TestGradients_DeduplicateWithinOneDocument covers the sequential-generation
// side of content-hash naming: drawing the same gradient repeatedly used to
// embed one shading object per call.
func TestGradients_DeduplicateWithinOneDocument(t *testing.T) {
	t.Parallel()

	f := newAbsorbTestPDF(t)
	for range 10 {
		f.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)
	}

	if got, want := len(f.gradientList), 2; got != want {
		t.Errorf("gradient count = %d, want %d (the same gradient must embed once)", got-1, want-1)
	}
}

// TestGraphicsStateNames_AreContentDerived pins the property splicing depends
// on: two independent documents drawing the same state name it identically, and
// a different state gets a different name.
func TestGraphicsStateNames_AreContentDerived(t *testing.T) {
	t.Parallel()

	first := newAbsorbTestPDF(t)
	first.SetAlpha(0.5, "Normal")
	first.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)

	second := newAbsorbTestPDF(t)
	second.SetAlpha(0.5, "Normal")
	second.LinearGradient(10, 10, 50, 50, 255, 0, 0, 0, 0, 255, 0, 0, 1, 1)
	second.SetAlpha(0.25, "Normal")

	if first.blendList[1].id != second.blendList[1].id {
		t.Error("the same blend mode got different names in two documents")
	}
	if first.gradientList[1].id != second.gradientList[1].id {
		t.Error("the same gradient got different names in two documents")
	}
	if second.blendList[1].id == second.blendList[2].id {
		t.Error("two different blend modes share a name")
	}

	content := first.pages[1].String()
	if !strings.Contains(content, "/GS"+first.blendList[1].id+" gs") {
		t.Error("content stream does not select the blend mode by its content-derived name")
	}
	if !strings.Contains(content, "/Sh"+first.gradientList[1].id+" sh") {
		t.Error("content stream does not paint the gradient by its content-derived name")
	}
}
