// survey-report generates a styled patient anamnesis report PDF from the
// embedded body.html. The report uses gradients, shadows, outlines,
// dashed borders, multi-column flex, and styled tables — every Plan B
// feature in one realistic document.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/examples/internal/examplepath"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/line"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/align"
	"github.com/avdoseferovic/paper/pkg/consts/fontfamily"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(12).
		WithBottomMargin(12).
		Build()

	m := paper.New(cfg)
	if err := m.RegisterHeader(buildHeader()...); err != nil {
		log.Fatalf("register header: %v", err)
	}
	if err := m.RegisterFooter(buildFooter()...); err != nil {
		log.Fatalf("register footer: %v", err)
	}

	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	assetsDir := examplepath.Module("cmd/survey-report/assets")
	rows, err := html.FromString(body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir(assetsDir),
		html.WithStylesheetBaseDir(assetsDir),
	)
	if err != nil {
		log.Fatalf("parse html: %v", err)
	}
	m.AddRows(rows...)

	doc, err := m.Generate()
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	out := examplepath.Repo("test/output/survey-report.pdf")
	if err := examplepath.EnsureParent(out); err != nil {
		log.Fatalf("create output directory: %v", err)
	}
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
		col.New(7).Add(text.New("PAPER MEDICAL CENTER", props.Text{
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
		col.New(8).Add(text.New("Document ref: ANAM-2026-04812 · Paper Medical Center", props.Text{
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
