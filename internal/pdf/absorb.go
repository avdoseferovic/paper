package pdf

import (
	"errors"
	"fmt"
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
// that is serialized once. Splicing is possible because content streams refer
// to fonts and images by a content hash of their definition (see
// generateFontID / generateImageID), not by a sequential index, so identical
// resources produce identical names in every worker and the resource tables
// merge as a plain set union.
//
// Both documents must still be in the rendering phase; call this before
// GenerateBytes/Output on f. src must not be used afterwards, since f takes
// ownership of its page buffers.
//
// Features whose PDF names are allocated sequentially (gradients, blend modes)
// or that store absolute page indices (links, outlines, annotations, page
// geometries) are rejected with ErrAbsorbUnsupported rather than silently
// producing a corrupt document.
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

	for n := 1; n <= src.page; n++ {
		f.page++
		f.pages = append(f.pages, src.pages[n])
		f.pageLinks = append(f.pageLinks, make([]linkType, 0))
		if size, ok := src.pageSizes[n]; ok {
			f.pageSizes[f.page] = size
		}
		if boxes, ok := src.pageBoxes[n]; ok {
			f.pageBoxes[f.page] = boxes
		}
	}

	f.absorbFonts(src)
	f.absorbImages(src)
	return nil
}

// absorbable reports whether f's pages can be spliced into another document
// without rewriting position-dependent names or page references.
func (f *PDF) absorbable() error {
	src := f
	// blendList and gradientList are 1-based with a placeholder at index 0, so
	// a length above one means the document actually registered something. Both
	// are named /GS<n> and /Sh<n> by slice position inside content streams.
	if len(src.blendList) > 1 {
		return fmt.Errorf("%w: blend modes", ErrAbsorbUnsupported)
	}
	if len(src.gradientList) > 1 {
		return fmt.Errorf("%w: gradients", ErrAbsorbUnsupported)
	}
	// links is 1-based with a placeholder at index 0 (see the PDF constructor),
	// so only a length above one means a link was actually registered.
	if len(src.links) > 1 {
		return fmt.Errorf("%w: internal links", ErrAbsorbUnsupported)
	}
	if len(src.outlines) > 0 {
		return fmt.Errorf("%w: outlines", ErrAbsorbUnsupported)
	}
	if len(src.pageAnnotations) > 0 {
		return fmt.Errorf("%w: page annotations", ErrAbsorbUnsupported)
	}
	if len(src.pageGeometries) > 0 {
		return fmt.Errorf("%w: page geometries", ErrAbsorbUnsupported)
	}
	if len(src.acroFormFields) > 0 {
		return fmt.Errorf("%w: form fields", ErrAbsorbUnsupported)
	}
	for n := 1; n <= src.page; n++ {
		if n < len(src.pageLinks) && len(src.pageLinks[n]) > 0 {
			return fmt.Errorf("%w: page links", ErrAbsorbUnsupported)
		}
	}
	return nil
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
