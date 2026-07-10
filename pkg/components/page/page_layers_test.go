package page_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

type renderLayerTestRow struct {
	name      string
	height    float64
	layer     int
	hasLayer  bool
	renderLog *[]string
}

func (r *renderLayerTestRow) SetConfig(*entity.Config) {}

func (r *renderLayerTestRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: r.name})
}

func (r *renderLayerTestRow) Add(...core.Col) core.Row { return r }

func (r *renderLayerTestRow) GetHeight(core.Provider, *entity.Cell) float64 {
	return r.height
}

func (r *renderLayerTestRow) GetColumns() []core.Col { return nil }

func (r *renderLayerTestRow) WithStyle(*props.Cell) core.Row { return r }

func (r *renderLayerTestRow) Render(_ core.Provider, cell entity.Cell) {
	*r.renderLog = append(*r.renderLog, fmt.Sprintf("%s@%.0f", r.name, cell.Y))
}

func (r *renderLayerTestRow) RenderLayer() int {
	if !r.hasLayer {
		return 0
	}
	return r.layer
}

func TestPage_RenderOrdersLayeredRowsAroundFlow(t *testing.T) {
	t.Parallel()

	var renderLog []string
	sut := page.New()
	sut.Add(
		&renderLayerTestRow{name: "front", layer: 1, hasLayer: true, renderLog: &renderLog},
		&renderLayerTestRow{name: "flow-a", height: 7, renderLog: &renderLog},
		&renderLayerTestRow{name: "behind", layer: -1, hasLayer: true, renderLog: &renderLog},
		&renderLayerTestRow{name: "flow-b", height: 5, renderLog: &renderLog},
	)
	sut.SetConfig(&entity.Config{})

	sut.Render(nil, entity.Cell{Width: 100, Height: 100})

	assert.Equal(t, []string{"behind@7", "flow-a@0", "flow-b@7", "front@0"}, renderLog)
}

type pageContextProviderStub struct {
	current        int
	total          int
	runningStrings map[string]string
}

func (p *pageContextProviderStub) WithPageContext(current, total int, fn func()) {
	p.WithPageContextStrings(current, total, nil, fn)
}

func (p *pageContextProviderStub) WithPageContextStrings(current, total int, runningStrings map[string]string, fn func()) {
	prevCurrent, prevTotal := p.current, p.total
	prevRunningStrings := p.runningStrings
	p.current, p.total = current, total
	p.runningStrings = clonePageContextStrings(runningStrings)
	defer func() {
		p.current, p.total = prevCurrent, prevTotal
		p.runningStrings = prevRunningStrings
	}()
	fn()
}

func clonePageContextStrings(runningStrings map[string]string) map[string]string {
	if len(runningStrings) == 0 {
		return nil
	}
	clone := make(map[string]string, len(runningStrings))
	for name, value := range runningStrings {
		clone[name] = value
	}
	return clone
}

func (p *pageContextProviderStub) CreateRow(float64) {}
func (p *pageContextProviderStub) CreateCol(float64, float64, *entity.Config, *props.Cell) {
}
func (p *pageContextProviderStub) AddLine(*entity.Cell, *props.Line) {}
func (p *pageContextProviderStub) AddText(string, *entity.Cell, *props.Text) {
}
func (p *pageContextProviderStub) AddCheckbox(string, *entity.Cell, *props.Checkbox) {
}
func (p *pageContextProviderStub) GetFontHeight(*props.Font) float64 { return 1 }
func (p *pageContextProviderStub) GetLinesQuantity(string, *props.Text, float64) int {
	return 1
}
func (p *pageContextProviderStub) AddMatrixCode(string, *entity.Cell, *props.Rect) {
}
func (p *pageContextProviderStub) AddQrCode(string, *entity.Cell, *props.Rect) {}
func (p *pageContextProviderStub) AddBarCode(string, *entity.Cell, *props.Barcode) {
}
func (p *pageContextProviderStub) GetDimensionsByMatrixCode(string) (*entity.Dimensions, error) {
	return nil, nil
}
func (p *pageContextProviderStub) GetDimensionsByQrCode(string) (*entity.Dimensions, error) {
	return nil, nil
}
func (p *pageContextProviderStub) GetDimensionsByImageByte([]byte, extension.Type) (*entity.Dimensions, error) {
	return nil, nil
}
func (p *pageContextProviderStub) GetDimensionsByImage(string) (*entity.Dimensions, error) {
	return nil, nil
}
func (p *pageContextProviderStub) AddImageFromFile(string, *entity.Cell, *props.Rect) {
}
func (p *pageContextProviderStub) AddImageFromBytes([]byte, *entity.Cell, *props.Rect, extension.Type) {
}
func (p *pageContextProviderStub) AddBackgroundImageFromBytes([]byte, *entity.Cell, *props.Rect, extension.Type) {
}
func (p *pageContextProviderStub) GenerateBytes() ([]byte, error)   { return nil, nil }
func (p *pageContextProviderStub) SetProtection(*entity.Protection) {}
func (p *pageContextProviderStub) SetCompression(bool)              {}
func (p *pageContextProviderStub) SetMetadata(*entity.Metadata)     {}

type pageContextRecordingRow struct {
	renderLog  *[]string
	stringName string
}

func (r *pageContextRecordingRow) SetConfig(*entity.Config) {}
func (r *pageContextRecordingRow) GetStructure() *node.Node[core.Structure] {
	return node.New(core.Structure{Type: "page-context"})
}
func (r *pageContextRecordingRow) Add(...core.Col) core.Row { return r }
func (r *pageContextRecordingRow) GetHeight(core.Provider, *entity.Cell) float64 {
	return 1
}
func (r *pageContextRecordingRow) GetColumns() []core.Col { return nil }
func (r *pageContextRecordingRow) WithStyle(*props.Cell) core.Row {
	return r
}
func (r *pageContextRecordingRow) Render(provider core.Provider, _ entity.Cell) {
	ctx := provider.(*pageContextProviderStub)
	entry := fmt.Sprintf("%d/%d", ctx.current, ctx.total)
	if r.stringName != "" {
		entry += ":" + ctx.runningStrings[r.stringName]
	}
	*r.renderLog = append(*r.renderLog, entry)
}

func TestPage_RenderProvidesPhysicalPageContext(t *testing.T) {
	t.Parallel()

	var renderLog []string
	sut := page.New()
	sut.Add(&pageContextRecordingRow{renderLog: &renderLog})
	sut.SetConfig(&entity.Config{})
	sut.(interface{ SetPageIndex(int) }).SetPageIndex(2)
	sut.(interface{ SetPageTotal(int) }).SetPageTotal(5)

	provider := &pageContextProviderStub{}
	sut.Render(provider, entity.Cell{Width: 100, Height: 100})

	assert.Equal(t, []string{"2/5"}, renderLog)
	assert.Equal(t, 0, provider.current)
	assert.Equal(t, 0, provider.total)
}

func TestPage_RenderProvidesRunningStringContext(t *testing.T) {
	t.Parallel()

	var renderLog []string
	sut := page.New()
	sut.Add(&pageContextRecordingRow{renderLog: &renderLog, stringName: "chapter"})
	sut.SetConfig(&entity.Config{})
	sut.(interface{ SetPageIndex(int) }).SetPageIndex(3)
	sut.(interface{ SetPageTotal(int) }).SetPageTotal(7)
	sut.(interface{ SetRunningStrings(map[string]string) }).SetRunningStrings(map[string]string{"chapter": "One"})

	provider := &pageContextProviderStub{}
	sut.Render(provider, entity.Cell{Width: 100, Height: 100})

	assert.Equal(t, []string{"3/7:One"}, renderLog)
	assert.Equal(t, 0, provider.current)
	assert.Equal(t, 0, provider.total)
	assert.Nil(t, provider.runningStrings)
}
