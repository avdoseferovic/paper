package config

import (
	"github.com/avdoseferovic/paper/v2/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/v2/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
)

// WithPageSize defines the page size, ex: A4, A4 and etc.
func (b *CfgBuilder) WithPageSize(size pagesize.Type) Builder {
	if size == "" {
		return b
	}

	b.pageSize = &size
	return b
}

// WithDimensions defines custom page dimensions, this overrides page size.
func (b *CfgBuilder) WithDimensions(width float64, height float64) Builder {
	if width <= 0 || height <= 0 {
		return b
	}

	b.dimensions = &entity.Dimensions{
		Width:  width,
		Height: height,
	}

	return b
}

// WithLeftMargin customize margin.
func (b *CfgBuilder) WithLeftMargin(left float64) Builder {
	if left < pagesize.MinLeftMargin {
		return b
	}

	b.margins.Left = left
	return b
}

// WithTopMargin customize margin.
func (b *CfgBuilder) WithTopMargin(top float64) Builder {
	if top < pagesize.MinTopMargin {
		return b
	}

	b.margins.Top = top
	return b
}

// WithRightMargin customize margin.
func (b *CfgBuilder) WithRightMargin(right float64) Builder {
	if right < pagesize.MinRightMargin {
		return b
	}

	b.margins.Right = right
	return b
}

// WithBottomMargin customize margin.
func (b *CfgBuilder) WithBottomMargin(bottom float64) Builder {
	if bottom < pagesize.MinBottomMargin {
		return b
	}

	b.margins.Bottom = bottom
	return b
}

func (b *CfgBuilder) getDimensions() *entity.Dimensions {
	if b.dimensions != nil {
		return b.dimensions
	}

	pageSize := pagesize.A4
	if b.pageSize != nil {
		pageSize = *b.pageSize
	}

	width, height := pagesize.GetDimensions(pageSize)
	dimensions := &entity.Dimensions{
		Width:  width,
		Height: height,
	}

	if b.orientation == orientation.Horizontal && height > width {
		dimensions.Width, dimensions.Height = dimensions.Height, dimensions.Width
	}

	return dimensions
}
