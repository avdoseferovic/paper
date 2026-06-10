package html_test

import (
	"bytes"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	htmlcomponent "github.com/avdoseferovic/paper/pkg/components/html"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNew(t *testing.T) {
	t.Parallel()

	component, err := htmlcomponent.New(`<h1>Heading</h1><p>Hello <b>HTML</b></p>`)

	require.NoError(t, err)
	require.NotNil(t, component)

	tree := component.GetStructure()
	require.Equal(t, "html", tree.GetData().Type)
	require.Equal(t, `<h1>Heading</h1><p>Hello <b>HTML</b></p>`, tree.GetData().Value)
	require.Equal(t, 2, tree.GetData().Details["rows"])
	require.Len(t, tree.GetNexts(), 2)
}

func TestNewCol(t *testing.T) {
	t.Parallel()

	c, err := htmlcomponent.NewCol(6, `<p>Column HTML</p>`)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, 6, c.GetStructure().GetData().Value)
}

func TestNewRow(t *testing.T) {
	t.Parallel()

	r, err := htmlcomponent.NewRow(12, `<p>Fixed row HTML</p>`)

	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 12.0, r.GetStructure().GetData().Value)
}

func TestNewAutoRow(t *testing.T) {
	t.Parallel()

	r, err := htmlcomponent.NewAutoRow(`<p>Auto row HTML</p>`)

	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, 0.0, r.GetStructure().GetData().Value)
}

func TestHTMLComponentMixedWithDirectComponentsGeneratesPDF(t *testing.T) {
	t.Parallel()

	component, err := htmlcomponent.New(`
<style>
  .box { padding: 2mm; border: 1px solid #336699; background: #eef5ff; }
</style>
<div class="box">
  <h2>HTML fragment</h2>
  <p>Rendered beside a direct Paper component.</p>
</div>`)
	require.NoError(t, err)

	m := paper.New()
	m.AddRows(
		row.New().Add(
			col.New(5).Add(text.New("Direct component")),
			col.New(7).Add(component),
		),
	)

	doc, err := m.Generate()
	require.NoError(t, err)

	pdfBytes := doc.GetBytes()
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")))
	assert.Greater(t, len(pdfBytes), 1000)
}

func TestHTMLComponentRenderRestoresCursorToCellOrigin(t *testing.T) {
	t.Parallel()

	component, err := htmlcomponent.New(`<p>One</p><p>Two</p>`)
	require.NoError(t, err)
	component.SetConfig(config.NewBuilder().Build())

	provider := &cursorProvider{}
	cell := &entity.Cell{X: 10, Y: 20, Width: 50, Height: 100}

	component.Render(provider, cell)

	require.NotEmpty(t, provider.cursors)
	last := provider.cursors[len(provider.cursors)-1]
	assert.InDelta(t, 10.0, last.x, 0.001)
	assert.InDelta(t, 20.0, last.y, 0.001)
}

var _ core.Component = (*htmlcomponent.HTML)(nil)

type cursorPoint struct {
	x float64
	y float64
}

type cursorProvider struct {
	cursors []cursorPoint
}

func (p *cursorProvider) CreateRow(_ float64) {}

func (p *cursorProvider) CreateCol(_ float64, _ float64, _ *entity.Config, _ *props.Cell) {}

func (p *cursorProvider) AddLine(_ *entity.Cell, _ *props.Line) {}

func (p *cursorProvider) AddText(_ string, _ *entity.Cell, _ *props.Text) {}

func (p *cursorProvider) AddCheckbox(_ string, _ *entity.Cell, _ *props.Checkbox) {}

func (p *cursorProvider) GetFontHeight(_ *props.Font) float64 { return 5 }

func (p *cursorProvider) GetLinesQuantity(_ string, _ *props.Text, _ float64) int { return 1 }

func (p *cursorProvider) AddMatrixCode(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddQrCode(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddBarCode(_ string, _ *entity.Cell, _ *props.Barcode) {}

func (p *cursorProvider) GetDimensionsByMatrixCode(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByQrCode(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByImageByte(_ []byte, _ extension.Type) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) GetDimensionsByImage(_ string) (*entity.Dimensions, error) {
	return nil, nil
}

func (p *cursorProvider) AddImageFromFile(_ string, _ *entity.Cell, _ *props.Rect) {}

func (p *cursorProvider) AddImageFromBytes(_ []byte, _ *entity.Cell, _ *props.Rect, _ extension.Type) {
}

func (p *cursorProvider) AddBackgroundImageFromBytes(_ []byte, _ *entity.Cell, _ *props.Rect, _ extension.Type) {
}

func (p *cursorProvider) GenerateBytes() ([]byte, error) { return nil, nil }

func (p *cursorProvider) SetProtection(_ *entity.Protection) {}

func (p *cursorProvider) SetCompression(_ bool) {}

func (p *cursorProvider) SetMetadata(_ *entity.Metadata) {}

func (p *cursorProvider) SetCursor(x, y float64) {
	p.cursors = append(p.cursors, cursorPoint{x: x, y: y})
}
