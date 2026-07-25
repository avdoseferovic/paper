// Command metadatas writes the metadatas example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/metadatas/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/metadatas"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/metadatas.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/metadatas.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
