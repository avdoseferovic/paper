package paper_test

import (
	"log"

	"github.com/johnfercher/paper/v2/pkg/components/text"

	"github.com/johnfercher/paper/v2"
	"github.com/johnfercher/paper/v2/pkg/components/code"
	"github.com/johnfercher/paper/v2/pkg/components/page"
	"github.com/johnfercher/paper/v2/pkg/config"
)

// ExampleNew demonstrates how to create a paper instance.
func ExampleNew() {
	// optional
	b := config.NewBuilder()
	cfg := b.Build()

	m := paper.New(cfg) // cfg is an optional

	// Do things and generate
	_, _ = m.Generate()
}

// ExampleNewMetricsDecorator demonstrates how to create a paper metrics decorator instance.
func ExampleNewMetricsDecorator() {
	// optional
	b := config.NewBuilder()
	cfg := b.Build()

	mrt := paper.New(cfg)               // cfg is an optional
	m := paper.NewMetricsDecorator(mrt) // decorator of paper

	// Do things and generate
	_, _ = m.Generate()
}

// ExampleFromHTML demonstrates the shortest path from HTML to PDF.
func ExampleFromHTML() {
	doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)
	if err != nil {
		log.Fatal(err)
	}

	_ = doc.GetBytes()
}

// ExamplePaper_AddPages demonstrates how to add a new page in paper.
func ExamplePaper_AddPages() {
	m := paper.New()

	p := page.New()
	p.Add(code.NewBarRow(10, "barcode"))

	m.AddPages(p)

	// Do things and generate
}

// ExamplePaper_AddRows demonstrates how to add new rows in paper.
func ExamplePaper_AddRows() {
	m := paper.New()

	m.AddRows(
		code.NewBarRow(12, "barcode"),
		text.NewRow(12, "text"),
	)

	// Do things and generate
}

// ExamplePaper_AddRow demonstrates how to add a new row in paper.
func ExamplePaper_AddRow() {
	m := paper.New()

	m.AddRow(10, text.NewCol(12, "text"))

	// Do things and generate
}

// ExamplePaper_FitlnCurrentPage demonstrate how to check if the new line fits on the current page
func ExamplePaper_FitlnCurrentPage() {
	m := paper.New()

	m.FitlnCurrentPage(12)

	// Do things and generate
}

// ExamplePaper_FitlnCurrentPage demonstrate how to check if the new line fits on the current page
func ExamplePaper_GetCurrentConfig() {
	m := paper.New()

	m.GetCurrentConfig()

	// Do things and generate
}

// ExamplePaper_RegisterHeader demonstrates how to register a header to me added in every new page.
// An error is returned if the area occupied by the header is greater than the page area.
func ExamplePaper_RegisterHeader() {
	m := paper.New()

	err := m.RegisterHeader(
		code.NewBarRow(12, "barcode"),
		text.NewRow(12, "text"))
	if err != nil {
		panic(err)
	}

	// Do things and generate
}

// ExamplePaper_RegisterFooter demonstrates how to register a footer to me added in every new page.
// An error is returned if the area occupied by the footer is greater than the page area.
func ExamplePaper_RegisterFooter() {
	m := paper.New()

	err := m.RegisterFooter(
		code.NewBarRow(12, "barcode"),
		text.NewRow(12, "text"))
	if err != nil {
		panic(err)
	}

	// Do things and generate
}

// ExamplePaper_Generate demonstrates how to generate a file.
func ExamplePaper_Generate() {
	m := paper.New()

	// Add rows, pages and etc.

	doc, err := m.Generate()
	if err != nil {
		log.Fatal(err)
	}

	// You can retrieve as Base64, Save file, Merge with another file or GetReport.
	_ = doc.GetBytes()
}

// ExamplePaperGetStruct demonstrates how to get paper component tree
func ExamplePaper_GetStructure() {
	m := paper.New()

	m.AddRow(40, text.NewCol(12, "text"))

	m.GetStructure()

	// Do things and generate
}
