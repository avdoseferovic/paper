package gofpdfwrapper_test

import (
	"fmt"
	"testing"

	gofpdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/stretchr/testify/assert"
)

func TestNewCustom(t *testing.T) {
	t.Parallel()
	// Act
	sut := gofpdfwrapper.NewCustom(&gofpdf.InitType{})

	// Assert
	assert.Equal(t, "*pdf.PDF", fmt.Sprintf("%T", sut))
}
