package main

import (
	"log"

	"github.com/avdoseferovic/paper/pkg/components/checkbox"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/line"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/text"
)

func main() {
	m := GetPaper()
	document, err := m.Generate()
	if err != nil {
		log.Fatal(err)
	}

	err = document.Save("docs/assets/pdf/simplestv2.pdf")
	if err != nil {
		log.Fatal(err)
	}
}

func GetPaper() core.Paper {
	m := paper.New()

	m.AddRow(20,
		code.NewBarCol(4, "barcode"),
		code.NewMatrixCol(4, "matrixcode"),
		code.NewQrCol(4, "qrcode"),
	)

	m.AddRow(10, col.New(12))

	m.AddRow(20,
		image.NewFromFileCol(4, "docs/assets/images/biplane.jpg"),
		signature.NewCol(4, "signature"),
		text.NewCol(4, "text"),
	)

	m.AddRow(10, col.New(12))

	m.AddRow(20,
		checkbox.NewCol(12, "agree"),
	)

	m.AddRow(20, line.NewCol(12))

	return m
}
