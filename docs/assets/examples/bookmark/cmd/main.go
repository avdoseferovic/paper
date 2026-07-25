// Command bookmark writes the bookmark example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/bookmark/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/bookmark"
)

func main() {
	m := bookmark.GetPaper()

	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/bookmark.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}
}
