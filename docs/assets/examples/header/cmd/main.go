// Command header writes the header example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/header/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/header"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/header.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/header.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
