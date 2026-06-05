package paper

import (
	"bytes"

	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
	"github.com/avdoseferovic/paper/v2/pkg/props"
)

func (g *provider) CreateRow(height float64) {
	g.fpdf.Ln(height)
}

func (g *provider) GenerateBytes() ([]byte, error) {
	var buffer bytes.Buffer
	err := g.fpdf.Output(&buffer)

	return buffer.Bytes(), err
}

func (g *provider) CreateCol(width, height float64, config *entity.Config, prop *props.Cell) {
	g.cellWriter.Apply(width, height, config, prop)
}

func (g *provider) SetCompression(compression bool) {
	g.fpdf.SetCompression(compression)
}
