// Package main demonstrates the per-page text watermark: translucent
// diagonal text drawn under the content of every page.
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	cfg := config.NewBuilder().
		WithWatermark("DRAFT").
		Build()

	m := paper.New(cfg)
	for i := 0; i < 2; i++ {
		m.AddRow(250, col.New(12).Add(text.New("Body content flows over the watermark.", props.Text{Top: 5})))
	}

	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/watermark.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}
}
