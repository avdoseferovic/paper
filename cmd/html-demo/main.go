// html-demo generates a rich HTML capability PDF from an HTML body parsed via pkg/html.
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
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)
	m.AddRows(buildHeader()...)
	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	rows, err := html.FromString(body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir("cmd/html-demo/assets"),
		html.WithStylesheetBaseDir("cmd/html-demo/assets"),
	)
	if err != nil {
		log.Fatalf("parse html: %v", err)
	}
	m.AddRows(rows...)

	doc, err := m.Generate()
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	out := "/Users/avdo/maroto/test/output/html-demo.pdf"
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
		col.New(8).Add(text.New("MAROTO HTML PDF DEMO", props.Text{
			Family: fontfamily.Helvetica,
			Style:  fontstyle.Bold,
			Size:   14,
			Color:  dark,
		})),
		col.New(4).Add(text.New("html-demo@maroto.example", props.Text{
			Family: fontfamily.Helvetica,
			Size:   9,
			Align:  align.Right,
			Color:  muted,
		})),
	)

	dividerRow := row.New(2).Add(col.New(12).Add(line.New(props.Line{
		Color:     dark,
		Thickness: 0.6,
	})))

	return []core.Row{titleRow, dividerRow}
}
