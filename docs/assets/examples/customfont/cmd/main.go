// Command customfont writes the customfont example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/customfont/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/customfont"
)

func main() {
	m := customfont.GetPaper("docs/assets/fonts/arial-unicode-ms.ttf")
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/customfont.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/customfont.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
