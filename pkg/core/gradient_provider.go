package core

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GradientProvider is an optional capability interface for providers that can
// render CSS gradient backgrounds as rasterised PNG images embedded in the PDF.
//
// Usage:
//
//	if gp, ok := provider.(core.GradientProvider); ok {
//	    gp.DrawGradient(cell, gradient, widthMM, heightMM)
//	}
type GradientProvider interface {
	// DrawGradient rasterises the gradient and paints it behind the cell area.
	// widthMM and heightMM are the cell dimensions in mm.
	DrawGradient(cell *entity.Cell, g *props.Gradient, widthMM, heightMM float64)
}
