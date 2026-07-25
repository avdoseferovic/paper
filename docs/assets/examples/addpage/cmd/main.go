// Command addpage writes the addpage example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/addpage/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/addpage"
)

func main() {
	m := addpage.GetPaper()

	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/addpage.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/addpage.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
