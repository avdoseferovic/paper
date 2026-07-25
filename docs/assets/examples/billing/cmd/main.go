// Command billing writes the billing example's PDF into docs/assets/pdf.
// Run it from the repository root: go run ./docs/assets/examples/billing/cmd
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/docs/assets/examples/billing"
)

func main() {
	m := billing.GetPaper()
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/billing.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/billing.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}
