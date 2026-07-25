// Command lowmemory writes the lowmemory example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/lowmemory/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/lowmemory"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/lowmemory.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/lowmemory.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
