package paper

import (
	"io"
	"time"

	pdf "github.com/avdoseferovic/paper/internal/pdf"
)

type providerPDF interface {
	AddLink() int
	AddPage()
	AddUTF8FontFromBytes(familyStr, styleStr string, bytes []byte)
	Circle(x, y, r float64, styleStr string)
	GetFillColor() (int, int, int)
	GetMargins() (left, top, right, bottom float64)
	GetXY() (float64, float64)
	Link(x, y, w, h float64, link int)
	PageNo() int
	SetAlpha(alpha float64, blendModeStr string)
	SetFillColor(r, g, b int)
	SetLink(link int, y float64, page int)
	SetXY(x, y float64)
}

type providerDocumentPDF interface {
	Ln(h float64)
	Output(w io.Writer) error
	SetCompression(compress bool)
}

type providerMetadataPDF interface {
	SetAuthor(authorStr string, isUTF8 bool)
	SetCreationDate(tm time.Time)
	SetCreator(creatorStr string, isUTF8 bool)
	SetKeywords(keywordsStr string, isUTF8 bool)
	SetProtection(actionFlag byte, userPassStr, ownerPassStr string)
	SetProtectionAlgorithm(algorithm pdf.ProtectionAlgorithm)
	SetSubject(subjectStr string, isUTF8 bool)
	SetTitle(titleStr string, isUTF8 bool)
}

type providerErrorPDF interface {
	ClearError()
	SetHomeXY()
}

type fontPDF interface {
	SetFont(familyStr, styleStr string, size float64)
	SetFontSize(size float64)
	SetFontStyle(styleStr string)
	SetTextColor(r, g, b int)
}

type textPDF interface {
	ClipEnd()
	ClipRect(x, y, w, h float64, outline bool)
	GetMargins() (left, top, right, bottom float64)
	GetStringWidth(s string) float64
	Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string)
	Link(x, y, w, h float64, link int)
	LinkString(x, y, w, h float64, linkStr string)
	Rect(x, y, w, h float64, styleStr string)
	RegisterImageOptionsReader(imgName string, options pdf.ImageOptions, r io.Reader) *pdf.ImageInfoType
	SetAlpha(alpha float64, blendModeStr string)
	SetFillColor(r, g, b int)
	SetTextColor(r, g, b int)
	Text(x, y float64, txtStr string)
	UnicodeTranslatorFromDescriptor(cpStr string) func(string) string
}

type imagePDF interface {
	ClipEnd()
	ClipRect(x, y, w, h float64, outline bool)
	Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string)
	RegisterImageOptionsReader(imgName string, options pdf.ImageOptions, r io.Reader) *pdf.ImageInfoType
}

type linePDF interface {
	GetMargins() (left, top, right, bottom float64)
	Line(x1, y1, x2, y2 float64)
	SetDashPattern(dashArray []float64, dashPhase float64)
	SetDrawColor(r, g, b int)
	SetLineWidth(width float64)
}

type checkboxPDF interface {
	GetMargins() (left, top, right, bottom float64)
	Line(x1, y1, x2, y2 float64)
	Rect(x, y, w, h float64, styleStr string)
	Text(x, y float64, txtStr string)
}

type gradientPDF interface {
	GetMargins() (left, top, right, bottom float64)
	Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string)
	RegisterImageOptionsReader(imgName string, options pdf.ImageOptions, r io.Reader) *pdf.ImageInfoType
}
