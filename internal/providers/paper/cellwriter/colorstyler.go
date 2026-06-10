package cellwriter

import (
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

type colorSetter func(pdf colorStylerPDF, color *props.Color)

func applyColorStyler(
	template *stylerTemplate,
	color *props.Color,
	defaultColor *props.Color,
	setColor colorSetter,
	width float64,
	height float64,
	config *entity.Config,
	prop *props.Cell,
) {
	if color == nil {
		template.GoToNext(width, height, config, prop)
		return
	}

	fpdf := asPDF[colorStylerPDF](template.fpdf)
	setColor(fpdf, color)
	if a := color.Alpha; a != nil && *a < 1 {
		fpdf.SetAlpha(clampAlpha(*a), "Normal")
		defer fpdf.SetAlpha(1, "Normal")
	}
	template.GoToNext(width, height, config, prop)
	setColor(fpdf, defaultColor)
}

func setDrawColor(pdf colorStylerPDF, color *props.Color) {
	pdf.SetDrawColor(color.Red, color.Green, color.Blue)
}

func setFillColor(pdf colorStylerPDF, color *props.Color) {
	pdf.SetFillColor(color.Red, color.Green, color.Blue)
}
