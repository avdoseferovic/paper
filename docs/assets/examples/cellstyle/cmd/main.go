// Command cellstyle writes the cellstyle example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/cellstyle/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/cellstyle"
)

func main() {
	m := cellstyle.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/cellstyle.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/cellstyle.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
