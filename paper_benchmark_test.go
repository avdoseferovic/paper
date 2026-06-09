package paper_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/col"
	componentimage "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/avdoseferovic/paper/pkg/props"
)

var benchmarkPDFBytes int

func BenchmarkPDFGeneration(b *testing.B) {
	htmlBody := mustReadString(b, "cmd/html-demo/assets/body.html")
	imageBytes := mustReadFile(b, "docs/assets/images/frontpage.png")
	cfg := benchmarkConfig()

	b.Run("HTMLDemoFull", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			doc := generateHTMLDemoDocument(b, htmlBody, cfg)
			consumeBenchmarkDocument(b, doc)
		}
	})

	b.Run("HTMLDemoTranslateOnly", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			rows := benchmarkHTMLRows(b, htmlBody, cfg)
			if len(rows) == 0 {
				b.Fatal("translated no HTML rows")
			}
			benchmarkPDFBytes += len(rows)
		}
	})

	b.Run("TextHeavy", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			m := paper.New(cfg)
			m.AddRows(benchmarkTextRows(180)...)

			doc, err := m.Generate()
			if err != nil {
				b.Fatalf("generate text-heavy document: %v", err)
			}
			consumeBenchmarkDocument(b, doc)
		}
	})

	b.Run("MixedComponents", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			m := paper.New(cfg)
			m.AddRows(benchmarkMixedRows(imageBytes, 40)...)

			doc, err := m.Generate()
			if err != nil {
				b.Fatalf("generate mixed document: %v", err)
			}
			consumeBenchmarkDocument(b, doc)
		}
	})
}

// BenchmarkPDFScaling sweeps document size to expose the per-row cost curve.
// Each sub-benchmark generates a text document with N body rows so the marginal
// cost of a row (and page breaks) can be read off as the slope of ns/op vs N.
func BenchmarkPDFScaling(b *testing.B) {
	cfg := benchmarkConfig()
	for _, rowCount := range []int{10, 50, 100, 500, 1000} {
		b.Run(fmt.Sprintf("Rows=%d", rowCount), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				m := paper.New(cfg)
				m.AddRows(benchmarkTextRows(rowCount)...)

				doc, err := m.Generate()
				if err != nil {
					b.Fatalf("generate %d-row document: %v", rowCount, err)
				}
				consumeBenchmarkDocument(b, doc)
			}
		})
	}
}

func generateHTMLDemoDocument(b *testing.B, htmlBody string, cfg *entity.Config) core.Document {
	b.Helper()

	m := paper.New(cfg)
	m.AddRows(benchmarkHeader()...)
	m.AddRows(benchmarkHTMLRows(b, htmlBody, cfg)...)

	doc, err := m.Generate()
	if err != nil {
		b.Fatalf("generate HTML demo document: %v", err)
	}
	return doc
}

func benchmarkConfig() *entity.Config {
	return config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()
}

func benchmarkHTMLRows(b *testing.B, htmlBody string, cfg *entity.Config) []core.Row {
	b.Helper()

	contentWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	rows, err := html.FromString(htmlBody,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(contentWidth),
		html.WithImageBaseDir("cmd/html-demo/assets"),
		html.WithStylesheetBaseDir("cmd/html-demo/assets"),
	)
	if err != nil {
		b.Fatalf("translate HTML demo body: %v", err)
	}
	return rows
}

func benchmarkHeader() []core.Row {
	dark := &props.Color{Red: 26, Green: 62, Blue: 114}
	muted := &props.Color{Red: 120, Green: 120, Blue: 120}

	return []core.Row{
		row.New(10).Add(
			col.New(8).Add(text.New("PAPER HTML PDF DEMO", props.Text{
				Style: fontstyle.Bold,
				Size:  14,
				Color: dark,
			})),
			col.New(4).Add(text.New("benchmark@paper.example", props.Text{
				Size:  9,
				Align: align.Right,
				Color: muted,
			})),
		),
		row.New(2).Add(col.New(12).Add(line.New(props.Line{
			Color:     dark,
			Thickness: 0.6,
		}))),
	}
}

func benchmarkTextRows(count int) []core.Row {
	rows := make([]core.Row, 0, count+1)
	rows = append(rows, text.NewRow(12, "Generation benchmark: text-heavy document", props.Text{
		Size:  14,
		Align: align.Center,
	}))

	base := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Donec ac condimentum sem. "
	for range count {
		rows = append(rows, text.NewRow(9, strings.Repeat(base, 3), props.Text{
			Size: 9,
			Top:  1,
		}))
	}
	return rows
}

func benchmarkMixedRows(imageBytes []byte, groups int) []core.Row {
	rows := make([]core.Row, 0, groups*5)
	for range groups {
		rows = append(rows,
			row.New(14).Add(
				text.NewCol(4, "Barcode", props.Text{Size: 10, Top: 4, Align: align.Center}),
				code.NewBarCol(8, "paper-benchmark", props.Barcode{Center: true, Percent: 70}),
			),
			row.New(18).Add(
				text.NewCol(4, "QR", props.Text{Size: 10, Top: 5, Align: align.Center}),
				code.NewQrCol(8, "https://github.com/avdoseferovic/paper", props.Rect{Center: true, Percent: 70}),
			),
			row.New(18).Add(
				text.NewCol(4, "Image", props.Text{Size: 10, Top: 5, Align: align.Center}),
				componentimage.NewFromBytesCol(8, imageBytes, extension.Png, props.Rect{Center: true, Percent: 60}),
			),
			row.New(16).Add(
				text.NewCol(4, "Signature", props.Text{Size: 10, Top: 5, Align: align.Center}),
				signature.NewCol(8, "Paper Benchmark", props.Signature{FontSize: 9}),
			),
			text.NewRow(8, "Mixed row payload for PDF generation timing.", props.Text{Size: 8}),
		)
	}
	return rows
}

func consumeBenchmarkDocument(b *testing.B, doc core.Document) {
	b.Helper()

	pdfBytes := doc.GetBytes()
	if len(pdfBytes) == 0 {
		b.Fatal("generated empty PDF")
	}
	benchmarkPDFBytes += len(pdfBytes)
}

func mustReadString(b *testing.B, path string) string {
	b.Helper()
	return string(mustReadFile(b, path))
}

func mustReadFile(b *testing.B, path string) []byte {
	b.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("read %s: %v", path, err)
	}
	return data
}
