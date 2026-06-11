// html-demo generates a rich HTML capability PDF from an HTML body parsed via pkg/html.
package main

import (
	"context"
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
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	cfg := config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := paper.New(cfg)
	m.AddRows(buildHeader()...)
	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	assetsDir := examplepath.Module("cmd/html-demo/assets")
	rows, err := html.FromString(context.Background(), body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir(assetsDir),
		html.WithStylesheetBaseDir(assetsDir),
	)
	if err != nil {
		log.Fatalf("parse html: %v", err)
	}
	m.AddRows(rows...)

	doc, err := m.Generate(context.Background())
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	out := examplepath.Repo("test/output/html-demo.pdf")
	if err := examplepath.EnsureParent(out); err != nil {
		log.Fatalf("create output directory: %v", err)
	}
	if err := doc.Save(out); err != nil {
		log.Fatalf("save: %v", err)
	}
	info, _ := os.Stat(out)
	fmt.Printf("Wrote %s (%d bytes)\n", out, info.Size())
}

// buildHeader returns the rows that render on every page above the body.
func buildHeader() []core.Row {
	dark := &props.Color{Red: 26, Green: 62, Blue: 114}
	muted := &props.Color{Red: 120, Green: 120, Blue: 120}

	titleRow := row.New(10).Add(
		col.New(8).Add(text.New("PAPER HTML PDF DEMO", props.Text{
			Family: consts.FontFamilyHelvetica,
			Style:  fontstyle.Bold,
			Size:   14,
			Color:  dark,
		})),
		col.New(4).Add(text.New("html-demo@paper.example", props.Text{
			Family: consts.FontFamilyHelvetica,
			Size:   9,
			Align:  consts.AlignRight,
			Color:  muted,
		})),
	)

	dividerRow := row.New(2).Add(col.New(12).Add(line.New(props.Line{
		Color:     dark,
		Thickness: 0.6,
	})))

	return []core.Row{titleRow, dividerRow}
}
