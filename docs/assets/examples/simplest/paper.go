// Package simplest demonstrates the smallest document: one row with one line of text.
package simplest

import (
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

// GetPaper builds the simplest example document.
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
