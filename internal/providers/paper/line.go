package paper

import (
	"github.com/avdoseferovic/paper/internal/providers/paper/gofpdfwrapper"
	"github.com/avdoseferovic/paper/pkg/consts/linestyle"
	"github.com/avdoseferovic/paper/pkg/consts/orientation"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type Line struct {
	pdf              gofpdfwrapper.Fpdf
	defaultColor     *props.Color
	defaultThickness float64
}

func NewLine(pdf gofpdfwrapper.Fpdf) *Line {
	defaultColor := props.Black()
	return &Line{
		pdf:              pdf,
		defaultColor:     &defaultColor,
		defaultThickness: linestyle.DefaultLineThickness,
	}
}

func (l *Line) Add(cell *entity.Cell, prop *props.Line) {
	if prop.Orientation == orientation.Vertical {
		l.renderVertical(cell, prop)
	} else {
		l.renderHorizontal(cell, prop)
	}
}

func (l *Line) renderVertical(cell *entity.Cell, prop *props.Line) {
	size := cell.Height * (prop.SizePercent / 100.0)
	position := cell.Width * (prop.OffsetPercent / 100.0)

	space := (cell.Height - size) / 2.0

	left, top, _, _ := l.pdf.GetMargins()

	if prop.Color != nil {
		l.pdf.SetDrawColor(prop.Color.Red, prop.Color.Green, prop.Color.Blue)
	}
	l.pdf.SetLineWidth(prop.Thickness)

	setDashPattern(l.pdf, prop.Style)

	l.pdf.Line(left+cell.X+position, top+cell.Y+space, left+cell.X+position, top+cell.Y+cell.Height-space)

	if prop.Color != nil {
		l.pdf.SetDrawColor(l.defaultColor.Red, l.defaultColor.Green, l.defaultColor.Blue)
	}
	l.pdf.SetLineWidth(l.defaultThickness)
	resetDashPattern(l.pdf, prop.Style)
}

func (l *Line) renderHorizontal(cell *entity.Cell, prop *props.Line) {
	size := cell.Width * (prop.SizePercent / 100.0)
	position := cell.Height * (prop.OffsetPercent / 100.0)

	space := (cell.Width - size) / 2.0

	left, top, _, _ := l.pdf.GetMargins()

	if prop.Color != nil {
		l.pdf.SetDrawColor(prop.Color.Red, prop.Color.Green, prop.Color.Blue)
	}
	l.pdf.SetLineWidth(prop.Thickness)

	setDashPattern(l.pdf, prop.Style)

	l.pdf.Line(left+cell.X+space, top+cell.Y+position, left+cell.X+cell.Width-space, top+cell.Y+position)

	if prop.Color != nil {
		l.pdf.SetDrawColor(l.defaultColor.Red, l.defaultColor.Green, l.defaultColor.Blue)
	}
	l.pdf.SetLineWidth(l.defaultThickness)
	resetDashPattern(l.pdf, prop.Style)
}

// setDashPattern applies the gofpdf dash pattern for the given line style.
// Solid is a no-op; Dashed uses [1,1]; Dotted uses [0.4,0.4].
func setDashPattern(pdf gofpdfwrapper.Fpdf, style linestyle.Type) {
	switch style {
	case linestyle.Solid:
		// no dash pattern needed
	case linestyle.Dashed:
		pdf.SetDashPattern([]float64{1, 1}, 0)
	case linestyle.Dotted:
		pdf.SetDashPattern([]float64{0.4, 0.4}, 0)
	}
}

// resetDashPattern restores the solid (no-dash) pattern after a non-solid line.
func resetDashPattern(pdf gofpdfwrapper.Fpdf, style linestyle.Type) {
	if style != linestyle.Solid {
		pdf.SetDashPattern([]float64{1, 0}, 0)
	}
}
