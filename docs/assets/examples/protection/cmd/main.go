// Command protection writes the protection example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/protection/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/protection"
)

func main() {
	m := example.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/protection.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/protection.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
