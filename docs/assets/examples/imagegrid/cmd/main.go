// Command imagegrid writes the imagegrid example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/imagegrid/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/imagegrid"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/imagegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/imagegrid.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
