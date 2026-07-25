// paper-showcase generates a concise PDF tour of Paper's public authoring APIs.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/examples/internal/examplepath"
	"github.com/avdoseferovic/paper/pkg/components/checkbox"
	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/col"
	htmlcomponent "github.com/avdoseferovic/paper/pkg/components/html"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/components/richtext"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/border"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/fontrepository"
	"github.com/avdoseferovic/paper/pkg/html/translate"
	"github.com/avdoseferovic/paper/pkg/props"
)

const (
	showcaseFont     = "arial-unicode-ms"
	showcaseFontFile = "docs/assets/fonts/arial-unicode-ms.ttf"
)

var (
	ink       = rgb(58, 50, 43)
	inkSoft   = rgb(94, 84, 74)
	muted     = rgb(126, 115, 103)
	faint     = rgb(178, 168, 154)
	brand     = ink
	teal      = rgb(190, 79, 54)
	green     = rgb(72, 132, 93)
	amber     = rgb(163, 91, 61)
	red       = rgb(150, 56, 39)
	blue      = rgb(78, 101, 158)
	white     = rgb(255, 255, 255)
	canvas    = rgb(250, 248, 244)
	codeBg    = rgb(252, 250, 247)
	deepInk   = rgb(41, 35, 30)
	softBlue  = rgb(238, 241, 248)
	softTeal  = rgb(251, 240, 236)
	softGold  = rgb(248, 242, 231)
	softRose  = rgb(249, 235, 231)
	borderC   = rgb(226, 219, 209)
	lineSoftC = rgb(240, 235, 228)
)

func main() {
	out := examplepath.Repo("docs/assets/pdf/showcase.pdf")
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := generateShowcase(out); err != nil {
		log.Fatal(err)
	}
	info, _ := os.Stat(out)
	fmt.Printf("Wrote %s (%d bytes)\n", out, info.Size())
}

func generateShowcase(out string) error {
	doc, err := buildShowcaseDocument()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := doc.Save(out); err != nil {
		return fmt.Errorf("save showcase: %w", err)
	}
	return nil
}

func buildShowcaseDocument() (core.Document, error) {
	m, err := buildShowcasePaper()
	if err != nil {
		return nil, err
	}
	doc, err := m.Generate(context.Background())
	if err != nil {
		return nil, fmt.Errorf("generate showcase: %w", err)
	}
	return doc, nil
}

func buildShowcasePaper() (core.Paper, error) {
	fontBytes, err := os.ReadFile(examplepath.Repo(showcaseFontFile))
	if err != nil {
		return nil, fmt.Errorf("read showcase font: %w", err)
	}

	customFonts, err := fontrepository.New().
		AddUTF8FontFromBytes(showcaseFont, fontstyle.Normal, fontBytes).
		AddUTF8FontFromBytes(showcaseFont, fontstyle.Italic, fontBytes).
		AddUTF8FontFromBytes(showcaseFont, fontstyle.Bold, fontBytes).
		AddUTF8FontFromBytes(showcaseFont, fontstyle.BoldItalic, fontBytes).
		Load()
	if err != nil {
		return nil, fmt.Errorf("load showcase font: %w", err)
	}

	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(12).
		WithBottomMargin(13).
		WithCustomFonts(customFonts).
		WithDefaultFont(&props.Font{Family: showcaseFont, Size: 9.5, Color: ink}).
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
			Family:  showcaseFont,
			Size:    7,
			Color:   muted,
		}).
		WithTitle("Paper Showcase", false).
		WithSubject("Component, grid, and HTML examples for Paper", false).
		WithCreator("examples/cmd/paper-showcase", false).
		WithCreationDate(time.Date(2026, time.June, 8, 0, 0, 0, 0, time.UTC)).
		Build()

	m := paper.New(cfg)
	if err := m.RegisterHeader(headerRows()...); err != nil {
		return nil, fmt.Errorf("register header: %w", err)
	}
	if err := m.RegisterFooter(footerRows()...); err != nil {
		return nil, fmt.Errorf("register footer: %w", err)
	}

	m.AddRows(overviewRows()...)
	m.AddRows(pageBreak())
	m.AddRows(componentRows()...)
	m.AddRows(pageBreak())
	m.AddRows(gridRows()...)
	m.AddRows(pageBreak())
	m.AddRows(htmlRows(cfg.MaxGridSize)...)
	m.AddRows(pageBreak())
	m.AddRows(testingMetricsRows()...)
	m.AddRows(pageBreak())
	m.AddRows(useCaseRows()...)
	m.AddRows(pageBreak())
	m.AddRows(invoiceExampleRows()...)
	m.AddRows(pageBreak())
	m.AddRows(installRows()...)
	return m, nil
}

func headerRows() []core.Row {
	return []core.Row{
		row.New(6).Add(
			text.NewCol(6, "PAPER", props.Text{
				Family: showcaseFont,
				Style:  fontstyle.Bold,
				Size:   9,
				Color:  brand,
			}),
			text.NewCol(6, "Go PDF documents from components and HTML", props.Text{
				Family: consts.FontFamilyCourier,
				Size:   7,
				Align:  consts.AlignRight,
				Color:  muted,
			}),
		),
		row.New(1.2).Add(col.New(12).Add(line.New(props.Line{
			Color:     borderC,
			Thickness: 0.25,
		}))),
	}
}

func footerRows() []core.Row {
	return []core.Row{
		row.New(1).Add(col.New(12).Add(line.New(props.Line{
			Color:     borderC,
			Thickness: 0.2,
		}))),
		row.New(4).Add(text.NewCol(12, "Generated by cd examples && go run ./cmd/paper-showcase", props.Text{
			Family: consts.FontFamilyCourier,
			Size:   6.5,
			Color:  muted,
		})),
	}
}

func overviewRows() []core.Row {
	return []core.Row{
		heroRow(),
		spacer(5),
		row.New(40).Add(
			codeSnippetCol(6, "main.go - one call", `package main

import "github.com/avdoseferovic/paper"

func main() {
  pdf, _ := paper.FromHTML(context.Background(), invoiceHTML)
  _ = pdf.Save("invoice.pdf")
}`, teal),
			codeSnippetCol(6, "layout.go - components", `doc := paper.New(cfg)
doc.AddRows(row.New(12).Add(
  col.New(8).Add(title),
  col.New(4).Add(qrCode),
))
pdf, _ := doc.Generate(context.Background())`, blue),
		),
		spacer(6),
		sectionTitle("Two Authoring Paths", "Start from HTML in one call, or compose deterministic documents with rows, columns, and typed components."),
		row.New(52).Add(
			authoringPathCard(6, "HTML -> PDF", "Bring existing templates and fragments. Paper translates supported HTML and CSS into a print-ready PDF document.", `pdf, err := paper.FromHTML(context.Background(), tmpl)
if err != nil { return err }
_ = pdf.Save("invoice.pdf")`, teal, []string{"templates", "fragments", "CSS-aware"}),
			authoringPathCard(6, "Component grid", "Build explicit document layouts with a 12-column grid. Every primitive is inspectable and reusable in tests.", `m.AddRows(row.New(12).Add(
  col.New(6).Add(header),
  col.New(6).Add(total),
))`, blue, []string{"12 columns", "typed", "testable"}),
		),
		spacer(5),
		row.New(16).Add(
			outputChip(2, ".Save()", "write to disk"),
			outputChip(2, ".Bytes()", "raw bytes"),
			outputChip(2, ".Base64()", "API payloads"),
			outputChip(2, ".Merge()", "combine PDFs"),
			outputChip(4, "+ HTML fragments", "mix markup inside the component grid"),
		),
	}
}

func heroRow() core.Row {
	return row.New(86).Add(
		col.New(7).Add(
			text.New("GO / PDF GENERATION", props.Text{
				Family: consts.FontFamilyCourier,
				Style:  fontstyle.Bold,
				Size:   7,
				Top:    7,
				Left:   5,
				Color:  teal,
			}),
			text.New("Generate PDFs from HTML and Go components", props.Text{
				Family:          showcaseFont,
				Style:           fontstyle.Bold,
				Size:            22,
				Top:             18,
				Left:            5,
				Right:           8,
				VerticalPadding: 1.05,
				Color:           ink,
			}),
			text.New("A precise, testable document engine for production Go services.", props.Text{
				Family:          showcaseFont,
				Size:            11.5,
				Top:             46,
				Left:            5,
				Right:           9,
				VerticalPadding: 1.15,
				Color:           inkSoft,
			}),
			text.New("Render HTML in one call or compose explicit PDF pages from rows, columns, and reusable components.", props.Text{
				Family:          showcaseFont,
				Size:            8,
				Top:             59,
				Left:            5,
				Right:           10,
				VerticalPadding: 1.25,
				Color:           muted,
			}),
			text.New("$ go get github.com/avdoseferovic/paper", props.Text{
				Family: consts.FontFamilyCourier,
				Style:  fontstyle.Bold,
				Size:   7.4,
				Top:    75,
				Left:   5,
				Color:  teal,
			}),
		),
		heroInvoicePreview(),
	).WithStyle(&props.Cell{
		BackgroundColor: canvas,
		BorderType:      border.Full,
		BorderColor:     lineSoftC,
		BorderThickness: 0.25,
		BorderRadius:    3,
	})
}

func heroInvoicePreview() core.Col {
	return col.New(5).Add(
		text.New("Invoice", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 16, Top: 7, Left: 5, Color: ink}),
		text.New("NO. INV-2026-0481", props.Text{Family: consts.FontFamilyCourier, Size: 6.2, Top: 17, Left: 5, Color: muted}),
		code.NewQr("https://github.com/avdoseferovic/paper", props.Rect{Top: 6, Left: 37, Percent: 20}),
		text.New("BILLED TO", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 5.8, Top: 27, Left: 5, Color: faint}),
		text.New("Northwind Trading Co.", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 7.2, Top: 33, Left: 5, Color: inkSoft}),
		text.New("ISSUED", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 5.8, Top: 27, Left: 36, Color: faint}),
		text.New("June 8, 2026", props.Text{Family: showcaseFont, Size: 7.2, Top: 33, Left: 36, Color: inkSoft}),
		text.New("ITEM", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 5.6, Top: 45, Left: 5, Color: faint}),
		text.New("AMOUNT", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 5.6, Top: 45, Align: consts.AlignRight, Right: 5, Color: faint}),
		text.New("Document API - Team", props.Text{Family: showcaseFont, Size: 6.7, Top: 53, Left: 5, Color: inkSoft}),
		text.New("$480.00", props.Text{Family: consts.FontFamilyCourier, Size: 6.7, Top: 53, Align: consts.AlignRight, Right: 5, Color: ink}),
		text.New("Rendering credits", props.Text{Family: showcaseFont, Size: 6.7, Top: 61, Left: 5, Color: inkSoft}),
		text.New("$240.00", props.Text{Family: consts.FontFamilyCourier, Size: 6.7, Top: 61, Align: consts.AlignRight, Right: 5, Color: ink}),
		text.New("TOTAL DUE", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 6, Top: 74, Left: 5, Color: muted}),
		text.New("$840.00", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 14, Top: 70, Align: consts.AlignRight, Right: 5, Color: ink}),
	).WithStyle(&props.Cell{
		BackgroundColor: white,
		BorderType:      border.Full,
		BorderColor:     borderC,
		BorderThickness: 0.25,
		BorderRadius:    2,
		MarginTop:       6,
		MarginRight:     5,
		MarginBottom:    6,
	})
}

func codeSnippetCol(size int, label, value string, accent *props.Color) core.Col {
	components := []core.Component{
		text.New(label, props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 6.5, Top: 3, Left: 4, Color: muted}),
	}
	for i, line := range strings.Split(strings.Trim(value, "\n"), "\n") {
		color := inkSoft
		if strings.Contains(line, "paper.") || strings.Contains(line, "doc.") || strings.Contains(line, "m.") {
			color = accent
		}
		components = append(components, text.New(line, props.Text{
			Family: consts.FontFamilyCourier,
			Size:   6.3,
			Top:    10 + float64(i)*3.3,
			Left:   4,
			Right:  4,
			Color:  color,
		}))
	}
	return col.New(size).Add(components...).WithStyle(codePanelStyle())
}

func authoringPathCard(size int, title, body, snippet string, accent *props.Color, chips []string) core.Col {
	components := []core.Component{
		text.New(title, props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 14, Top: 5, Left: 5, Color: ink}),
		text.New(body, props.Text{Family: showcaseFont, Size: 7.7, Top: 17, Left: 5, Right: 5, VerticalPadding: 1.2, Color: muted}),
	}
	for i, line := range strings.Split(strings.Trim(snippet, "\n"), "\n") {
		components = append(components, text.New(line, props.Text{
			Family: consts.FontFamilyCourier,
			Size:   6.1,
			Top:    31 + float64(i)*3.1,
			Left:   5,
			Right:  5,
			Color:  inkSoft,
		}))
	}
	for i, chip := range chips {
		components = append(components, text.New(chip, props.Text{
			Family: consts.FontFamilyCourier,
			Style:  fontstyle.Bold,
			Size:   5.4,
			Top:    45,
			Left:   5 + float64(i)*23,
			Color:  accent,
		}))
	}
	return col.New(size).Add(components...).WithStyle(cardStyle(white, borderC))
}

func outputChip(size int, command, label string) core.Col {
	return col.New(size).Add(
		text.New(command, props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 7, Top: 3, Align: consts.AlignCenter, Color: teal}),
		text.New(label, props.Text{Family: showcaseFont, Size: 6.2, Top: 9, Align: consts.AlignCenter, Left: 2, Right: 2, Color: muted}),
	).WithStyle(cardStyle(white, borderC))
}

func codePanelStyle() *props.Cell {
	return &props.Cell{
		BackgroundColor: codeBg,
		BorderType:      border.Full,
		BorderColor:     borderC,
		BorderThickness: 0.25,
		BorderRadius:    2,
		MarginLeft:      0.8,
		MarginRight:     0.8,
	}
}

func overviewMatrixTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Surface", true, white), Style: tableHeaderStyle()},
			{Content: tableText("What it shows", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Typical output", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Components", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Explicit rows, columns, and typed primitives.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Operational PDFs and forms.", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("HTML", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Markup translated into Paper rows.", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Rich fragments and imported content.", false, ink), Style: tableCellStyle(softTeal)},
		},
		{
			{Content: tableText("Document", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Headers, footers, page numbers, metadata, and bytes.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Files, streams, base64, and merged PDFs.", false, ink), Style: tableCellStyle(nil)},
		},
	}, table.WithColumnWidths([]float64{1.05, 2.1, 1.9}))
	if err != nil {
		panic(err)
	}
	return t
}

func componentRows() []core.Row {
	return []core.Row{
		sectionTitle("Component Library", "Every document primitive can be dropped into a column, composed into larger blocks, and inspected in tests."),
		row.New(34).Add(
			componentColumn(3, "Text", teal,
				text.New("Headings, labels, and body copy wrap inside the available column width.", props.Text{Top: 12, Left: 4, Right: 4, Size: 7.2, Color: inkSoft, VerticalPadding: 1.1}),
			),
			componentColumn(3, "Rich text", blue,
				richtext.New([]props.RichRun{
					{Text: "Mix ", Size: 7.2},
					{Text: "bold", Style: fontstyle.Bold, Color: ink, Size: 7.2},
					{Text: ", ", Size: 7.2},
					{Text: "links", Underline: true, Color: teal, Size: 7.2},
					{Text: ", and accents.", Size: 7.2},
				}, props.RichText{Top: 13, Left: 4, Right: 4, LineHeight: 1.15}),
			),
			componentColumn(3, "Images", green,
				image.NewFromFile(examplepath.Repo("docs/assets/images/paper-icon.svg"), props.Rect{Top: 11, Left: 25, Percent: 28}),
			),
			componentColumn(3, "Tables", amber,
				text.New("Item      Qty   Total", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 13, Left: 4, Color: faint}),
				text.New("API        1    $480", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 20, Left: 4, Color: inkSoft}),
				text.New("Credits   12k  $240", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 27, Left: 4, Color: inkSoft}),
			),
		),
		spacer(3),
		row.New(34).Add(
			componentColumn(3, "QR codes", teal,
				code.NewQr("https://github.com/avdoseferovic/paper", props.Rect{Top: 10, Left: 20, Percent: 35}),
			),
			componentColumn(3, "Barcodes", blue,
				code.NewBar("PAPER-2026", props.Barcode{Top: 16, Left: 9, Percent: 70, Proportion: props.Proportion{Width: 12, Height: 1.6}}),
			),
			componentColumn(3, "Data Matrix", green,
				code.NewMatrix("PAPER-2026-DM", props.Rect{Top: 10, Left: 20, Percent: 35}),
			),
			componentColumn(3, "Checkboxes", amber,
				checkbox.New("Approved", props.Checkbox{Checked: true, Top: 14, Left: 5, Size: 3.6}),
				checkbox.New("Pending", props.Checkbox{Top: 23, Left: 5, Size: 3.6}),
			),
		),
		spacer(3),
		row.New(34).Add(
			componentColumn(3, "Signatures", teal,
				signature.New("Avdo S.", props.Signature{FontColor: ink, LineColor: borderC, LineThickness: 0.3}),
			),
			componentColumn(3, "Lines", blue,
				line.New(props.Line{Color: ink, Thickness: 0.3, OffsetPercent: 46, SizePercent: 72}),
				line.New(props.Line{Color: faint, Thickness: 0.35, Style: consts.LineStyleDashed, OffsetPercent: 66, SizePercent: 72}),
			),
			componentColumn(3, "Page numbers", green,
				text.New("Page 3 of 12", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 8, Top: 17, Align: consts.AlignCenter, Color: inkSoft}),
			),
			componentColumn(3, "Headers & footers", amber,
				text.New("HEADER", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 10, Align: consts.AlignCenter, Color: muted}),
				text.New("main content region", props.Text{Family: showcaseFont, Size: 6.7, Top: 18, Align: consts.AlignCenter, Color: inkSoft}),
				text.New("FOOTER", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 28, Align: consts.AlignCenter, Color: teal}),
			),
		),
		spacer(6),
		sectionTitle("Components stay inspectable", "The rendered page and the test structure come from the same rows, columns, and components."),
		row.New().Add(col.New(12).Add(componentPlacementTable())),
	}
}

func componentColumn(size int, title string, accent *props.Color, components ...core.Component) core.Col {
	all := make([]core.Component, 0, len(components)+1)
	all = append(all, text.New(title, props.Text{
		Family: showcaseFont,
		Style:  fontstyle.Bold,
		Size:   9,
		Top:    1,
		Left:   4,
		Color:  accent,
	}))
	all = append(all, components...)
	return col.New(size).Add(all...).WithStyle(cardStyle(white, borderC))
}

func componentPlacementTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Component", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Placement behavior", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Practical note", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Text / rich text", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Measures wrapping inside the available column width.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Use auto rows for variable-length body copy.", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Images / codes", false, ink), Style: tableCellStyle(softGold)},
			{Content: tableText("Scale from the cell dimensions and local offsets.", false, ink), Style: tableCellStyle(softGold)},
			{Content: tableText("Reserve clear space around scannable content.", false, ink), Style: tableCellStyle(softGold)},
		},
		{
			{Content: tableText("Tables", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Compute row height from their cell contents.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Good for dense facts, ledgers, and compatibility matrices.", false, ink), Style: tableCellStyle(nil)},
		},
	}, table.WithColumnWidths([]float64{1.25, 2.15, 2.05}))
	if err != nil {
		panic(err)
	}
	return t
}

func gridBehaviorTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Case", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Behavior", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Why it matters", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Exact fit", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Column units add to the configured grid size.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("The row fills the useful page width.", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Underflow", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Unused grid units remain blank space.", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Useful for indents, side notes, and sparse layouts.", false, ink), Style: tableCellStyle(softTeal)},
		},
		{
			{Content: tableText("Auto height", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Largest child height defines the row.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Variable text and tables can remain deterministic.", false, ink), Style: tableCellStyle(nil)},
		},
	}, table.WithColumnWidths([]float64{1.0, 2.2, 2.2}))
	if err != nil {
		panic(err)
	}
	return t
}

func gridRows() []core.Row {
	return []core.Row{
		sectionTitle("Layout Model", "Rows and columns on a 12-column grid, resolved to exact points on the PDF page."),
		row.New(23).Add(
			cardCol(4, "Default denominator", "Paper defaults to a 12-unit grid.", softBlue, blue),
			cardCol(4, "Column widths", "col.New(3) gets 3/12 of the row width.", softTeal, teal),
			cardCol(4, "Auto height", "row.New() measures child content before rendering.", softGold, amber),
		),
		spacer(5),
		gridExampleRow("col.New(12)", []gridCellSpec{{12, "12", blue}}),
		spacer(2),
		gridExampleRow("col.New(6), col.New(6)", []gridCellSpec{{6, "6", teal}, {6, "6", green}}),
		spacer(2),
		gridExampleRow("col.New(4), col.New(4), col.New(4)", []gridCellSpec{{4, "4", amber}, {4, "4", red}, {4, "4", blue}}),
		spacer(2),
		gridExampleRow("four equal columns", []gridCellSpec{{3, "3", blue}, {3, "3", teal}, {3, "3", amber}, {3, "3", red}}),
		spacer(6),
		sectionTitle("How placement works", "Paper measures rows before rendering and creates new pages when the useful area is full."),
		row.New(18).Add(
			col.New(4).Add(text.New("1. Compute useful page area after margins, header, and footer.", props.Text{Top: 1, Right: 4, Size: 8, Color: ink})),
			col.New(4).Add(text.New("2. Measure each row. Rows that overflow move to a new page.", props.Text{Top: 1, Right: 4, Size: 8, Color: ink})),
			col.New(4).Add(text.New("3. Normalize column units against MaxGridSize.", props.Text{Top: 1, Right: 4, Size: 8, Color: ink})),
		),
		thinRule(),
		spacer(4),
		row.New().Add(col.New(12).Add(gridBehaviorTable())),
		spacer(5),
		row.New(23).Add(
			col.New(3).Add(text.New("Nested rows", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: blue}),
				text.New("Components can contain translated HTML or tables inside normal columns.", props.Text{Top: 8, Right: 4, Size: 7.5, Color: ink})),
			col.New(3).Add(text.New("Auto rows", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: teal}),
				text.New("Measured content determines height before drawing begins.", props.Text{Top: 8, Right: 4, Size: 7.5, Color: ink})),
			col.New(3).Add(text.New("Manual rows", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: amber}),
				text.New("Fixed heights keep reports and form regions predictable.", props.Text{Top: 8, Right: 4, Size: 7.5, Color: ink})),
			col.New(3).Add(text.New("Page flow", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: red}),
				text.New("Rows that do not fit continue on the next page with header and footer.", props.Text{Top: 8, Right: 4, Size: 7.5, Color: ink})),
		),
	}
}

func htmlRows(gridSize int) []core.Row {
	return []core.Row{
		sectionTitle("HTML Pipeline", "HTML in, structured Paper rows out, then rendered through the same PDF provider."),
		row.New(32).Add(
			infoCol(4, "Full document", "paper.FromHTML returns a generated PDF document from one HTML string.", blue),
			infoCol(4, "Append rows", "m.AddHTML parses HTML into rows using the active Paper configuration.", teal),
			infoCol(4, "Column fragment", "htmlcomponent.New wraps translated rows as a regular component.", amber),
		),
		spacer(5),
		sectionTitle("HTML support snapshot", "The translator maps common HTML and CSS features onto Paper primitives."),
		row.New().Add(col.New(12).Add(htmlSupportTable())),
		spacer(5),
		row.New().Add(col.New(12).Add(htmlPipelineTable())),
		spacer(5),
		thinRule(),
		spacer(4),
		row.New(31).Add(
			col.New(4).Add(text.New("Inline content", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: blue}),
				text.New("Paragraphs, headings, strong/emphasis, links, and colored text become text or rich text runs.", props.Text{Top: 8, Right: 5, Size: 7.8, Color: ink, VerticalPadding: 1.1})),
			col.New(4).Add(text.New("Block layout", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: teal}),
				text.New("Flex rows and common block spacing are mapped onto Paper rows and grid columns.", props.Text{Top: 8, Right: 5, Size: 7.8, Color: ink, VerticalPadding: 1.1})),
			col.New(4).Add(text.New("Document fit", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 9, Color: amber}),
				text.New("Translated rows use the same page measurement, break, header, and footer behavior as hand-built rows.", props.Text{Top: 8, Right: 5, Size: 7.8, Color: ink, VerticalPadding: 1.1})),
		),
		pageBreak(),
		sectionTitle("HTML Rendered Output", "This fragment is parsed through pkg/components/html and rendered as Paper rows."),
		htmlShowcaseBlock(gridSize),
		spacer(5),
		row.New().Add(col.New(12).Add(htmlRenderedNotesTable())),
		spacer(5),
		thinRule(),
		spacer(4),
		row.New(32).Add(
			col.New(6).Add(text.New("Why this page matters", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 10, Color: brand}),
				text.New("The HTML example is not a screenshot. It is parsed into Paper components and rendered by the same provider pipeline as the rest of the document.", props.Text{Top: 9, Right: 7, Size: 8, Color: ink, VerticalPadding: 1.1})),
			col.New(6).Add(text.New("What to inspect", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 10, Color: teal}),
				text.New("The panel border, flex-style KPI row, table header, cell borders, and text wrapping all come from translated HTML and CSS.", props.Text{Top: 9, Right: 7, Size: 8, Color: ink, VerticalPadding: 1.1})),
		),
	}
}

func htmlShowcaseBlock(gridSize int) core.Row {
	component, err := htmlcomponent.New(context.Background(), `
<style>
.panel { border: 1px solid #e2dbd1; border-radius: 4px; padding: 6px; background: #fcfaf7; }
.kpis { display: flex; gap: 6px; margin: 5px 0; }
.kpi { flex: 1; padding: 5px; background: #fbf0ec; border-left: 3px solid #be4f36; font-size: 10px; }
table { width: 100%; border-collapse: collapse; margin-top: 5px; font-size: 10px; }
th { background: #3a322b; color: white; font-size: 10px; }
td, th { border: 1px solid #e2dbd1; padding: 4px; }
</style>
<div class="panel">
  <h2 style="font-size:22px; margin:0 0 3px 0;">Rendered HTML Fragment</h2>
  <p style="font-size:10px; margin:0 0 5px 0;">Paper translates supported tags and CSS into the same row, column, text, table, and style primitives used elsewhere in this PDF.</p>
  <div class="kpis">
    <div class="kpi"><strong>Blocks</strong><br>headings, paragraphs, lists</div>
    <div class="kpi"><strong>Layout</strong><br>tables and flex rows</div>
    <div class="kpi"><strong>Styles</strong><br>colors, borders, spacing</div>
  </div>
  <table>
    <tr><th>HTML</th><th>Paper output</th></tr>
    <tr><td>&lt;p&gt;, &lt;strong&gt;, &lt;em&gt;</td><td>rich text runs</td></tr>
    <tr><td>display:flex</td><td>grid columns</td></tr>
    <tr><td>table</td><td>table component cells</td></tr>
  </table>
</div>`, htmlcomponent.WithGridSize(gridSize), htmlcomponent.WithContentWidth(176))
	if err != nil {
		panic(err)
	}
	return row.New().Add(col.New(12).Add(component))
}

func htmlPipelineTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Stage", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Input", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Paper primitive", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Parse", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("DOM nodes and CSS declarations", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Intermediate block tree", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Translate", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Blocks, flex, tables, inline runs", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("Rows, columns, text, rich text, tables", false, ink), Style: tableCellStyle(softTeal)},
		},
		{
			{Content: tableText("Render", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Measured Paper rows", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("PDF cells on the page", false, ink), Style: tableCellStyle(nil)},
		},
	}, table.WithColumnWidths([]float64{0.9, 2.1, 2.1}))
	if err != nil {
		panic(err)
	}
	return t
}

func htmlRenderedNotesTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Rendered feature", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Visible result", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Backed by", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Panel styling", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Border, radius, padding, and background.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Cell styles", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Flex KPI row", false, ink), Style: tableCellStyle(softBlue)},
			{Content: tableText("Three equal columns with gaps.", false, ink), Style: tableCellStyle(softBlue)},
			{Content: tableText("Grid columns", false, ink), Style: tableCellStyle(softBlue)},
		},
		{
			{Content: tableText("HTML table", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Header row, borders, and data rows.", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Table component", false, ink), Style: tableCellStyle(nil)},
		},
	}, table.WithColumnWidths([]float64{1.3, 2.15, 1.4}))
	if err != nil {
		panic(err)
	}
	return t
}

func testingMetricsRows() []core.Row {
	return []core.Row{
		sectionTitle("Testing and Metrics", "Inspect the component tree before rendering, then track generation behavior after each run."),
		row.New(74).Add(
			treeSnapshotCol(7),
			metricsSnapshotCol(5),
		),
		spacer(6),
		row.New(28).Add(
			cardCol(4, "Deterministic snapshots", "Assert structure without image diffs. Rows, columns, tables, and codes all appear in the tree.", softBlue, blue),
			cardCol(4, "Generation timing", "Optional metrics expose build phases, add-row timings, page creation, and output size.", softTeal, teal),
			cardCol(4, "Visual verification", "Render the final PDF pages when layout or assets change, then keep the source as normal Go.", softGold, amber),
		),
	}
}

func treeSnapshotCol(size int) core.Col {
	lines := []string{
		"Document a4 portrait",
		"+-- Header",
		"|   +-- Row [12]",
		"+-- Row [8,4]",
		"|   +-- Text \"Invoice\" ok",
		"|   +-- QRCode 88x88 ok",
		"+-- Table 4 rows",
		"+-- Footer PageNumber",
	}
	components := []core.Component{
		text.New("doc.Tree() - deterministic snapshot", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 7, Top: 4, Left: 5, Color: muted}),
	}
	for i, line := range lines {
		color := inkSoft
		if strings.Contains(line, "Document") || strings.Contains(line, "Text") || strings.Contains(line, "Table") {
			color = blue
		}
		components = append(components, text.New(line, props.Text{
			Family: consts.FontFamilyCourier,
			Size:   7,
			Top:    14 + float64(i)*5.2,
			Left:   6,
			Color:  color,
		}))
	}
	return col.New(size).Add(components...).WithStyle(codePanelStyle())
}

func metricsSnapshotCol(size int) core.Col {
	return col.New(size).Add(
		text.New("paper.Metrics() - last run", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 7, Top: 4, Left: 5, Color: muted}),
		text.New("18 ms", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 18, Top: 16, Left: 7, Color: ink}),
		text.New("generation time", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 28, Left: 7, Color: faint}),
		text.New("7", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 18, Top: 16, Left: 39, Color: ink}),
		text.New("components", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 28, Left: 39, Color: faint}),
		text.New("1", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 18, Top: 42, Left: 7, Color: ink}),
		text.New("page", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 54, Left: 7, Color: faint}),
		text.New("42 kb", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 18, Top: 42, Left: 39, Color: ink}),
		text.New("output size", props.Text{Family: consts.FontFamilyCourier, Size: 5.8, Top: 54, Left: 39, Color: faint}),
		text.New("throughput  1,240 docs/sec", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 7, Top: 65, Left: 7, Color: teal}),
	).WithStyle(cardStyle(white, borderC))
}

func useCaseRows() []core.Row {
	return []core.Row{
		sectionTitle("Use Cases", "Built for the documents Go services generate on demand and at volume."),
		row.New(44).Add(
			useCaseCard(2, "Invoices", "totals, tax, QR pay"),
			useCaseCard(2, "Reports", "tables, charts, pages"),
			useCaseCard(2, "Forms", "checks, fields, signs"),
			useCaseCard(2, "Labels", "barcodes, batches"),
			useCaseCard(2, "Certificates", "signatures, seals"),
			useCaseCard(2, "Contracts", "clauses, appendices"),
		),
		spacer(6),
		row.New(36).Add(
			cardCol(3, "Production APIs", "Return bytes, base64 payloads, or files from the same document object.", softBlue, blue),
			cardCol(3, "Document automation", "Generate customer-facing PDFs from queue workers and service jobs.", softTeal, teal),
			cardCol(3, "HTML migration", "Reuse existing markup while moving toward typed component layouts.", softGold, amber),
			cardCol(3, "Audit-friendly", "Keep the generated structure testable and repeatable across releases.", softRose, red),
		),
		spacer(5),
		row.New().Add(col.New(12).Add(overviewMatrixTable())),
	}
}

func useCaseCard(size int, name, detail string) core.Col {
	return col.New(size).Add(
		text.New(name, props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 8.2, Top: 23, Align: consts.AlignCenter, Color: ink}),
		text.New(detail, props.Text{Family: consts.FontFamilyCourier, Size: 5.7, Top: 30, Align: consts.AlignCenter, Left: 2, Right: 2, Color: muted}),
		text.New("PDF", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 15, Top: 7, Align: consts.AlignCenter, Color: teal}),
	).WithStyle(&props.Cell{
		BackgroundColor: white,
		BorderType:      border.Full,
		BorderColor:     borderC,
		BorderThickness: 0.25,
		BorderRadius:    2,
		MarginLeft:      0.6,
		MarginRight:     0.6,
	})
}

func invoiceExampleRows() []core.Row {
	return []core.Row{
		sectionTitle("Generated PDF Example", "A single invoice-style page using the same visual language as the showcase hero."),
		row.New(36).Add(
			col.New(7).Add(
				text.New("Paper Labs, Inc.", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 14, Top: 5, Left: 3, Color: ink}),
				text.New("120 Foundry Street, Suite 4 / Brooklyn, NY 11222 / hello@paperlabs.dev", props.Text{Family: showcaseFont, Size: 7, Top: 15, Left: 3, Right: 8, Color: muted, VerticalPadding: 1.12}),
			),
			col.New(3).Add(
				text.New("Invoice", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 22, Top: 4, Align: consts.AlignRight, Right: 2, Color: ink}),
				text.New("NO. INV-2026-0481", props.Text{Family: consts.FontFamilyCourier, Size: 7, Top: 21, Align: consts.AlignRight, Right: 2, Color: teal}),
			),
			col.New(2).Add(code.NewQr("pay.paperlabs.dev/inv-0481", props.Rect{Percent: 45, Center: true})),
		).WithStyle(sheetHeaderStyle()),
		spacer(5),
		row.New(28).Add(
			invoiceMetaCol(3, "Billed to", "Northwind Trading Co. / Accounts Payable"),
			invoiceMetaCol(3, "Issue date", "June 8, 2026 / Due July 8, 2026"),
			invoiceMetaCol(3, "Terms", "Net 30 / PO-77-4521"),
			invoiceMetaCol(3, "Currency", "USD ($) / Account NW-00231"),
		),
		spacer(5),
		row.New().Add(col.New(12).Add(invoiceItemsTable())),
		spacer(6),
		row.New(45).Add(
			col.New(6).Add(
				text.New("Payment details", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 7, Top: 3, Left: 3, Color: faint}),
				text.New("Wire transfer or ACH within 30 days. Reference the invoice number on all remittances.", props.Text{Family: showcaseFont, Size: 7.4, Top: 12, Left: 3, Right: 8, Color: muted, VerticalPadding: 1.2}),
				code.NewBar("INV-2026-0481", props.Barcode{Top: 29, Left: 3, Percent: 46, Proportion: props.Proportion{Width: 12, Height: 1.4}}),
			),
			col.New(6).Add(invoiceTotalsTable()),
		),
		spacer(5),
		row.New(38).Add(
			col.New(6).Add(
				text.New("Avdo S.", props.Text{Family: showcaseFont, Style: fontstyle.Italic, Size: 20, Top: 6, Left: 3, Color: ink}),
				text.New("Authorized - Paper Labs", props.Text{Family: consts.FontFamilyCourier, Size: 6, Top: 25, Left: 3, Color: faint}),
			),
			col.New(6).Add(
				checkbox.New("Goods and services received in full", props.Checkbox{Checked: true, Top: 7, Left: 3, Size: 3.8}),
				checkbox.New("Amount verified against PO-77-4521", props.Checkbox{Checked: true, Top: 17, Left: 3, Size: 3.8}),
				checkbox.New("Recurring billing authorized", props.Checkbox{Top: 27, Left: 3, Size: 3.8}),
			),
		).WithStyle(sheetFooterStyle()),
	}
}

func invoiceMetaCol(size int, label, value string) core.Col {
	return col.New(size).Add(
		text.New(label, props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 6, Top: 3, Left: 3, Color: faint}),
		text.New(value, props.Text{Family: showcaseFont, Size: 7.4, Top: 10, Left: 3, Right: 3, Color: inkSoft, VerticalPadding: 1.15}),
	).WithStyle(cardStyle(white, borderC))
}

func invoiceItemsTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Description", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Qty", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Unit price", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Amount", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Document API - Team plan", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("1", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("$480.00", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("$480.00", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Rendering credits", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("12k", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("$0.02", false, ink), Style: tableCellStyle(softTeal)},
			{Content: tableText("$240.00", false, ink), Style: tableCellStyle(softTeal)},
		},
		{
			{Content: tableText("Priority support", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("1", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("$120.00", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("$120.00", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Custom font embedding", false, ink), Style: tableCellStyle(softGold)},
			{Content: tableText("3", false, ink), Style: tableCellStyle(softGold)},
			{Content: tableText("$30.00", false, ink), Style: tableCellStyle(softGold)},
			{Content: tableText("$90.00", false, ink), Style: tableCellStyle(softGold)},
		},
	}, table.WithColumnWidths([]float64{2.7, 0.6, 0.9, 0.9}))
	if err != nil {
		panic(err)
	}
	return t
}

func invoiceTotalsTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Subtotal", false, muted), Style: tableCellStyle(nil)},
			{Content: tableText("$930.00", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Volume discount (5%)", false, muted), Style: tableCellStyle(softTeal)},
			{Content: tableText("-$46.50", false, green), Style: tableCellStyle(softTeal)},
		},
		{
			{Content: tableText("Tax (NY 8.875%)", false, muted), Style: tableCellStyle(nil)},
			{Content: tableText("$78.41", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("Total due", true, white), Style: tableHeaderStyle()},
			{Content: tableText("$961.91", true, white), Style: tableHeaderStyle()},
		},
	}, table.WithColumnWidths([]float64{1.6, 1.0}))
	if err != nil {
		panic(err)
	}
	return t
}

func sheetHeaderStyle() *props.Cell {
	return &props.Cell{
		BackgroundColor: white,
		BorderType:      border.Full,
		BorderColor:     borderC,
		BorderThickness: 0.25,
		BorderRadius:    2,
		PaddingTop:      2,
		PaddingBottom:   2,
	}
}

func sheetFooterStyle() *props.Cell {
	return &props.Cell{
		BackgroundColor:    white,
		BorderTopColor:     borderC,
		BorderTopThickness: 0.3,
		PaddingTop:         2,
	}
}

func installRows() []core.Row {
	return []core.Row{
		row.New(96).Add(
			col.New(12).Add(
				text.New("Install Paper", props.Text{Family: showcaseFont, Style: fontstyle.Bold, Size: 26, Top: 18, Align: consts.AlignCenter, Color: white}),
				text.New("Ship documents that render exactly right.", props.Text{Family: showcaseFont, Style: fontstyle.Italic, Size: 15, Top: 39, Align: consts.AlignCenter, Color: rgb(231, 166, 139)}),
				text.New("Add Paper to your Go service and generate production PDFs in a single import.", props.Text{Family: showcaseFont, Size: 8.6, Top: 55, Align: consts.AlignCenter, Color: rgb(207, 198, 188)}),
				text.New("$ go get github.com/avdoseferovic/paper", props.Text{Family: consts.FontFamilyCourier, Style: fontstyle.Bold, Size: 10, Top: 72, Align: consts.AlignCenter, Color: white}),
			),
		).WithStyle(&props.Cell{
			BackgroundColor: deepInk,
			BorderRadius:    3,
		}),
		spacer(8),
		row.New(34).Add(
			cardCol(4, "Docs", "Read examples for HTML, components, tables, codes, and generated output.", softBlue, blue),
			cardCol(4, "GitHub", "Use the repository examples as executable documentation for your service.", softTeal, teal),
			cardCol(4, "MIT License", "Build PDFs in Go without adding a browser runtime to production jobs.", softGold, amber),
		),
	}
}

func htmlSupportTable() core.Component {
	t, err := table.New([][]table.Cell{
		{
			{Content: tableText("Input", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Rendered as", true, white), Style: tableHeaderStyle()},
			{Content: tableText("Notes", true, white), Style: tableHeaderStyle()},
		},
		{
			{Content: tableText("Headings, paragraphs, inline tags", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Text and rich text runs", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Supports bold, emphasis, links, colors, and spacing.", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("display:flex", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Rows and columns", false, ink), Style: tableCellStyle(nil)},
			{Content: tableText("Flex widths quantize to the configured grid size.", false, ink), Style: tableCellStyle(nil)},
		},
		{
			{Content: tableText("table, border, background", false, ink), Style: tableCellStyle(softBlue)},
			{Content: tableText("Table cells and cell styles", false, ink), Style: tableCellStyle(softBlue)},
			{Content: tableText("Useful for reports, invoices, and rich document fragments.", false, ink), Style: tableCellStyle(softBlue)},
		},
	}, table.WithColumnWidths([]float64{1.35, 1.25, 2.2}))
	if err != nil {
		panic(err)
	}
	return t
}

type gridCellSpec struct {
	size  int
	label string
	color *props.Color
}

func gridExampleRow(label string, cells []gridCellSpec) core.Row {
	return row.New(12).Add(
		col.New(3).Add(text.New(label, props.Text{
			Family: consts.FontFamilyCourier,
			Size:   6.5,
			Top:    2,
			Color:  muted,
		})),
		col.New(9).Add(rowStrip(cells)),
	)
}

func rowStrip(cells []gridCellSpec) core.Component {
	t, err := table.New(gridCells(cells))
	if err != nil {
		panic(err)
	}
	return t
}

func gridCells(cells []gridCellSpec) [][]table.Cell {
	out := make([]table.Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, table.Cell{
			Content: text.New(cell.label, props.Text{
				Family: showcaseFont,
				Style:  fontstyle.Bold,
				Size:   8,
				Align:  consts.AlignCenter,
				Color:  white,
			}),
			Colspan: cell.size,
			Style: &props.Cell{
				BackgroundColor: cell.color,
				BorderType:      border.Full,
				BorderColor:     white,
				BorderThickness: 0.35,
				PaddingTop:      2.2,
				PaddingBottom:   2.2,
			},
		})
	}
	return [][]table.Cell{out}
}

func tableHeaderStyle() *props.Cell {
	return &props.Cell{
		BackgroundColor: deepInk,
		BorderType:      border.Full,
		BorderColor:     deepInk,
		PaddingTop:      2,
		PaddingLeft:     2,
		PaddingBottom:   2,
	}
}

func tableCellStyle(background *props.Color) *props.Cell {
	return &props.Cell{
		BackgroundColor: background,
		BorderType:      border.Full,
		BorderColor:     lineSoftC,
		BorderThickness: 0.25,
		PaddingTop:      1.8,
		PaddingLeft:     2,
		PaddingBottom:   1.8,
	}
}

func tableText(value string, bold bool, color *props.Color) core.Component {
	style := fontstyle.Normal
	if bold {
		style = fontstyle.Bold
	}
	return text.New(value, props.Text{
		Family: showcaseFont,
		Style:  style,
		Size:   7.5,
		Color:  color,
	})
}

func cardTitle(title string, color *props.Color) core.Component {
	return text.New(title, props.Text{
		Family: showcaseFont,
		Style:  fontstyle.Bold,
		Size:   10,
		Top:    3,
		Left:   4,
		Color:  color,
	})
}

func cardCol(size int, title, body string, bg, accent *props.Color) core.Col {
	return col.New(size).Add(
		cardTitle(title, accent),
		text.New(body, props.Text{
			Family:          showcaseFont,
			Size:            8,
			Top:             11,
			Left:            4,
			Right:           4,
			VerticalPadding: 1.2,
			Color:           ink,
		}),
	).WithStyle(cardStyle(bg, borderC))
}

func sectionTitle(title, subtitle string) core.Row {
	return row.New(17).Add(col.New(12).Add(
		text.New(title, props.Text{
			Family: showcaseFont,
			Style:  fontstyle.Bold,
			Size:   16,
			Top:    1,
			Color:  brand,
		}),
		text.New(subtitle, props.Text{
			Family: showcaseFont,
			Size:   8.5,
			Top:    10,
			Color:  muted,
		}),
	))
}

func thinRule() core.Row {
	return row.New(1.2).Add(col.New(12).Add(line.New(props.Line{
		Color:     borderC,
		Thickness: 0.25,
	})))
}

func infoCol(size int, title, body string, accent *props.Color) core.Col {
	return col.New(size).Add(
		text.New(title, props.Text{
			Family: showcaseFont,
			Style:  fontstyle.Bold,
			Size:   9,
			Top:    3,
			Color:  accent,
		}),
		text.New(body, props.Text{
			Family:          showcaseFont,
			Size:            8,
			Top:             12,
			Right:           5,
			VerticalPadding: 1.1,
			Color:           ink,
		}),
	)
}

func cardStyle(bg, stroke *props.Color) *props.Cell {
	return &props.Cell{
		BackgroundColor: bg,
		BorderType:      border.Full,
		BorderColor:     stroke,
		BorderThickness: 0.25,
		BorderRadius:    2,
		MarginLeft:      0.9,
		MarginRight:     0.9,
	}
}

func spacer(height float64) core.Row {
	return row.New(height).Add(col.New(12))
}

func pageBreak() core.Row {
	return translate.NewPageBreakRow()
}

func rgb(r, g, b int) *props.Color {
	return &props.Color{Red: r, Green: g, Blue: b}
}
