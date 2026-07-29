package config

import (
	"cmp"
	"slices"

	"github.com/avdoseferovic/paper/internal/htmllimits"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// WithProtection defines protection types to the PDF document.
//
// Protection uses the PDF standard security handler. RC4 is the compatibility
// default; use WithProtectionAlgorithm to select AES-128. Protection deters
// casual copying or printing, but is not confidentiality-grade encryption. For
// confidentiality, encrypt the file at rest or in transit.
func (b *CfgBuilder) WithProtection(protectionType protection.Type, userPassword, ownerPassword string) Builder {
	b.protection = &entity.Protection{
		Type:          protectionType,
		UserPassword:  userPassword,
		OwnerPassword: ownerPassword,
		Algorithm:     b.protectionAlgorithm,
	}

	return b
}

// WithProtectionAlgorithm selects the protection encryption algorithm.
//
// The default remains the legacy RC4 security handler for compatibility.
// AES128 selects AESV2, PDF standard security handler revision 4.
func (b *CfgBuilder) WithProtectionAlgorithm(algorithm protection.Encryption) Builder {
	b.protectionAlgorithm = algorithm
	if b.protection != nil {
		b.protection.Algorithm = algorithm
	}

	return b
}

// WithCompression defines compression.
func (b *CfgBuilder) WithCompression(compression bool) Builder {
	b.compression = compression
	return b
}

// WithOrientation defines the page orientation. The default orientation is vertical,
// if horizontal is defined width and height will be flipped.
func (b *CfgBuilder) WithOrientation(pageOrientation consts.Orientation) Builder {
	b.orientation = pageOrientation
	return b
}

// WithCustomFonts add custom fonts.
func (b *CfgBuilder) WithCustomFonts(customFonts []entity.CustomFont) Builder {
	b.customFonts = slices.Clone(customFonts)
	return b
}

// WithBackgroundImage defines the background image that will be applied in every page.
func (b *CfgBuilder) WithBackgroundImage(bytes []byte, ext extension.Type) Builder {
	b.backgroundImage = newConfigImage(bytes, ext)

	return b
}

// WithBackgroundImageAt defines a background image placed in physical page coordinates on every page.
func (b *CfgBuilder) WithBackgroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.backgroundImage = newConfigPageImage(bytes, ext, x, y, width, height, "")

	return b
}

// WithBackgroundImageAtFit defines a background image placed in physical page
// coordinates on every page with object-fit behavior.
func (b *CfgBuilder) WithBackgroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.backgroundImage = newConfigPageImage(bytes, ext, x, y, width, height, objectFit)

	return b
}

// WithFirstPageBackgroundImage defines a background image that will be applied only on the first page.
func (b *CfgBuilder) WithFirstPageBackgroundImage(bytes []byte, ext extension.Type) Builder {
	b.firstPageBackgroundImage = newConfigImage(bytes, ext)

	return b
}

// WithFirstPageBackgroundImageAt defines a background image placed in physical page coordinates only on the first page.
func (b *CfgBuilder) WithFirstPageBackgroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.firstPageBackgroundImage = newConfigPageImage(bytes, ext, x, y, width, height, "")

	return b
}

// WithFirstPageBackgroundImageAtFit defines a background image placed in
// physical page coordinates only on the first page with object-fit behavior.
func (b *CfgBuilder) WithFirstPageBackgroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.firstPageBackgroundImage = newConfigPageImage(bytes, ext, x, y, width, height, objectFit)

	return b
}

// WithFirstPageForegroundImage defines a foreground image applied only on the
// first page after page rows are rendered.
func (b *CfgBuilder) WithFirstPageForegroundImage(bytes []byte, ext extension.Type) Builder {
	b.setFirstPageForegroundImage(newConfigImage(bytes, ext))

	return b
}

// WithFirstPageForegroundImageAt defines a foreground image placed in physical
// page coordinates only on the first page after page rows are rendered.
func (b *CfgBuilder) WithFirstPageForegroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.setFirstPageForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, ""))

	return b
}

// WithFirstPageForegroundImageAtFit defines a foreground image placed in
// physical page coordinates only on the first page after page rows are rendered
// with object-fit behavior.
func (b *CfgBuilder) WithFirstPageForegroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.setFirstPageForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, objectFit))

	return b
}

// WithAdditionalFirstPageForegroundImage adds a foreground image applied only
// on the first page after page rows are rendered.
func (b *CfgBuilder) WithAdditionalFirstPageForegroundImage(bytes []byte, ext extension.Type) Builder {
	b.addFirstPageForegroundImage(newConfigImage(bytes, ext))

	return b
}

// WithAdditionalFirstPageForegroundImageAt adds a foreground image placed in
// physical page coordinates only on the first page after page rows are rendered.
func (b *CfgBuilder) WithAdditionalFirstPageForegroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.addFirstPageForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, ""))

	return b
}

// WithAdditionalFirstPageForegroundImageAtFit adds a foreground image placed in
// physical page coordinates only on the first page after page rows are rendered
// with object-fit behavior.
func (b *CfgBuilder) WithAdditionalFirstPageForegroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.addFirstPageForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, objectFit))

	return b
}

func (b *CfgBuilder) setFirstPageForegroundImage(image *entity.Image) {
	b.firstPageForegroundImage = image
	b.firstPageForegroundImages = []*entity.Image{image}
}

func (b *CfgBuilder) addFirstPageForegroundImage(image *entity.Image) {
	b.firstPageForegroundImage = cmp.Or(b.firstPageForegroundImage, image)
	b.firstPageForegroundImages = append(b.firstPageForegroundImages, image)
}

// WithFirstPageFinalForegroundImage defines a foreground image applied only on
// the first page after page rows and page numbers are rendered.
func (b *CfgBuilder) WithFirstPageFinalForegroundImage(bytes []byte, ext extension.Type) Builder {
	b.setFirstPageFinalForegroundImage(newConfigImage(bytes, ext))

	return b
}

// WithFirstPageFinalForegroundImageAt defines a foreground image placed in
// physical page coordinates only on the first page after rows and page numbers.
func (b *CfgBuilder) WithFirstPageFinalForegroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.setFirstPageFinalForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, ""))

	return b
}

// WithFirstPageFinalForegroundImageAtFit defines a foreground image placed in
// physical page coordinates only on the first page after rows and page numbers
// with object-fit behavior.
func (b *CfgBuilder) WithFirstPageFinalForegroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.setFirstPageFinalForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, objectFit))

	return b
}

// WithAdditionalFirstPageFinalForegroundImage adds a foreground image applied
// only on the first page after page rows and page numbers are rendered.
func (b *CfgBuilder) WithAdditionalFirstPageFinalForegroundImage(bytes []byte, ext extension.Type) Builder {
	b.addFirstPageFinalForegroundImage(newConfigImage(bytes, ext))

	return b
}

// WithAdditionalFirstPageFinalForegroundImageAt adds a foreground image placed
// in physical page coordinates only on the first page after rows and numbers.
func (b *CfgBuilder) WithAdditionalFirstPageFinalForegroundImageAt(bytes []byte, ext extension.Type, x, y, width, height float64) Builder {
	b.addFirstPageFinalForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, ""))

	return b
}

// WithAdditionalFirstPageFinalForegroundImageAtFit adds a foreground image
// placed in physical page coordinates only on the first page after rows and
// numbers with object-fit behavior.
func (b *CfgBuilder) WithAdditionalFirstPageFinalForegroundImageAtFit(
	bytes []byte,
	ext extension.Type,
	x, y, width, height float64,
	objectFit string,
) Builder {
	b.addFirstPageFinalForegroundImage(newConfigPageImage(bytes, ext, x, y, width, height, objectFit))

	return b
}

func (b *CfgBuilder) setFirstPageFinalForegroundImage(image *entity.Image) {
	b.firstPageFinalForegroundImages = []*entity.Image{image}
}

func (b *CfgBuilder) addFirstPageFinalForegroundImage(image *entity.Image) {
	b.firstPageFinalForegroundImages = append(b.firstPageFinalForegroundImages, image)
}

func newConfigImage(bytes []byte, ext extension.Type) *entity.Image {
	return &entity.Image{
		Bytes:     slices.Clone(bytes),
		Extension: ext,
	}
}

func newConfigPageImage(bytes []byte, ext extension.Type, x, y, width, height float64, objectFit string) *entity.Image {
	image := newConfigImage(bytes, ext)
	image.PageCell = &entity.Cell{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
	image.ObjectFit = objectFit
	return image
}

// WithDisableAutoPageBreak defines the option to disable automatic page breaks.
func (b *CfgBuilder) WithDisableAutoPageBreak(disabled bool) Builder {
	b.disableAutoPageBreak = disabled
	return b
}

// WithOutlineFromHeadings makes HTML conversion (AddHTML/FromHTML) add h1-h6
// headings to the PDF document outline: h1 becomes a level-0 entry, h2 a
// level-1 entry, and so on. Hidden headings are skipped. Default: off.
func (b *CfgBuilder) WithOutlineFromHeadings(enabled bool) Builder {
	b.outlineFromHeadings = enabled
	return b
}

// WithWatermark stamps every page with translucent diagonal text, drawn under
// the page content. Optional props customize font, size, color, opacity, and
// angle; zero values fall back to defaults (48pt, alpha 0.12, 45 degrees).
// An empty text removes the watermark.
func (b *CfgBuilder) WithWatermark(text string, ps ...props.Watermark) Builder {
	if text == "" {
		b.watermark = nil
		return b
	}
	prop := props.Watermark{}
	if len(ps) > 0 {
		prop = ps[0]
	}
	prop.Text = text
	b.watermark = &prop
	return b
}

// WithHTMLLimits configures resource limits for AddHTML/FromHTML translation.
func (b *CfgBuilder) WithHTMLLimits(limits entity.HTMLLimits) Builder {
	b.htmlLimits = htmllimits.Normalize(limits)
	return b
}

// WithUnsafeNoHTMLLimits disables resource limits for AddHTML/FromHTML.
func (b *CfgBuilder) WithUnsafeNoHTMLLimits() Builder {
	b.htmlLimits = htmllimits.NoLimits()
	return b
}
