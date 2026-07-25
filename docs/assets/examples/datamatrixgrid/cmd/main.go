// Command datamatrixgrid writes the datamatrixgrid example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/datamatrixgrid/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/datamatrixgrid"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/datamatrixgrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/datamatrixgrid.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
