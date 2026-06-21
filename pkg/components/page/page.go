// Package page implements creation of pages.
package page

import (
	"github.com/avdoseferovic/paper/pkg/tree/node"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type Page struct {
	number  int
	total   int
	index   int
	rows    []core.Row
	config  *entity.Config
	prop    props.PageNumber
	control core.PageControl
}

// New is responsible to create a core.Page.
func New(ps ...props.PageNumber) core.Page {
	prop := props.PageNumber{}
	if len(ps) > 0 {
		prop = props.ClonePageNumber(ps[0])
	}

	return &Page{
		prop: prop,
	}
}

// Render renders a Page into a PDF context.
func (p *Page) Render(provider core.Provider, cell entity.Cell) {
	innerCell := p.contentCell(cell)

	if p.config.BackgroundImage != nil {
		backgroundCell := pageImageCell(innerCell, p.config, p.config.BackgroundImage)
		backgroundProp := pageImageProp(p.config.BackgroundImage)
		provider.AddBackgroundImageFromBytes(
			p.config.BackgroundImage.Bytes,
			&backgroundCell,
			backgroundProp,
			p.config.BackgroundImage.Extension,
		)
	}

	if p.isFirstPage() && p.config.FirstPageBackgroundImage != nil {
		backgroundCell := pageImageCell(innerCell, p.config, p.config.FirstPageBackgroundImage)
		backgroundProp := pageImageProp(p.config.FirstPageBackgroundImage)
		provider.AddBackgroundImageFromBytes(
			p.config.FirstPageBackgroundImage.Bytes,
			&backgroundCell,
			backgroundProp,
			p.config.FirstPageBackgroundImage.Extension,
		)
	}

	if p.config.Watermark != nil {
		if wp, ok := provider.(core.WatermarkProvider); ok {
			wp.AddWatermark(&innerCell, p.config.Watermark)
		}
	}

	for _, row := range p.rows {
		row.Render(provider, innerCell)
		innerCell.Y += row.GetHeight(provider, &innerCell)
	}

	if p.isFirstPage() {
		for _, foregroundImage := range firstPageForegroundImages(p.config) {
			foregroundCell := pageImageCell(cell, p.config, foregroundImage)
			foregroundProp := pageImageProp(foregroundImage)
			provider.AddBackgroundImageFromBytes(foregroundImage.Bytes, &foregroundCell, foregroundProp, foregroundImage.Extension)
		}
	}

	if p.prop.Pattern != "" {
		provider.AddText(p.prop.GetPageString(p.number, p.total), &cell, p.prop.GetNumberTextProp(cell.Height))
	}

	if p.isFirstPage() {
		for _, foregroundImage := range p.config.FirstPageFinalForegroundImages {
			foregroundCell := pageImageCell(cell, p.config, foregroundImage)
			foregroundProp := pageImageProp(foregroundImage)
			provider.AddBackgroundImageFromBytes(foregroundImage.Bytes, &foregroundCell, foregroundProp, foregroundImage.Extension)
		}
	}
}

func pageImageProp(image *entity.Image) *props.Rect {
	prop := &props.Rect{}
	prop.MakeValid()
	if image != nil && image.ObjectFit != "" {
		prop.ObjectFit = image.ObjectFit
	}
	return prop
}

func firstPageForegroundImages(cfg *entity.Config) []*entity.Image {
	if cfg == nil {
		return nil
	}
	if len(cfg.FirstPageForegroundImages) != 0 {
		return cfg.FirstPageForegroundImages
	}
	if cfg.FirstPageForegroundImage != nil {
		return []*entity.Image{cfg.FirstPageForegroundImage}
	}
	return nil
}

func pageImageCell(cell entity.Cell, cfg *entity.Config, image *entity.Image) entity.Cell {
	if image == nil || image.PageCell == nil || image.PageCell.Width <= 0 || image.PageCell.Height <= 0 {
		return fullPageBackgroundCell(cell, cfg)
	}

	var marginLeft, marginTop float64
	if cfg != nil && cfg.Margins != nil {
		marginLeft = cfg.Margins.Left
		marginTop = cfg.Margins.Top
	}
	return entity.Cell{
		X:      image.PageCell.X - marginLeft,
		Y:      image.PageCell.Y - marginTop,
		Width:  image.PageCell.Width,
		Height: image.PageCell.Height,
	}
}

func fullPageBackgroundCell(cell entity.Cell, cfg *entity.Config) entity.Cell {
	if cfg == nil || cfg.Dimensions == nil || cfg.Margins == nil ||
		cfg.Dimensions.Width <= 0 || cfg.Dimensions.Height <= 0 {
		return cell
	}
	return entity.Cell{
		X:      -cfg.Margins.Left,
		Y:      -cfg.Margins.Top,
		Width:  cfg.Dimensions.Width,
		Height: cfg.Dimensions.Height,
	}
}

// SetConfig sets the Page configuration.
func (p *Page) SetConfig(config *entity.Config) {
	p.config = config
	for _, row := range p.rows {
		row.SetConfig(config)
	}
}

// SetNumber sets the Page number and total.
func (p *Page) SetNumber(number int, total int) {
	p.number = number
	p.total = total
}

// SetPageIndex records the physical 1-based page index.
func (p *Page) SetPageIndex(index int) {
	p.index = index
}

func (p *Page) SetPageControl(control core.PageControl) {
	p.control = control
}

// GetNumber returns the Page number.
func (p *Page) GetNumber() int {
	return p.number
}

// Add adds one or more rows to the Page.
func (p *Page) Add(rows ...core.Row) core.Page {
	p.rows = append(p.rows, rows...)
	return p
}

// GetRows returns the rows of the Page.
func (p *Page) GetRows() []core.Row {
	return p.rows
}

// GetStructure returns the Structure of a Page.
func (p *Page) GetStructure() *node.Node[core.Structure] {
	str := core.Structure{
		Type: "page",
	}

	n := node.New(str)
	for _, r := range p.rows {
		inner := r.GetStructure()
		n.AddNext(inner)
	}

	return n
}

func (p *Page) contentCell(cell entity.Cell) entity.Cell {
	inner := cell.Copy()
	if p.config != nil && p.config.Margins != nil && p.control.TopMargin != nil {
		delta := p.config.Margins.Top - *p.control.TopMargin
		inner.Y -= delta
		inner.Height += delta
		if inner.Height < 0 {
			inner.Height = 0
		}
	}
	if p.control.RenderOffsetX != nil {
		inner.X += *p.control.RenderOffsetX
	}
	if p.control.RenderOffsetY != nil {
		inner.Y += *p.control.RenderOffsetY
	}
	return inner
}

func (p *Page) isFirstPage() bool {
	return p.index == 1 || (p.index == 0 && p.number == 1)
}
