package paper_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// buildRealTextProvider builds a provider with the real builder (real gofpdf,
// real cp1252 translator) so glyph fallbacks behave exactly as in production.
func buildRealTextProvider(t *testing.T) core.Provider {
	t.Helper()
	fontProp := props.Font{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 10}
	cfg := &entity.Config{
		Dimensions:  &entity.Dimensions{Width: 210, Height: 297},
		Margins:     &entity.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10},
		DefaultFont: &fontProp,
	}
	return gofpdf.New(gofpdf.NewBuilder().Build(cfg, nil))
}

func TestProvider_AddTextReportsUnsupportedGlyphs(t *testing.T) {
	t.Parallel()

	t.Run("when text contains glyphs outside cp1252, should record a render issue", func(t *testing.T) {
		t.Parallel()
		sut := buildRealTextProvider(t)
		prop := &props.Text{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 10, Align: consts.AlignLeft}
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 20}

		sut.AddText("Zażółć", cell, prop)

		issues := sut.(core.RenderIssueProvider).RenderIssues()
		require.Len(t, issues, 1)
		assert.Equal(t, "text.unsupported-glyphs", issues[0].Operation)
		// cp1252 has "ó" but not "ż", "ł", "ć"
		assert.Contains(t, issues[0].Message, "ż")
		assert.Contains(t, issues[0].Message, "ł")
		assert.Contains(t, issues[0].Message, "ć")
	})

	t.Run("when text is fully representable in cp1252, should record nothing", func(t *testing.T) {
		t.Parallel()
		sut := buildRealTextProvider(t)
		prop := &props.Text{Family: consts.FontFamilyArial, Style: fontstyle.Normal, Size: 10, Align: consts.AlignLeft}
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 20}

		sut.AddText("café", cell, prop)

		assert.Len(t, sut.(core.RenderIssueProvider).RenderIssues(), 0)
	})

	t.Run("when rich text contains glyphs outside cp1252, should record one deduped issue per add", func(t *testing.T) {
		t.Parallel()
		sut := buildRealTextProvider(t)
		cell := &entity.Cell{X: 0, Y: 0, Width: 100, Height: 20}
		runs := []props.RichRun{{Text: "Zażółć zażółć"}}

		sut.(core.RichTextProvider).AddRichText(runs, cell, &props.RichText{})

		issues := sut.(core.RenderIssueProvider).RenderIssues()
		require.Len(t, issues, 1)
		assert.Equal(t, "text.unsupported-glyphs", issues[0].Operation)
	})
}
