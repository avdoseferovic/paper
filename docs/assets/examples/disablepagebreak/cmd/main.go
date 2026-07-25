// Command disablepagebreak writes the disablepagebreak example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/disablepagebreak/cmd
package main

import (
	"context"
	"log"

	example "github.com/avdoseferovic/paper/docs/assets/examples/disablepagebreak"
)

func main() {
	backgroundImage := "docs/assets/images/certificate.png"
	m := example.GetPaper(backgroundImage)
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/disablepagebreak.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/disablepagebreak.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
