package reader_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	pagecomponent "github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/reader"
)

func TestParseGeneratedPDFExposesPageMetadataAndText(t *testing.T) {
	t.Parallel()

	pdfBytes := generatedReaderPDF(t, false, "Reader foundation text")

	r, err := reader.Parse(pdfBytes)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := r.PageCount(); got != 1 {
		t.Fatalf("PageCount() = %d, want 1", got)
	}
	if !bytes.Equal(r.RawBytes(), pdfBytes) {
		t.Fatal("RawBytes() must return the parsed PDF bytes")
	}
	if got := r.Version(); got == "" {
		t.Fatal("Version() returned empty string")
	}

	page, err := r.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	if page.Number != 1 {
		t.Fatalf("page.Number = %d, want 1", page.Number)
	}
	if page.Width <= 0 || page.Height <= 0 {
		t.Fatalf("page dimensions must be positive, got %.2fx%.2f", page.Width, page.Height)
	}
	if page.MediaBox.Width() <= 0 || page.MediaBox.Height() <= 0 {
		t.Fatalf("MediaBox must be populated, got %+v", page.MediaBox)
	}
	if page.VisibleBox().Width() <= 0 || page.VisibleBox().Height() <= 0 {
		t.Fatalf("VisibleBox must be populated, got %+v", page.VisibleBox())
	}

	content, err := page.ContentStream()
	if err != nil {
		t.Fatalf("ContentStream() error = %v", err)
	}
	if !bytes.Contains(content, []byte("Reader foundation text")) {
		t.Fatalf("content stream did not include expected text:\n%s", content)
	}

	extracted, err := page.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText() error = %v", err)
	}
	if !strings.Contains(extracted, "Reader foundation text") {
		t.Fatalf("ExtractText() = %q, want generated text", extracted)
	}
}

func TestParseGeneratedCompressedPDFExtractsText(t *testing.T) {
	t.Parallel()

	pdfBytes := generatedReaderPDF(t, true, "Compressed reader text")

	r, err := reader.Parse(pdfBytes)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	page, err := r.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	extracted, err := page.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText() error = %v", err)
	}
	if !strings.Contains(extracted, "Compressed reader text") {
		t.Fatalf("ExtractText() = %q, want compressed text", extracted)
	}
}

func TestParseRejectsEncryptedPDFs(t *testing.T) {
	t.Parallel()

	pdfBytes := generatedReaderPDF(t, false, "secret")
	pdfBytes = bytes.Replace(pdfBytes, []byte("/Producer"), []byte("/Encrypt\n/Producer"), 1)

	_, err := reader.Parse(pdfBytes)
	if err == nil {
		t.Fatal("Parse() expected encrypted PDF error")
	}
	if !strings.Contains(err.Error(), "encrypted") {
		t.Fatalf("Parse() error = %v, want encrypted", err)
	}
}

func TestLoadAcceptsLeadingGarbageBeforeHeader(t *testing.T) {
	t.Parallel()

	pdfBytes := append([]byte("prefix bytes\n"), generatedReaderPDF(t, false, "load text")...)
	path := filepath.Join(t.TempDir(), "with-prefix.pdf")
	if err := os.WriteFile(path, pdfBytes, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r, err := reader.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := r.PageCount(); got != 1 {
		t.Fatalf("PageCount() = %d, want 1", got)
	}
}

func TestPageRejectsOutOfRangeIndex(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedReaderPDF(t, false, "one page"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	_, err = r.Page(1)
	if err == nil {
		t.Fatal("Page(1) expected out-of-range error")
	}
}

func TestMergeReadersWritesMergedPDF(t *testing.T) {
	t.Parallel()

	first, err := reader.Parse(generatedReaderPDF(t, false, "first"))
	if err != nil {
		t.Fatalf("Parse(first) error = %v", err)
	}
	second, err := reader.Parse(generatedReaderPDF(t, false, "second"))
	if err != nil {
		t.Fatalf("Parse(second) error = %v", err)
	}

	merged, err := reader.Merge(first, second)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	mergedBytes := merged.Bytes()
	if !bytes.HasPrefix(mergedBytes, []byte("%PDF-")) {
		t.Fatalf("merged PDF has invalid header: %.20q", mergedBytes)
	}

	parsed, err := reader.Parse(mergedBytes)
	if err != nil {
		t.Fatalf("Parse(merged) error = %v", err)
	}
	if got := parsed.PageCount(); got != 2 {
		t.Fatalf("merged PageCount() = %d, want 2", got)
	}

	outPath := filepath.Join(t.TempDir(), "merged.pdf")
	if err := merged.SaveTo(outPath); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}
	if info, err := os.Stat(outPath); err != nil || info.Size() == 0 {
		t.Fatalf("SaveTo() wrote invalid file, info=%v err=%v", info, err)
	}
}

func TestExtractPagesWritesSelectedPagesInOrder(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedMultiPageReaderPDF(t))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	extracted, err := reader.ExtractPages(r, 2, 0)
	if err != nil {
		t.Fatalf("ExtractPages() error = %v", err)
	}
	parsed, err := reader.Parse(extracted.Bytes())
	if err != nil {
		t.Fatalf("Parse(extracted) error = %v", err)
	}
	if got := parsed.PageCount(); got != 2 {
		t.Fatalf("PageCount() = %d, want 2", got)
	}
	first, err := parsed.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	text0, err := first.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText(0) error = %v", err)
	}
	if !strings.Contains(text0, "third page") {
		t.Fatalf("first extracted page text = %q, want third page", text0)
	}
	second, err := parsed.Page(1)
	if err != nil {
		t.Fatalf("Page(1) error = %v", err)
	}
	text1, err := second.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText(1) error = %v", err)
	}
	if !strings.Contains(text1, "first page") {
		t.Fatalf("second extracted page text = %q, want first page", text1)
	}
}

func TestExtractPagesRejectsOutOfRangePage(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedMultiPageReaderPDF(t))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	_, err = reader.ExtractPages(r, 3)
	if err == nil {
		t.Fatal("ExtractPages() expected out-of-range error")
	}
}

func TestModifierRemoveAndReorderPages(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedMultiPageReaderPDF(t))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	modifier, err := reader.Merge(r)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if err := modifier.RemovePage(1); err != nil {
		t.Fatalf("RemovePage() error = %v", err)
	}
	if got := modifier.PageCount(); got != 2 {
		t.Fatalf("PageCount() after RemovePage = %d, want 2", got)
	}
	if err := modifier.ReorderPages([]int{1, 0}); err != nil {
		t.Fatalf("ReorderPages() error = %v", err)
	}

	parsed, err := reader.Parse(modifier.Bytes())
	if err != nil {
		t.Fatalf("Parse(modified) error = %v", err)
	}
	first, err := parsed.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	firstText, err := first.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText(0) error = %v", err)
	}
	if !strings.Contains(firstText, "third page") {
		t.Fatalf("first modified page text = %q, want third page", firstText)
	}
	second, err := parsed.Page(1)
	if err != nil {
		t.Fatalf("Page(1) error = %v", err)
	}
	secondText, err := second.ExtractText()
	if err != nil {
		t.Fatalf("ExtractText(1) error = %v", err)
	}
	if !strings.Contains(secondText, "first page") {
		t.Fatalf("second modified page text = %q, want first page", secondText)
	}
}

func TestModifierRotateCropAndAddBlankPage(t *testing.T) {
	t.Parallel()

	r, err := reader.Parse(generatedReaderPDF(t, false, "geometry page"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	modifier, err := reader.Merge(r)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if err := modifier.RotatePage(0, 90); err != nil {
		t.Fatalf("RotatePage() error = %v", err)
	}
	if err := modifier.CropPage(0, [4]float64{36, 36, 576, 756}); err != nil {
		t.Fatalf("CropPage() error = %v", err)
	}
	if err := modifier.AddBlankPage(200, 300); err != nil {
		t.Fatalf("AddBlankPage() error = %v", err)
	}

	parsed, err := reader.Parse(modifier.Bytes())
	if err != nil {
		t.Fatalf("Parse(modified) error = %v", err)
	}
	if got := parsed.PageCount(); got != 2 {
		t.Fatalf("PageCount() = %d, want 2", got)
	}
	first, err := parsed.Page(0)
	if err != nil {
		t.Fatalf("Page(0) error = %v", err)
	}
	if first.Rotate != 90 {
		t.Fatalf("Rotate = %d, want 90", first.Rotate)
	}
	if first.CropBox != (reader.Box{X1: 36, Y1: 36, X2: 576, Y2: 756}) {
		t.Fatalf("CropBox = %+v, want 36 36 576 756", first.CropBox)
	}
	blank, err := parsed.Page(1)
	if err != nil {
		t.Fatalf("Page(1) error = %v", err)
	}
	if blank.MediaBox.Width() != 200 || blank.MediaBox.Height() != 300 {
		t.Fatalf("blank MediaBox = %.0fx%.0f, want 200x300", blank.MediaBox.Width(), blank.MediaBox.Height())
	}
}

func generatedReaderPDF(t *testing.T, compression bool, value string) []byte {
	t.Helper()

	cfg := config.NewBuilder().
		WithPageSize(pagesize.Letter).
		WithCompression(compression).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New(value)))
	pdf, err := doc.Generate(t.Context())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return pdf.GetBytes()
}

func generatedMultiPageReaderPDF(t *testing.T) []byte {
	t.Helper()

	doc := paper.New(config.NewBuilder().WithCompression(false).Build())
	doc.AddPages(
		pagecomponent.New().Add(text.NewRow(10, "first page")),
		pagecomponent.New().Add(text.NewRow(10, "second page")),
		pagecomponent.New().Add(text.NewRow(10, "third page")),
	)
	pdf, err := doc.Generate(t.Context())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return pdf.GetBytes()
}

func generatedReaderPDFWithMetadata(t *testing.T, value, author, title string) []byte {
	t.Helper()

	cfg := config.NewBuilder().
		WithPageSize(pagesize.Letter).
		WithCompression(false).
		WithAuthor(author, false).
		WithTitle(title, false).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New(value)))
	pdf, err := doc.Generate(t.Context())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return pdf.GetBytes()
}
