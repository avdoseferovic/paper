package translate

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/html/dom"
	"github.com/avdoseferovic/paper/pkg/props"
)

// parseInternalDoc parses HTML for internal-package tests (the external test
// package has its own parseDoc helper).
func parseInternalDoc(t *testing.T, src string) *dom.Document {
	t.Helper()
	doc, err := dom.Parse(src)
	require.NoError(t, err)
	return doc
}

// setBlockTagTestConfig applies a default render config to all rows.
func setBlockTagTestConfig(rows []core.Row) {
	cfg := &entity.Config{
		MaxGridSize: 12,
		DefaultFont: &props.Font{
			Family: "Helvetica",
			Style:  fontstyle.Normal,
			Size:   10,
		},
	}
	for _, r := range rows {
		r.SetConfig(cfg)
	}
}

type blockTagTextCall struct {
	text string
	prop *props.Text
}

// blockTagRecordingProvider records AddText calls so tests can assert on the
// rendered text and its resolved props.
type blockTagRecordingProvider struct {
	cursorProvider
	texts []blockTagTextCall
}

func (p *blockTagRecordingProvider) AddText(text string, _ *entity.Cell, prop *props.Text) {
	p.texts = append(p.texts, blockTagTextCall{text: text, prop: prop})
}
