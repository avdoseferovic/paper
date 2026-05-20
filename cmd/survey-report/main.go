// survey-report generates a styled patient anamnesis report PDF from the
// embedded body.html. The report uses gradients, shadows, outlines,
// dashed borders, multi-column flex, and styled tables — every Plan B
// feature in one realistic document.
package main

import (
	"fmt"
	"log"
	"os"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

func main() {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(12).
		WithBottomMargin(12).
		Build()

	m := maroto.New(cfg)
	if err := m.RegisterHeader(buildHeader()...); err != nil {
		log.Fatalf("register header: %v", err)
	}
	if err := m.RegisterFooter(buildFooter()...); err != nil {
		log.Fatalf("register footer: %v", err)
	}

	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	rows, err := html.FromString(body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir("cmd/survey-report/assets"),
		html.WithStylesheetBaseDir("cmd/survey-report/assets"),
	)
	if err != nil {
		log.Fatalf("parse html: %v", err)
	}
	m.AddRows(rows...)

	doc, err := m.Generate()
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	out := "/Users/avdo/maroto/test/output/survey-report.pdf"
	if err := doc.Save(out); err != nil {
		log.Fatalf("save: %v", err)
	}
	info, _ := os.Stat(out)
	fmt.Printf("Wrote %s (%d bytes)\n", out, info.Size())
}

// buildHeader produces the running page header.
func buildHeader() []core.Row {
	brand := &props.Color{Red: 22, Green: 58, Blue: 95}
	muted := &props.Color{Red: 110, Green: 119, Blue: 133}

	title := row.New(7).Add(
		col.New(7).Add(text.New("MAROTO MEDICAL CENTER", props.Text{
			Family: fontfamily.Helvetica,
			Style:  fontstyle.Bold,
			Size:   9,
			Color:  brand,
		})),
		col.New(5).Add(text.New("CONFIDENTIAL · PATIENT RECORD", props.Text{
			Family: fontfamily.Helvetica,
			Style:  fontstyle.Bold,
			Size:   7,
			Align:  align.Right,
			Color:  muted,
		})),
	)
	rule := row.New(1.5).Add(col.New(12).Add(line.New(props.Line{Color: brand, Thickness: 0.5})))
	return []core.Row{title, rule}
}

// buildFooter produces a slim footer rendered on every page.
func buildFooter() []core.Row {
	muted := &props.Color{Red: 110, Green: 119, Blue: 133}
	rule := row.New(0.6).Add(col.New(12).Add(line.New(props.Line{
		Color: &props.Color{Red: 214, Green: 220, Blue: 228}, Thickness: 0.25,
	})))
	footRow := row.New(5).Add(
		col.New(8).Add(text.New("Document ref: ANAM-2026-04812 · Maroto Medical Center", props.Text{
			Family: fontfamily.Helvetica,
			Size:   7,
			Color:  muted,
		})),
		col.New(4).Add(text.New("Page {current} of {total}", props.Text{
			Family: fontfamily.Helvetica,
			Size:   7,
			Align:  align.Right,
			Color:  muted,
		})),
	)
	return []core.Row{rule, footRow}
}
