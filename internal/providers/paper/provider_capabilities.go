package paper

import (
	"math"

	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// WithCharSpacing runs fn without adjusting character spacing (no-op).
// See the compile-time assertion in provider.go for context.
func (g *provider) WithCharSpacing(_ float64, fn func()) {
	fn()
}

// RegisterFont makes a TTF/OTF font available for subsequent text rendering.
// Errors from gofpdf's font parser are surfaced lazily by the next text draw.
func (g *provider) RegisterFont(family string, style fontstyle.Type, bytes []byte) {
	g.fpdf.AddUTF8FontFromBytes(family, string(style), bytes)
}

// Bookmark records a PDF outline entry on the current page. y is measured
// from the top of the page content area (entity.Cell convention); the page
// top margin is added so the destination matches where components draw.
func (g *provider) Bookmark(title string, level int, y float64) {
	if level < 0 {
		level = 0
	}
	left, top, _, _ := g.fpdf.GetMargins()
	_ = left // only the top margin matters for the outline destination
	g.fpdf.Bookmark(title, level, y+top)
}

// AddLink reserves a new internal link target ID.
func (g *provider) AddLink() int { return g.fpdf.AddLink() }

// SetLink registers the target's Y position and page number for a link ID.
func (g *provider) SetLink(linkID int, y float64, page int) {
	g.fpdf.SetLink(linkID, y, page)
}

// Link makes a rectangular area clickable, jumping to the named link ID.
func (g *provider) Link(x, y, w, h float64, linkID int) {
	g.fpdf.Link(x, y, w, h, linkID)
}

// SetCursor resets the gofpdf pen position. x and y are margin-relative
// (entity.Cell convention: X=0 means left content edge, not page edge).
// We add the page margins so the resulting absolute position matches where
// all other provider methods (AddText, AddRichText, Image, Line, ...) draw
// for the same cell coordinates.
func (g *provider) SetCursor(x, y float64) {
	left, top, _, _ := g.fpdf.GetMargins()
	g.fpdf.SetXY(x+left, y+top)
}

// EnsurePage advances the physical PDF document until pageNumber is current.
// Paper builds logical pages before rendering, but some HTML render paths draw
// by absolute coordinates and do not reliably trigger gofpdf's cursor-based
// automatic page break between logical pages.
func (g *provider) EnsurePage(pageNumber int) {
	for g.fpdf.PageNo() < pageNumber {
		g.fpdf.AddPage()
	}
}

// WithAlpha runs fn with the gofpdf alpha temporarily set to a (clamped to
// [0, 1], NaN treated as 1). Alpha is always restored to 1.0 via defer so it
// cannot leak into subsequent native rendering, even if fn panics.
func (g *provider) WithAlpha(a float64, fn func()) {
	if math.IsNaN(a) {
		a = 1
	}
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	g.fpdf.SetAlpha(a, "Normal")
	defer g.fpdf.SetAlpha(1, "Normal")
	fn()
}

// DrawFilledCircle draws a filled circle inscribed inside the cell with the
// given fill color (defaulting to black). The circle is centered horizontally
// and vertically with a radius half of the cell's smaller dimension.
func (g *provider) DrawFilledCircle(cell *entity.Cell, fill *props.Color) {
	if cell == nil || cell.Width <= 0 || cell.Height <= 0 {
		return
	}
	color := fill
	if color == nil {
		black := props.Black()
		color = &black
	}
	origR, origG, origB := g.fpdf.GetFillColor()
	defer g.fpdf.SetFillColor(origR, origG, origB)

	g.fpdf.SetFillColor(color.Red, color.Green, color.Blue)
	radius := cell.Width / 2
	if cell.Height/2 < radius {
		radius = cell.Height / 2
	}
	left, top, _, _ := g.fpdf.GetMargins()
	cx := cell.X + cell.Width/2 + left
	cy := cell.Y + cell.Height/2 + top
	g.fpdf.Circle(cx, cy, radius, "F")
}
