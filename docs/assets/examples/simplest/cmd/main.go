// Command simplest writes the simplest example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/simplest/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/simplest"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	err = document.Save("docs/assets/pdf/simplest.pdf")
	if err != nil {
		log.Fatal(err)
	}
}
