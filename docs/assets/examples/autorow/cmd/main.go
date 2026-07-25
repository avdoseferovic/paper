// Command autorow writes the autorow example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/autorow/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/autorow"
)

func main() {
	m := autorow.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/autorow.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/autorow.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
