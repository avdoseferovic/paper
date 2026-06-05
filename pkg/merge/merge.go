// Package merge implements PDF merge.
package merge

import (
	"errors"
	"fmt"
)

var ErrCannotMergePDFs = errors.New("cannot merge PDFs")

// Bytes merges PDFs from byte slices.
func Bytes(pdfs ...[]byte) ([]byte, error) {
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("%w: no PDFs provided", ErrCannotMergePDFs)
	}

	documents := make([]*pdfDocument, 0, len(pdfs))
	for _, pdf := range pdfs {
		document, err := parsePDF(pdf)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
		}
		documents = append(documents, document)
	}

	merged, err := writeMergedPDF(documents)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
	}
	return merged, nil
}
