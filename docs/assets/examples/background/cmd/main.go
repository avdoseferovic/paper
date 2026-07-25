// Command background writes the background example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/background/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/background"
)

func main() {
	backgroundImage := "docs/assets/images/certificate.png"
	m := background.GetPaper(backgroundImage)
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/background.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/background.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
