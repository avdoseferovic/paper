package merge

import (
	"context"
	"fmt"
)

// PageSelection selects zero-based pages of one source PDF. Pages lists the
// page indexes to include, in output order; indexes may repeat.
type PageSelection struct {
	PDF   []byte
	Pages []int
}

// BytesSelected merges the selected pages of each source PDF into one
// document, preserving the requested page order. It observes ctx between
// documents. Unselected pages are removed from the merged page tree; their
// underlying objects remain in the file so cross-references (for example
// outline destinations) stay resolvable.
func BytesSelected(ctx context.Context, selections ...PageSelection) ([]byte, error) {
	if len(selections) == 0 {
		return nil, fmt.Errorf("%w: no PDFs provided", ErrCannotMergePDFs)
	}

	documents := make([]*pdfDocument, 0, len(selections))
	pageLists := make([][]int, 0, len(selections))
	for _, selection := range selections {
		err := mergeCanceled(ctx)
		if err != nil {
			return nil, err
		}
		if len(selection.Pages) == 0 {
			return nil, fmt.Errorf("%w: no pages selected", ErrCannotMergePDFs)
		}
		document, err := parsePDF(selection.PDF)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
		}
		for _, pageIndex := range selection.Pages {
			if pageIndex < 0 || pageIndex >= len(document.pageIDs) {
				return nil, fmt.Errorf("%w: page index %d out of range (0-%d)",
					ErrCannotMergePDFs, pageIndex, len(document.pageIDs)-1)
			}
		}
		documents = append(documents, document)
		pageLists = append(pageLists, selection.Pages)
	}

	err := mergeCanceled(ctx)
	if err != nil {
		return nil, err
	}

	merged, err := writeMergedPDF(documents, pageLists)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
	}
	return merged, nil
}

// selectedPageIDs returns the source page object IDs to place in the merged
// page tree for one document, honoring an optional page selection.
func selectedPageIDs(document *pdfDocument, selections [][]int, source int) []int {
	if selections == nil || source >= len(selections) || selections[source] == nil {
		return document.pageIDs
	}
	ids := make([]int, 0, len(selections[source]))
	for _, pageIndex := range selections[source] {
		ids = append(ids, document.pageIDs[pageIndex])
	}
	return ids
}
