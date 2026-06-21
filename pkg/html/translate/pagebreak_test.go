package translate_test

import (
	"context"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/translate"
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
	rows, err := translate.Translate(context.Background(), doc)
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
	rows, err := translate.Translate(context.Background(), doc)
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

func TestTranslate_PaperBlankPageCustomProperty(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<div style="--paper-page: blank; --paper-page-footer: none; --paper-page-number: none"></div>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "blank page row must implement PageController")
	control := pc.PageControl()
	assert.True(t, control.Blank)
	assert.True(t, control.SuppressFooter)
	assert.True(t, control.SuppressPageNumber)
	assert.False(t, control.CountPageNumber)
}

func TestTranslate_PaperBlankPageCanCountWithoutStamp(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<div style="--paper-page: blank; --paper-page-footer: none; --paper-page-number: count"></div>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "blank page row must implement PageController")
	control := pc.PageControl()
	assert.True(t, control.Blank)
	assert.True(t, control.SuppressFooter)
	assert.True(t, control.SuppressPageNumber)
	assert.True(t, control.CountPageNumber)
}

func TestTranslate_PaperPageDecorationCustomProperties(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-footer: none; --paper-page-number: none"><p>First unnumbered page</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	assert.False(t, control.Blank)
	assert.True(t, control.SuppressFooter)
	assert.True(t, control.SuppressPageNumber)
}

func TestTranslate_PaperPageDecorationFirstPageScope(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-footer: none; --paper-page-number: none; --paper-page-decor-scope: first-page"><p>First page undecorated only</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	assert.True(t, control.SuppressFooter)
	assert.True(t, control.SuppressPageNumber)
	assert.True(t, control.DecorFirstPageOnly)
}

func TestTranslate_PaperPageTopMarginCustomProperty(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-top-margin: 0mm"><p>Flush page</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	require.NotNil(t, control.TopMargin)
	assert.Equal(t, 0.0, *control.TopMargin)
}

func TestTranslate_PaperPageTopMarginFirstPageScope(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-top-margin: 0mm; --paper-page-top-margin-scope: first-page"><p>Flush first page</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	require.NotNil(t, control.TopMargin)
	assert.Equal(t, 0.0, *control.TopMargin)
	assert.True(t, control.TopMarginFirstPageOnly)
}

func TestTranslate_PaperPageContinuationTopMarginCustomProperty(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-top-margin: 1mm; --paper-page-top-margin-scope: first-page; --paper-page-top-margin-continuation: 0mm"><p>Scoped top margin</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	require.NotNil(t, control.TopMargin)
	assert.Equal(t, 1.0, *control.TopMargin)
	assert.True(t, control.TopMarginFirstPageOnly)
	require.NotNil(t, control.ContinuationTopMargin)
	assert.Equal(t, 0.0, *control.ContinuationTopMargin)
}

func TestTranslate_PaperPageRenderOffsetCustomProperties(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-render-offset-x: 0.5mm; --paper-page-render-offset-y: 1mm; --paper-page-render-offset-scope: first-page; --paper-page-render-offset-x-continuation: 0mm; --paper-page-render-offset-y-continuation: 0mm"><p>Offset render only</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 2)

	pc, ok := rows[0].(core.PageController)
	require.True(t, ok, "first row should control the current page")
	control := pc.PageControl()
	require.NotNil(t, control.RenderOffsetX)
	assert.Equal(t, 0.5, *control.RenderOffsetX)
	require.NotNil(t, control.RenderOffsetY)
	assert.Equal(t, 1.0, *control.RenderOffsetY)
	assert.True(t, control.RenderOffsetFirstPageOnly)
	require.NotNil(t, control.ContinuationRenderOffsetX)
	assert.Equal(t, 0.0, *control.ContinuationRenderOffsetX)
	require.NotNil(t, control.ContinuationRenderOffsetY)
	assert.Equal(t, 0.0, *control.ContinuationRenderOffsetY)
}

func TestTranslate_PageControlPrecedesElementMarginsAfterPageBreak(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="page-break-before: always; margin-top: 2.5mm; --paper-page-top-margin: 0mm"><p>Flush page</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)
	require.True(t, len(rows) >= 3)

	pb, ok := rows[0].(core.PageBreaker)
	require.True(t, ok, "first row should be the requested page break")
	assert.True(t, pb.IsPageBreak())
	_, ok = rows[1].(core.PageController)
	require.True(t, ok, "page control should precede the element's margin spacer")
}

func TestTranslate_PaperPageControlsDoNotRepeatFromInheritedVars(t *testing.T) {
	t.Parallel()

	doc := parseDoc(t, `<html><body>
	<section style="--paper-page-footer: none; --paper-page-number: none"><p>child</p><p>child two</p></section>
	</body></html>`)
	rows, err := translate.Translate(context.Background(), doc)
	require.NoError(t, err)

	controls := 0
	for _, r := range rows {
		if _, ok := r.(core.PageController); ok {
			controls++
		}
	}
	assert.Equal(t, 1, controls)
}
