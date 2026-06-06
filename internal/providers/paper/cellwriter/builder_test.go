package cellwriter_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/providers/paper/cellwriter"

	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	t.Parallel()
	// Act
	sut := cellwriter.NewBuilder()

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*cellwriter.WriterBuilder", fmt.Sprintf("%T", sut))
}

func TestCellWriterBuilder_Build(t *testing.T) {
	t.Parallel()
	// Arrange
	sut := cellwriter.NewBuilder()

	// Act
	chain := sut.Build(nil)

	// Assert: shadow first, outline before cellWriter, no gradient styler (nil drawer)
	assert.Equal(t, "shadowStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "perSideBorderStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "borderRadiusStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "borderThicknessStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "borderLineStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "borderColorStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "fillColorStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "backgroundImageStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "outlineStyler", chain.GetName())
	chain = chain.GetNext()
	assert.Equal(t, "cellWriter", chain.GetName())
	chain = chain.GetNext()
	assert.Nil(t, chain)
}
