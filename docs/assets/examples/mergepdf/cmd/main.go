// Command mergepdf writes the mergepdf example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/mergepdf/cmd
package main

import (
	"context"
	"log"
	"os"

	example "github.com/avdoseferovic/paper/docs/assets/examples/mergepdf"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	savedPdf, err := os.ReadFile("docs/assets/pdf/paper.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Merge(context.Background(), savedPdf)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/mergepdf.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/mergepdf.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
