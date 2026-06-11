package translate_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/translate"
)

func TestPageBreakRow_RenderAndSetConfig_AreNoOps(t *testing.T) {
	t.Parallel()
	row := translate.NewPageBreakRow()

	// Both calls must be safe with nil/zero arguments — the row carries no content.
	row.Render(nil, entity.Cell{})
	row.SetConfig(nil)
}

func TestPageBreakRow_GetStructure(t *testing.T) {
	t.Parallel()
	row := translate.NewPageBreakRow()
	str := row.GetStructure()
	assert.NotNil(t, str)
	assert.Equal(t, "page_break", str.GetData().Type)
}

func TestPageBreakRow_RowMethods_AreNoOps(t *testing.T) {
	t.Parallel()
	row := translate.NewPageBreakRow()

	assert.Equal(t, row, row.Add())
	assert.Equal(t, row, row.WithStyle(nil))
	assert.Nil(t, row.GetColumns())
}
