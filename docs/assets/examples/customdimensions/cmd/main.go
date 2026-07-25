// Command customdimensions writes the customdimensions example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/customdimensions/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/customdimensions"
)

func main() {
	m := customdimensions.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/customdimensions.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/customdimensions.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
