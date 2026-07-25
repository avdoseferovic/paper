// Command checkbox writes the checkbox example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/checkbox/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/checkbox"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/checkbox.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/checkbox.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
