package gofpdfwrapper_test

import (
	"fmt"
	"testing"

	gofpdf "github.com/johnfercher/paper/v2/internal/paperpdf"
	"github.com/johnfercher/paper/v2/internal/providers/paper/gofpdfwrapper"
	"github.com/stretchr/testify/assert"
)

func TestNewCustom(t *testing.T) {
	t.Parallel()
	// Act
	sut := gofpdfwrapper.NewCustom(&gofpdf.InitType{})

	// Assert
	assert.Equal(t, "*paperpdf.Fpdf", fmt.Sprintf("%T", sut))
}
