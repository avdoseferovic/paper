package paper

import (
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"strings"

	"github.com/johnfercher/paper/v2/pkg/consts/barcode"

	"github.com/johnfercher/paper/v2/internal/cache"
	"github.com/johnfercher/paper/v2/internal/merror"
	"github.com/johnfercher/paper/v2/internal/providers/paper/cellwriter"
	"github.com/johnfercher/paper/v2/internal/providers/paper/gofpdfwrapper"
	"github.com/johnfercher/paper/v2/pkg/consts/extension"
	"github.com/johnfercher/paper/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/paper/v2/pkg/core"
	"github.com/johnfercher/paper/v2/pkg/core/entity"
	"github.com/johnfercher/paper/v2/pkg/props"
)

var ErrCannotReadImageOptions = errors.New("could not read image options, maybe path/name is wrong")

// compile-time assertion: *provider satisfies core.RichTextProvider.
var _ core.RichTextProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.ShapeProvider.
var _ core.ShapeProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.PositionProvider.
var _ core.PositionProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.PageProvider.
var _ core.PageProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.AlphaProvider.
var _ core.AlphaProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.LinkProvider.
var _ core.LinkProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.LateFontProvider.
var _ core.LateFontProvider = (*provider)(nil)

// compile-time assertion: *provider satisfies core.CharSpacingProvider.
// Note: the underlying phpdave11/gofpdf fork does not expose SetCharSpacing,
// so WithCharSpacing is currently a no-op (documented limitation in
// docs/v2/html-support.md). The capability interface is in place so a future
// fork swap can light up the feature without further wiring changes.
var _ core.CharSpacingProvider = (*provider)(nil)

// WithCharSpacing runs fn without adjusting character spacing (no-op).
// See the compile-time assertion above for context.
func (g *provider) WithCharSpacing(_ float64, fn func()) {
	fn()
}

// RegisterFont makes a TTF/OTF font available for subsequent text rendering.
// Errors from gofpdf's font parser are surfaced lazily by the next text draw.
func (g *provider) RegisterFont(family string, style fontstyle.Type, bytes []byte) {
	g.fpdf.AddUTF8FontFromBytes(family, string(style), bytes)
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
// all other provider methods (AddText, AddRichText, Image, Line, …) draw
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

type provider struct {
	fpdf       gofpdfwrapper.Fpdf
	font       core.Font
	text       core.Text
	richText   *Text // typed pointer for RichTextProvider; nil-safe when text is a mock
	code       core.Code
	image      core.Image
	line       core.Line
	checkbox   core.Checkbox
	cache      cache.Cache
	cellWriter cellwriter.CellWriter
	cfg        *entity.Config
}

// New is the constructor of provider for gofpdf
func New(dep *Dependencies) core.Provider {
	richText, _ := dep.Text.(*Text)
	return &provider{
		fpdf:       dep.Fpdf,
		font:       dep.Font,
		text:       dep.Text,
		richText:   richText,
		code:       dep.Code,
		image:      dep.Image,
		line:       dep.Line,
		checkbox:   dep.Checkbox,
		cellWriter: dep.CellWriter,
		cfg:        dep.Cfg,
		cache:      dep.Cache,
	}
}

func (g *provider) MeasureString(text string, prop *props.Text) float64 {
	if g.richText == nil {
		return 0
	}
	return g.richText.MeasureString(text, prop)
}

func (g *provider) AddTextAt(x, y float64, text string, prop *props.Text) {
	if g.richText == nil {
		return
	}
	g.richText.AddTextAt(x, y, text, prop)
}

func (g *provider) AddRichText(runs []props.RichRun, cell *entity.Cell, prop *props.RichText) {
	if g.richText == nil {
		return
	}
	g.richText.AddRichText(runs, cell, prop)
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
		color = &props.BlackColor
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

func (g *provider) AddText(text string, cell *entity.Cell, prop *props.Text) {
	g.text.Add(text, cell, prop)
}

func (g *provider) GetLinesQuantity(text string, textProp *props.Text, colWidth float64) int {
	return g.text.GetLinesQuantity(text, textProp, colWidth)
}

func (g *provider) GetFontHeight(prop *props.Font) float64 {
	return g.font.GetHeight(prop.Family, prop.Style, prop.Size)
}

func (g *provider) AddLine(cell *entity.Cell, prop *props.Line) {
	g.line.Add(cell, prop)
}

func (g *provider) AddCheckbox(label string, cell *entity.Cell, prop *props.Checkbox) {
	g.checkbox.Add(label, cell, prop)
}

func (g *provider) AddMatrixCode(code string, cell *entity.Cell, prop *props.Rect) {
	img, err := g.loadCode(code, "matrix-code-", g.code.GenDataMatrix)
	if err != nil {
		g.text.Add("could not generate matrixcode", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension.Png, false)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add matrixcode to document", cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddQrCode(code string, cell *entity.Cell, prop *props.Rect) {
	img, err := g.loadCode(code, "qr-code-", g.code.GenQr)
	if err != nil {
		g.text.Add("could not generate qrcode", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension.Png, false)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add qrcode to document", cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddBarCode(code string, cell *entity.Cell, prop *props.Barcode) {
	image, err := g.cache.GetImage(g.getBarcodeImageName("bar-code-"+code, prop), extension.Png)
	if err != nil {
		image, err = g.code.GenBar(code, cell, prop)
	}
	if err != nil {
		g.text.Add("could not generate barcode", cell, merror.DefaultErrorText)
		return
	}

	g.cache.AddImage(g.getBarcodeImageName("bar-code-"+code, prop), image)
	err = g.image.Add(image, cell, g.cfg.Margins, prop.ToRectProp(), extension.Png, false)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add barcode to document", cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddImageFromFile(file string, cell *entity.Cell, prop *props.Rect) {
	extensionStr := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
	image, err := g.loadImage(file, extensionStr)
	if err != nil {
		g.text.Add("could not load image", cell, merror.DefaultErrorText)
		return
	}

	g.AddImageFromBytes(image.Bytes, cell, prop, extension.Type(extensionStr))
}

func (g *provider) AddImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		g.text.Add("could not parse image bytes", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension, false)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add image to document", cell, merror.DefaultErrorText)
	}
}

func (g *provider) AddBackgroundImageFromBytes(bytes []byte, cell *entity.Cell, prop *props.Rect, extension extension.Type) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		g.text.Add("could not parse image bytes", cell, merror.DefaultErrorText)
		return
	}

	err = g.image.Add(img, cell, g.cfg.Margins, prop, extension, true)
	if err != nil {
		g.fpdf.ClearError()
		g.text.Add("could not add image to document", cell, merror.DefaultErrorText)
	}
	g.fpdf.SetHomeXY()
}

func (g *provider) CreateRow(height float64) {
	g.fpdf.Ln(height)
}

func (g *provider) SetProtection(protection *entity.Protection) {
	if protection == nil {
		return
	}

	g.fpdf.SetProtection(byte(protection.Type), protection.UserPassword, protection.OwnerPassword)
}

func (g *provider) SetMetadata(metadata *entity.Metadata) {
	if metadata == nil {
		return
	}

	if metadata.Author != nil {
		g.fpdf.SetAuthor(metadata.Author.Text, metadata.Author.UTF8)
	}

	if metadata.Creator != nil {
		g.fpdf.SetCreator(metadata.Creator.Text, metadata.Creator.UTF8)
	}

	if metadata.Subject != nil {
		g.fpdf.SetSubject(metadata.Subject.Text, metadata.Subject.UTF8)
	}

	if metadata.Title != nil {
		g.fpdf.SetTitle(metadata.Title.Text, metadata.Title.UTF8)
	}

	if metadata.CreationDate != nil {
		g.fpdf.SetCreationDate(*metadata.CreationDate)
	}

	if metadata.KeywordsStr != nil {
		g.fpdf.SetKeywords(metadata.KeywordsStr.Text, metadata.KeywordsStr.UTF8)
	}
}

// GetDimensionsByImage is responsible for obtaining the dimensions of an image
// If the image cannot be loaded, an error is returned
func (g *provider) GetDimensionsByImage(file string) (*entity.Dimensions, error) {
	extensionStr := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
	img, err := g.loadImage(file, extensionStr)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Type(extensionStr))

	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// GetDimensionsByImageByte is responsible for obtaining the dimensions of an image
// If the image cannot be loaded, an error is returned
func (g *provider) GetDimensionsByImageByte(bytes []byte, extension extension.Type) (*entity.Dimensions, error) {
	img, err := FromBytes(bytes, extension)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension)
	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// GetDimensionsByMatrixCode is responsible for obtaining the dimensions of an MatrixCode
// If the image cannot be loaded, an error is returned
func (g *provider) GetDimensionsByMatrixCode(code string) (*entity.Dimensions, error) {
	img, err := g.loadCode(code, "matrix-code-", g.code.GenDataMatrix)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Png)

	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
}

// GetDimensionsByQrCode is responsible for obtaining the dimensions of an QrCode
// If the image cannot be loaded, an error is returned
func (g *provider) GetDimensionsByQrCode(code string) (*entity.Dimensions, error) {
	img, err := g.loadCode(code, "qr-code-", g.code.GenQr)
	if err != nil {
		return nil, err
	}

	dimensions, _ := g.image.GetImageDimensions(img, extension.Png)
	if dimensions == nil {
		return nil, ErrCannotReadImageOptions
	}
	return dimensions, nil
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

func (g *provider) getBarcodeImageName(code string, prop *props.Barcode) string {
	if prop == nil {
		return code + string(barcode.Code128)
	}

	return code + string(prop.Type)
}

// loadImage is responsible for loading an codes
func (g *provider) loadCode(code, codeType string, generate func(code string) (*entity.Image, error)) (*entity.Image, error) {
	image, err := g.cache.GetImage(codeType+code, extension.Png)
	if err != nil {
		image, err = generate(code)
	} else {
		return image, nil
	}
	if err != nil {
		return nil, err
	}
	g.cache.AddImage(codeType+code, image)

	return image, nil
}

// loadImage is responsible for loading an image
func (g *provider) loadImage(file, extensionStr string) (*entity.Image, error) {
	image, err := g.cache.GetImage(file, extension.Type(extensionStr))

	if err == nil {
		return image, nil
	}

	err = g.cache.LoadImage(file, extension.Type(extensionStr))
	if err != nil {
		return nil, err
	}

	return g.cache.GetImage(file, extension.Type(extensionStr))
}
