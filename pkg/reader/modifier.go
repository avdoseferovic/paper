package reader

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/avdoseferovic/paper/internal/pdfscan"
	"github.com/avdoseferovic/paper/pkg/merge"
)

// RemovePage removes the zero-based page at index.
func (m *Modifier) RemovePage(index int) error {
	r, err := modifierReader(m)
	if err != nil {
		return err
	}
	if index < 0 || index >= r.PageCount() {
		return fmt.Errorf("%w: %d", ErrPageOutOfRange, index)
	}
	pages := make([]int, 0, r.PageCount()-1)
	for pageIndex := range r.PageCount() {
		if pageIndex != index {
			pages = append(pages, pageIndex)
		}
	}
	out, err := merge.BytesSelected(context.Background(), merge.PageSelection{
		PDF:   m.Bytes(),
		Pages: pages,
	})
	if err != nil {
		return err
	}
	m.data = out
	m.pageCount = len(pages)
	return nil
}

// ReorderPages rearranges pages according to order, a zero-based permutation.
func (m *Modifier) ReorderPages(order []int) error {
	r, err := modifierReader(m)
	if err != nil {
		return err
	}
	if len(order) != r.PageCount() {
		return fmt.Errorf("%w: order length %d does not match page count %d", ErrPageOutOfRange, len(order), r.PageCount())
	}
	seen := make(map[int]bool, len(order))
	for _, pageIndex := range order {
		if pageIndex < 0 || pageIndex >= r.PageCount() {
			return fmt.Errorf("%w: %d", ErrPageOutOfRange, pageIndex)
		}
		if seen[pageIndex] {
			return fmt.Errorf("%w: duplicate page index %d", ErrPageOutOfRange, pageIndex)
		}
		seen[pageIndex] = true
	}
	out, err := merge.BytesSelected(context.Background(), merge.PageSelection{
		PDF:   m.Bytes(),
		Pages: slices.Clone(order),
	})
	if err != nil {
		return err
	}
	m.data = out
	m.pageCount = len(order)
	return nil
}

// RotatePage sets the zero-based page rotation. Degrees must be a multiple of 90.
func (m *Modifier) RotatePage(index, degrees int) error {
	if degrees%90 != 0 {
		return fmt.Errorf("%w: rotation must be a multiple of 90", ErrUnsupportedPDF)
	}
	return m.updatePageDictionary(index, "Rotate", strconv.Itoa(degrees))
}

// CropPage sets the zero-based page CropBox.
func (m *Modifier) CropPage(index int, rect [4]float64) error {
	value := fmt.Sprintf("[%s %s %s %s]",
		formatReaderNumber(rect[0]),
		formatReaderNumber(rect[1]),
		formatReaderNumber(rect[2]),
		formatReaderNumber(rect[3]))
	return m.updatePageDictionary(index, "CropBox", value)
}

// AddBlankPage appends one blank page with dimensions in PDF points.
func (m *Modifier) AddBlankPage(width, height float64) error {
	if m == nil {
		return fmt.Errorf("%w: nil modifier", ErrUnsupportedPDF)
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("%w: blank page dimensions must be positive", ErrUnsupportedPDF)
	}
	out, err := merge.Bytes(context.Background(), m.Bytes(), blankPagePDF(width, height))
	if err != nil {
		return err
	}
	m.data = out
	m.pageCount++
	return nil
}

func (m *Modifier) updatePageDictionary(index int, key, value string) error {
	r, err := modifierReader(m)
	if err != nil {
		return err
	}
	page, err := r.Page(index)
	if err != nil {
		return err
	}
	object := r.objects[page.objectID]
	updated, err := setReaderDictionaryEntry(object.content, key, value)
	if err != nil {
		return err
	}
	object.content = updated
	r.objects[page.objectID] = object

	out, err := rewriteReaderPDF(r.data, r.objects, r.rootID)
	if err != nil {
		return err
	}
	m.data = out
	m.pageCount = r.PageCount()
	return nil
}

func modifierReader(m *Modifier) (*PdfReader, error) {
	if m == nil || len(m.data) == 0 {
		return nil, fmt.Errorf("%w: nil modifier", ErrUnsupportedPDF)
	}
	return Parse(m.data)
}

func rewriteReaderPDF(original []byte, objects map[int]pdfObject, rootID int) ([]byte, error) {
	if len(objects) == 0 {
		return nil, fmt.Errorf("%w: no objects to rewrite", ErrUnsupportedPDF)
	}
	ids := make([]int, 0, len(objects))
	maxID := 0
	for id := range objects {
		ids = append(ids, id)
		maxID = max(maxID, id)
	}
	slices.Sort(ids)

	var out bytes.Buffer
	fmt.Fprintf(&out, "%%PDF-%s\n", parseVersion(original))
	offsets := make(map[int]int, len(objects))
	for _, id := range ids {
		offsets[id] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", id)
		out.Write(bytes.TrimSpace(objects[id].content))
		out.WriteString("\nendobj\n")
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", maxID+1)
	out.WriteString("0000000000 65535 f \n")
	for id := 1; id <= maxID; id++ {
		offset, ok := offsets[id]
		if !ok {
			out.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", maxID+1, rootID, xrefOffset)
	return out.Bytes(), nil
}

func blankPagePDF(width, height float64) []byte {
	mediaBox := fmt.Sprintf("[0 0 %s %s]", formatReaderNumber(width), formatReaderNumber(height))
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox %s >>", mediaBox),
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox %s /Resources << /ProcSet [/PDF] >> /Contents 4 0 R >>", mediaBox),
		"<< /Length 0 >>\nstream\n\nendstream",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		id := i + 1
		offsets[id] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", id, object)
	}
	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for id := 1; id <= len(objects); id++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return out.Bytes()
}

func setReaderDictionaryEntry(content []byte, key, value string) ([]byte, error) {
	cleaned := pdfscan.RemoveDictEntry(bytes.TrimSpace(content), key)
	end := bytes.LastIndex(cleaned, []byte(">>"))
	if end < 0 {
		return nil, fmt.Errorf("%w: page dictionary is malformed", ErrUnsupportedPDF)
	}
	var out bytes.Buffer
	out.Write(bytes.TrimRight(cleaned[:end], " \t\r\n"))
	fmt.Fprintf(&out, " /%s %s ", key, value)
	out.Write(bytes.TrimSpace(cleaned[end:]))
	return out.Bytes(), nil
}

func formatReaderNumber(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
