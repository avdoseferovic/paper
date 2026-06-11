// Package merge implements PDF merge.
package merge

import (
	"context"
	"errors"
	"fmt"
)

var ErrCannotMergePDFs = errors.New("cannot merge PDFs")

// Bytes merges PDFs from byte slices. It observes ctx between documents.
func Bytes(ctx context.Context, pdfs ...[]byte) ([]byte, error) {
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("%w: no PDFs provided", ErrCannotMergePDFs)
	}

	documents := make([]*pdfDocument, 0, len(pdfs))
	for _, pdf := range pdfs {
		err := mergeCanceled(ctx)
		if err != nil {
			return nil, err
		}
		document, err := parsePDF(pdf)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
		}
		documents = append(documents, document)
	}

	err := mergeCanceled(ctx)
	if err != nil {
		return nil, err
	}

	merged, err := writeMergedPDF(documents)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCannotMergePDFs, err)
	}
	return merged, nil
}

func mergeCanceled(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("merge: canceled: %w", err)
	}
	return nil
}
