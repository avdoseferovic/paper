package main_test

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/components/line"

	"github.com/avdoseferovic/paper"

	"github.com/avdoseferovic/paper/pkg/components/code"
	"github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/signature"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/test"
)

func TestPaper_GetStructure(t *testing.T) {
	t.Parallel()
	// Arrange
	m := paper.New()

	m.AddRow(10,
		code.NewBarCol(4, "barcode"),
		code.NewMatrixCol(4, "matrixcode"),
		code.NewQrCol(4, "qrcode"),
	)

	m.AddRow(10,
		image.NewFromFileCol(3, "image"),
		image.NewFromBytesCol(3, []byte{0, 1, 2}, extension.Png),
		signature.NewCol(3, "signature"),
		text.NewCol(3, "text"),
	)

	m.AddRow(10, line.NewCol(12))

	// Assert
	test.New(t).Assert(m.GetStructure()).Equals("example_unit_test.json")
}
