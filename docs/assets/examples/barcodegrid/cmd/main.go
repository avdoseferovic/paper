// Command barcodegrid writes the barcodegrid example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/barcodegrid/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/barcodegrid"
)

func main() {
	m := barcodegrid.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/barcodegrid.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/barcodegrid.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
