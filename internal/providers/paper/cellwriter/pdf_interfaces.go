package cellwriter

import (
	"io"

	pdf "github.com/avdoseferovic/paper/internal/pdf"
)

type cellFormatPDF interface {
	CellFormat(w, h float64, txtStr, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string)
}

type backgroundImagePDF interface {
	ClipEnd()
	ClipRect(x, y, w, h float64, outline bool)
	GetAlpha() (float64, string)
	GetXY() (float64, float64)
	Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string)
	RegisterImageOptionsReader(imgName string, options pdf.ImageOptions, r io.Reader) *pdf.ImageInfoType
	SetAlpha(alpha float64, blendModeStr string)
}

type colorStylerPDF interface {
	SetAlpha(alpha float64, blendModeStr string)
	SetDrawColor(r, g, b int)
	SetFillColor(r, g, b int)
}

type dashPatternPDF interface {
	SetDashPattern(dashArray []float64, dashPhase float64)
}

type lineWidthPDF interface {
	SetLineWidth(width float64)
}

type borderRadiusPDF interface {
	ClosePath()
	CurveBezierCubicTo(cx0, cy0, cx1, cy1, x, y float64)
	DrawPath(styleStr string)
	GetDrawColor() (int, int, int)
	GetFillColor() (int, int, int)
	GetLineWidth() float64
	GetXY() (float64, float64)
	LineTo(x, y float64)
	MoveTo(x, y float64)
	SetAlpha(alpha float64, blendModeStr string)
	SetDrawColor(r, g, b int)
	SetFillColor(r, g, b int)
	SetLineWidth(width float64)
}

type gradientStylerPDF interface {
	GetMargins() (left, top, right, bottom float64)
	GetXY() (float64, float64)
}

type outlinePDF interface {
	GetDrawColor() (int, int, int)
	GetLineWidth() float64
	GetXY() (float64, float64)
	Rect(x, y, w, h float64, styleStr string)
	SetDashPattern(dashArray []float64, dashPhase float64)
	SetDrawColor(r, g, b int)
	SetLineWidth(width float64)
}

type perSideBorderPDF interface {
	GetDrawColor() (int, int, int)
	GetLineWidth() float64
	GetXY() (float64, float64)
	Line(x1, y1, x2, y2 float64)
	SetDashPattern(dashArray []float64, dashPhase float64)
	SetDrawColor(r, g, b int)
	SetLineWidth(width float64)
}

type shadowPDF interface {
	GetXY() (float64, float64)
	Rect(x, y, w, h float64, styleStr string)
	SetAlpha(alpha float64, blendModeStr string)
	SetFillColor(r, g, b int)
	SetXY(x, y float64)
}

func asPDF[T any](pdf any) T {
	typed, _ := pdf.(T)
	return typed
}
