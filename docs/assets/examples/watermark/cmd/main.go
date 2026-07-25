// Command watermark writes the watermark example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/watermark/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/watermark"
)

func main() {
	m := example.GetPaper()

	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/watermark.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}
}
