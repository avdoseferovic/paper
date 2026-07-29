package pdf

import (
	"errors"
	"fmt"
	"slices"
)

// ErrAbsorbUnsupported reports that a source document uses a feature whose
// PDF names or page references are position-dependent, so its pages cannot be
// spliced into another document without rewriting them.
var ErrAbsorbUnsupported = errors.New("pdf: source document uses features unsupported by page absorption")

// AbsorbPages appends the rendered pages of src to f and merges src's document
// level resource tables into f's.
//
// This exists to support rendering pages concurrently: each worker renders its
// own pages into a private PDF, and the results are spliced into one document
// that is serialized once. Splicing works because everything a content stream
// names is identified independently of its position: fonts, images, gradients
// and blend modes are named by a content hash of their definition (see
// generateFontID / generateImageID / gradientType.id / blendModeType.id), so
// identical resources produce identical names in every worker and the resource
// tables merge as a plain set union. What is left is state that records
// absolute page numbers — links and outlines — which is rewritten here by the
// number of pages already in f.
//
// Both documents must still be in the rendering phase; call this before
// GenerateBytes/Output on f. src must not be used afterwards, since f takes
// ownership of its page buffers.
//
// Document-catalog features that are written once per document (annotations,
// page geometries, form fields) are rejected with ErrAbsorbUnsupported rather
// than silently producing a document that has lost them.
func (f *PDF) AbsorbPages(src *PDF) error {
	if f.err != nil {
		return f.err
	}
	if src.err != nil {
		return src.err
	}
	if err := src.absorbable(); err != nil {
		return err
	}

	pageOffset := f.page
	links := f.absorbLinks(src, pageOffset)

	for n := 1; n <= src.page; n++ {
		f.page++
		f.pages = append(f.pages, src.pages[n])
		f.pageLinks = append(f.pageLinks, remapPageLinks(src.pageLinks[n], links))
		if size, ok := src.pageSizes[n]; ok {
			f.pageSizes[f.page] = size
		}
		if boxes, ok := src.pageBoxes[n]; ok {
			f.pageBoxes[f.page] = boxes
		}
	}

	f.absorbOutlines(src, pageOffset)
	f.absorbFonts(src)
	f.absorbImages(src)
	f.absorbBlendModes(src)
	f.absorbGradients(src)
	return nil
}

// PagesAbsorbable reports whether the pages rendered into f so far could be
// spliced into another document, returning ErrAbsorbUnsupported if not.
//
// Callers rendering pages concurrently use this to notice an unspliceable
// feature as soon as it is drawn, rather than after rendering the whole
// document, so the work already done can be abandoned early. It only inspects
// slice lengths, so it is cheap enough to call per page.
func (f *PDF) PagesAbsorbable() error {
	return f.absorbable()
}

// absorbable reports whether f's pages can be spliced into another document.
//
// Everything left here is a document-catalog feature: it is written once for
// the whole document and carries page references throughout, so splicing would
// need a design pass of its own rather than an index offset.
//
// None of these is reachable through paper's public API, which routes the
// configuration that produces them to single-document generation before a
// generation mode is chosen (see entity.Config.RequiresSingleDocumentGeneration).
// The check is kept as defence in depth: it is what turns "someone added a new
// position-dependent feature" into a sequential fallback instead of a silently
// corrupt PDF.
func (f *PDF) absorbable() error {
	if len(f.pageAnnotations) > 0 {
		return fmt.Errorf("%w: page annotations", ErrAbsorbUnsupported)
	}
	if len(f.pageGeometries) > 0 {
		return fmt.Errorf("%w: page geometries", ErrAbsorbUnsupported)
	}
	if len(f.acroFormFields) > 0 {
		return fmt.Errorf("%w: form fields", ErrAbsorbUnsupported)
	}
	return nil
}

// absorbLinks merges src's internal link destinations into f, shifting the page
// numbers of resolved destinations by pageOffset. It returns the translation
// from src's link identifiers to f's, indexed by src identifier; entry 0 is the
// "no internal link" identifier and always maps to itself.
func (f *PDF) absorbLinks(src *PDF, pageOffset int) []int {
	translation := make([]int, len(src.links))
	for link := 1; link < len(src.links); link++ {
		destination := src.links[link]
		if destination.page != 0 {
			destination.page += pageOffset
		}
		translation[link] = f.mergeLink(destination)
	}
	return translation
}

// mergeLink adds a link destination to f and returns its identifier there.
//
// A named destination that f already knows keeps its existing identifier, so
// the worker that rendered the target page and the workers that only drew
// clickable areas pointing at it converge on one link. Only one of them
// resolved the destination, so a resolved entry always wins over a reservation.
func (f *PDF) mergeLink(destination intLinkType) int {
	if destination.name == "" {
		f.links = append(f.links, destination)
		return len(f.links) - 1
	}
	link, exists := f.linkNames[destination.name]
	if !exists {
		f.links = append(f.links, destination)
		link = len(f.links) - 1
		f.linkNames[destination.name] = link
		return link
	}
	if f.links[link].page == 0 {
		f.links[link] = destination
	}
	return link
}

// remapPageLinks copies a page's clickable areas, rewriting internal link
// identifiers through translation. External links carry a URL rather than an
// identifier and need no rewriting.
func remapPageLinks(pageLinks []linkType, translation []int) []linkType {
	absorbed := slices.Clone(pageLinks)
	for i := range absorbed {
		if absorbed[i].link != 0 {
			absorbed[i].link = translation[absorbed[i].link]
		}
	}
	return absorbed
}

// absorbOutlines appends src's bookmarks, shifting the page each points at.
//
// The outline tree itself is wired at serialization time by
// prepareBookmarkOutlines, so only the destination needs rewriting here.
// Bookmarks end up in document order because page groups are absorbed in page
// order.
func (f *PDF) absorbOutlines(src *PDF, pageOffset int) {
	for _, outline := range src.outlines {
		outline.p += pageOffset
		f.outlines = append(f.outlines, outline)
	}
}

// absorbFonts unions src's font tables into f. Font names in content streams
// are content hashes, so an existing key needs no work. DiffN, by contrast, is
// a 1-based index into the destination's diffs slice and must be reallocated.
func (f *PDF) absorbFonts(src *PDF) {
	for key, def := range src.fonts {
		if _, exists := f.fonts[key]; exists {
			continue
		}
		if def.DiffN > 0 {
			f.registerFontDiff(&def)
		}
		f.fonts[key] = def
	}
	for key, file := range src.fontFiles {
		if _, exists := f.fontFiles[key]; !exists {
			f.fontFiles[key] = file
		}
	}
}

// absorbImages unions src's image table into f. Image names are content hashes,
// so duplicates across workers collapse to a single embedded object.
func (f *PDF) absorbImages(src *PDF) {
	for key, img := range src.images {
		if _, exists := f.images[key]; !exists {
			f.images[key] = img
		}
	}
}

// absorbBlendModes unions src's alpha and blend mode states into f. They are
// named by a content hash in content streams, so the same state registered by
// two workers collapses to a single ExtGState object.
func (f *PDF) absorbBlendModes(src *PDF) {
	for i := 1; i < len(src.blendList); i++ {
		blend := src.blendList[i]
		if _, exists := f.blendMap[blend.id]; exists {
			continue
		}
		f.blendMap[blend.id] = len(f.blendList)
		f.blendList = append(f.blendList, blend)
	}
}

// absorbGradients unions src's gradients into f, on the same content-hash basis
// as blend modes.
func (f *PDF) absorbGradients(src *PDF) {
	for i := 1; i < len(src.gradientList); i++ {
		gradient := src.gradientList[i]
		if _, exists := f.gradientMap[gradient.id]; exists {
			continue
		}
		f.gradientMap[gradient.id] = len(f.gradientList)
		f.gradientList = append(f.gradientList, gradient)
	}
}
