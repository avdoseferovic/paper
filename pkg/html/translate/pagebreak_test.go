package translate_test

import (
	"testing"

	"github.com/johnfercher/paper/v2/pkg/core"
	"github.com/johnfercher/paper/v2/pkg/html/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageBreakRow_IsPageBreak(t *testing.T) {
	t.Parallel()
	row := translate.NewPageBreakRow()
	// Must implement core.PageBreaker
	pb, ok := row.(core.PageBreaker)
	require.True(t, ok, "pageBreakRow must implement core.PageBreaker")
	assert.True(t, pb.IsPageBreak())
}

func TestPageBreakRow_GetHeight_IsZero(t *testing.T) {
	t.Parallel()
	row := translate.NewPageBreakRow()
	// GetHeight must return 0 (no real content)
	h := row.GetHeight(nil, nil)
	assert.Equal(t, 0.0, h)
}

func TestTranslate_PageBreakAfter_ProducesBreakRow(t *testing.T) {
	t.Parallel()
	doc := parseDoc(t, `<html><body>
	<div style="page-break-after:always">Section 1</div>
	<p>Section 2</p>
	</body></html>`)
	rows, err := translate.Translate(doc)
	require.NoError(t, err)
	// There should be at least one PageBreaker row in the output
	var foundBreak bool
	for _, r := range rows {
		if pb, ok := r.(core.PageBreaker); ok && pb.IsPageBreak() {
			foundBreak = true
			break
		}
	}
	assert.True(t, foundBreak, "translate should emit a PageBreaker row after page-break-after:always")
}

func TestTranslate_PageBreakBefore_ProducesBreakRow(t *testing.T) {
	t.Parallel()
	doc := parseDoc(t, `<html><body>
	<p>Section 1</p>
	<div style="page-break-before:always">Section 2</div>
	</body></html>`)
	rows, err := translate.Translate(doc)
	require.NoError(t, err)
	var foundBreak bool
	for _, r := range rows {
		if pb, ok := r.(core.PageBreaker); ok && pb.IsPageBreak() {
			foundBreak = true
			break
		}
	}
	assert.True(t, foundBreak, "translate should emit a PageBreaker row before page-break-before:always")
}
