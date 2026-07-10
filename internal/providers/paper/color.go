package paper

import "github.com/avdoseferovic/paper/pkg/props"

type drawColorSetter interface {
	SetDrawColor(r, g, b int)
}

type fillColorSetter interface {
	SetFillColor(r, g, b int)
}

type textColorSetter interface {
	SetTextColor(r, g, b int)
}

type drawCMYKColorSetter interface {
	SetDrawCMYKColor(c, m, y, k float64)
}

type fillCMYKColorSetter interface {
	SetFillCMYKColor(c, m, y, k float64)
}

type textCMYKColorSetter interface {
	SetTextCMYKColor(c, m, y, k float64)
}

func setPDFDrawColor(pdf drawColorSetter, color *props.Color) {
	if color == nil {
		return
	}
	if color.CMYK != nil {
		if cmykPDF, ok := any(pdf).(drawCMYKColorSetter); ok {
			cmyk := color.CMYK
			cmykPDF.SetDrawCMYKColor(cmyk.Cyan, cmyk.Magenta, cmyk.Yellow, cmyk.Key)
			return
		}
	}
	pdf.SetDrawColor(color.Red, color.Green, color.Blue)
}

func setPDFFillColor(pdf fillColorSetter, color *props.Color) {
	if color == nil {
		return
	}
	if color.CMYK != nil {
		if cmykPDF, ok := any(pdf).(fillCMYKColorSetter); ok {
			cmyk := color.CMYK
			cmykPDF.SetFillCMYKColor(cmyk.Cyan, cmyk.Magenta, cmyk.Yellow, cmyk.Key)
			return
		}
	}
	pdf.SetFillColor(color.Red, color.Green, color.Blue)
}

func setPDFTextColor(pdf textColorSetter, color *props.Color) {
	if color == nil {
		return
	}
	if color.CMYK != nil {
		if cmykPDF, ok := any(pdf).(textCMYKColorSetter); ok {
			cmyk := color.CMYK
			cmykPDF.SetTextCMYKColor(cmyk.Cyan, cmyk.Magenta, cmyk.Yellow, cmyk.Key)
			return
		}
	}
	pdf.SetTextColor(color.Red, color.Green, color.Blue)
}
