package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/html/dom"
)

func TestMarkedTrailingMarginSurvivesAutomaticPageBreak(t *testing.T) {
	doc, err := dom.Parse(`<html><body><p style="margin-top:0; margin-bottom:20pt" data-preserve-overflow-margin="true">Last content</p></body></html>`)
	require.NoError(t, err)
	rows, err := Translate(t.Context(), doc)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	spacer, ok := rows[1].(marginSpacerRow)
	if !ok {
		t.Fatalf("last row = %T, want margin spacer", rows[1])
	}
	if spacer.DiscardAtAutomaticPageTop() {
		t.Fatal("marked trailing margin was discarded after the page break")
	}
}

func TestCollapsedTrailingMarginKeepsOverflowMarker(t *testing.T) {
	marked := newMarginSpacer(20)
	marked.preserveAtAutomaticPageTop = true
	rows := collapseMarginSpacers([]core.Row{marked, newMarginSpacer(30)})
	require.Len(t, rows, 1)
	spacer, ok := rows[0].(marginSpacerRow)
	if !ok || spacer.height != 30 || spacer.DiscardAtAutomaticPageTop() {
		t.Fatalf("collapsed spacer = %#v, want 30-unit preserved margin", rows[0])
	}
}
