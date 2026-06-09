package pdf

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
)

// GetPageSize returns the current page's width and height. This is the paper's
// size. To compute the size of the area being used, subtract the margins (see
// GetMargins()).
func (f *PDF) GetPageSize() (float64, float64) {
	return f.w, f.h
}

// GetMargins returns the left, top, right, and bottom margins. The first three
// are set with the SetMargins() method. The bottom margin is set with the
// SetAutoPageBreak() method.
func (f *PDF) GetMargins() (float64, float64, float64, float64) {
	return f.lMargin, f.tMargin, f.rMargin, f.bMargin
}

// SetMargins defines the left, top and right margins. By default, they equal 1
// cm. Call this method to change them. If the value of the right margin is
// less than zero, it is set to the same as the left margin.
func (f *PDF) SetMargins(left, top, right float64) {
	f.lMargin = left
	f.tMargin = top
	if right < 0 {
		right = left
	}
	f.rMargin = right
}

// SetLeftMargin defines the left margin. The method can be called before
// creating the first page. If the current abscissa gets out of page, it is
// brought back to the margin.
func (f *PDF) SetLeftMargin(margin float64) {
	f.lMargin = margin
	if f.page > 0 && f.x < margin {
		f.x = margin
	}
}

// GetCellMargin returns the cell margin. This is the amount of space before
// and after the text within a cell that's left blank, and is in units passed
// to New(). It defaults to 1mm.
func (f *PDF) GetCellMargin() float64 {
	return f.cMargin
}

// SetCellMargin sets the cell margin. This is the amount of space before and
// after the text within a cell that's left blank, and is in units passed to
// New().
func (f *PDF) SetCellMargin(margin float64) {
	f.cMargin = margin
}

// SetPageBoxRec sets the page box for the current page, and any following
// pages. Allowable types are trim, trimbox, crop, cropbox, bleed, bleedbox,
// art and artbox box types are case insensitive. See SetPageBox() for a method
// that specifies the coordinates and extent of the page box individually.
func (f *PDF) SetPageBoxRec(t string, pb PageBox) {
	switch strings.ToLower(t) {
	case "trim":
		fallthrough
	case "trimbox":
		t = "TrimBox"
	case "crop":
		fallthrough
	case "cropbox":
		t = "CropBox"
	case "bleed":
		fallthrough
	case "bleedbox":
		t = "BleedBox"
	case "art":
		fallthrough
	case "artbox":
		t = "ArtBox"
	default:
		f.err = fmt.Errorf("%w: %s", errInvalidPageBoxType, t)
		return
	}

	pb.X *= f.k
	pb.Y *= f.k
	pb.Wd = (pb.Wd * f.k) + pb.X
	pb.Ht = (pb.Ht * f.k) + pb.Y

	if f.page > 0 {
		f.pageBoxes[f.page][t] = pb
	}

	f.defPageBoxes[t] = pb
}

// SetPageBox sets the page box for the current page, and any following pages.
// Allowable types are trim, trimbox, crop, cropbox, bleed, bleedbox, art and
// artbox box types are case insensitive.
func (f *PDF) SetPageBox(t string, x, y, wd, ht float64) {
	f.SetPageBoxRec(t, PageBox{SizeType{Wd: wd, Ht: ht}, PointType{X: x, Y: y}})
}

// SetPage sets the current page to that of a valid page in the PDF document.
// pageNum is one-based. The SetPage() example demonstrates this method.
func (f *PDF) SetPage(pageNum int) {
	if (pageNum > 0) && (pageNum < len(f.pages)) {
		f.page = pageNum
	}
}

// PageCount returns the number of pages currently in the document. Since page
// numbers in gofpdf are one-based, the page count is the same as the page
// number of the current last page.
func (f *PDF) PageCount() int {
	return len(f.pages) - 1
}

// SetHeaderFuncMode sets the function that lets the application render the
// page header. See SetHeaderFunc() for more details. The value for homeMode
// should be set to true to have the current position set to the left and top
// margin after the header function is called.
func (f *PDF) SetHeaderFuncMode(fnc func(), homeMode bool) {
	f.headerFnc = fnc
	f.headerHomeMode = homeMode
}

// SetHeaderFunc sets the function that lets the application render the page
// header. The specified function is automatically called by AddPage() and
// should not be called directly by the application. The implementation in PDF
// is empty, so you have to provide an appropriate function if you want page
// headers. fnc will typically be a closure that has access to the PDF
// instance and other document generation variables.
//
// A header is a convenient place to put background content that repeats on
// each page such as a watermark. When this is done, remember to reset the X
// and Y values so the normal content begins where expected. Including a
// watermark on each page is demonstrated in the example for TransformRotate.
//
// This method is demonstrated in the example for AddPage().
func (f *PDF) SetHeaderFunc(fnc func()) {
	f.headerFnc = fnc
}

// SetFooterFunc sets the function that lets the application render the page
// footer. The specified function is automatically called by AddPage() and
// Close() and should not be called directly by the application. The
// implementation in PDF is empty, so you have to provide an appropriate
// function if you want page footers. fnc will typically be a closure that has
// access to the PDF instance and other document generation variables. See
// SetFooterFuncLpi for a similar function that passes a last page indicator.
//
// This method is demonstrated in the example for AddPage().
func (f *PDF) SetFooterFunc(fnc func()) {
	f.footerFnc = fnc
	f.footerFncLpi = nil
}

// SetFooterFuncLpi sets the function that lets the application render the page
// footer. The specified function is automatically called by AddPage() and
// Close() and should not be called directly by the application. It is passed a
// boolean that is true if the last page of the document is being rendered. The
// implementation in PDF is empty, so you have to provide an appropriate
// function if you want page footers. fnc will typically be a closure that has
// access to the PDF instance and other document generation variables.
func (f *PDF) SetFooterFuncLpi(fnc func(lastPage bool)) {
	f.footerFncLpi = fnc
	f.footerFnc = nil
}

// SetTopMargin defines the top margin. The method can be called before
// creating the first page.
func (f *PDF) SetTopMargin(margin float64) {
	f.tMargin = margin
}

// SetRightMargin defines the right margin. The method can be called before
// creating the first page.
func (f *PDF) SetRightMargin(margin float64) {
	f.rMargin = margin
}

// GetAutoPageBreak returns true if automatic pages breaks are enabled, false
// otherwise. This is followed by the triggering limit from the bottom of the
// page. This value applies only if automatic page breaks are enabled.
func (f *PDF) GetAutoPageBreak() (bool, float64) {
	return f.autoPageBreak, f.bMargin
}

// SetAutoPageBreak enables or disables the automatic page breaking mode. When
// enabling, the second parameter is the distance from the bottom of the page
// that defines the triggering limit. By default, the mode is on and the margin
// is 2 cm.
func (f *PDF) SetAutoPageBreak(auto bool, margin float64) {
	f.autoPageBreak = auto
	f.bMargin = margin
	f.pageBreakTrigger = f.h - margin
}

// PageSize returns the width and height of the specified page in the units
// established in New(). These return values are followed by the unit of
// measure itself. If pageNum is zero or otherwise out of bounds, it returns
// the default page size, that is, the size of the page that would be added by
// AddPage().
func (f *PDF) PageSize(pageNum int) (float64, float64, string) {
	sz, ok := f.pageSizes[pageNum]
	if ok {
		sz.Wd, sz.Ht = sz.Wd/f.k, sz.Ht/f.k
	} else {
		sz = f.defPageSize
	}
	return sz.Wd, sz.Ht, f.unitStr
}

// AddPageFormat adds a new page with non-default orientation or size. See
// AddPage() for more details.
//
// See New() for a description of orientationStr.
//
// size specifies the size of the new page in the units established in New().
//
// The PageSize() example demonstrates this method.
func (f *PDF) AddPageFormat(orientationStr string, size SizeType) {
	if f.err != nil {
		return
	}
	if f.page != len(f.pages)-1 {
		f.page = len(f.pages) - 1
	}
	if f.state == 0 {
		f.open()
	}
	familyStr := f.fontFamily
	style := f.fontStyle
	if f.underline {
		style += "U"
	}
	if f.strikeout {
		style += "S"
	}
	fontsize := f.fontSizePt
	lw := f.lineWidth
	dc := f.color.draw
	fc := f.color.fill
	tc := f.color.text
	cf := f.colorFlag

	if f.page > 0 {
		f.inFooter = true

		if f.footerFnc != nil {
			f.footerFnc()
		} else if f.footerFncLpi != nil {
			f.footerFncLpi(false)
		}
		f.inFooter = false

		f.endpage()
	}

	f.beginpage(orientationStr, size)

	f.outf("%d J", f.capStyle)

	f.outf("%d j", f.joinStyle)

	f.lineWidth = lw
	f.outf("%.2f w", lw*f.k)

	if len(f.dashArray) > 0 {
		f.outputDashPattern()
	}

	if familyStr != "" {
		f.SetFont(familyStr, style, fontsize)
		if f.err != nil {
			return
		}
	}

	f.color.draw = dc
	if dc.str != "0 G" {
		f.out(dc.str)
	}
	f.color.fill = fc
	if fc.str != "0 g" {
		f.out(fc.str)
	}
	f.color.text = tc
	f.colorFlag = cf

	if f.headerFnc != nil {
		f.inHeader = true
		f.headerFnc()
		f.inHeader = false
		if f.headerHomeMode {
			f.SetHomeXY()
		}
	}

	if f.lineWidth != lw {
		f.lineWidth = lw
		f.outf("%.2f w", lw*f.k)
	}

	if familyStr != "" {
		f.SetFont(familyStr, style, fontsize)
		if f.err != nil {
			return
		}
	}

	if f.color.draw.str != dc.str {
		f.color.draw = dc
		f.out(dc.str)
	}
	if f.color.fill.str != fc.str {
		f.color.fill = fc
		f.out(fc.str)
	}
	f.color.text = tc
	f.colorFlag = cf
}

// AddPage adds a new page to the document. If a page is already present, the
// Footer() method is called first to output the footer. Then the page is
// added, the current position set to the top-left corner according to the left
// and top margins, and Header() is called to display the header.
//
// The font which was set before calling is automatically restored. There is no
// need to call SetFont() again if you want to continue with the same font. The
// same is true for colors and line width.
//
// The origin of the coordinate system is at the top-left corner and increasing
// ordinates go downwards.
//
// See AddPageFormat() for a version of this method that allows the page size
// and orientation to be different than the default.
func (f *PDF) AddPage() {
	if f.err != nil {
		return
	}

	f.AddPageFormat(f.defOrientation, f.defPageSize)
}

// PageNo returns the current page number.
//
// See the example for AddPage() for a demonstration of this method.
func (f *PDF) PageNo() int {
	return f.page
}

// SetAcceptPageBreakFunc allows the application to control where page breaks
// occur.
//
// fnc is an application function (typically a closure) that is called by the
// library whenever a page break condition is met. The break is issued if true
// is returned. The default implementation returns a value according to the
// mode selected by SetAutoPageBreak. The function provided should not be
// called by the application.
//
// See the example for SetLeftMargin() to see how this function can be used to
// manage multiple columns.
func (f *PDF) SetAcceptPageBreakFunc(fnc func() bool) {
	f.acceptPageBreak = fnc
}

// Ln performs a line break. The current abscissa goes back to the left margin
// and the ordinate increases by the amount passed in parameter. A negative
// value of h indicates the height of the last printed cell.
//
// This method is demonstrated in the example for MultiCell.
func (f *PDF) Ln(h float64) {
	f.x = f.lMargin
	if h < 0 {
		f.y += f.lasth
	} else {
		f.y += h
	}
}

func (f *PDF) getpagesizestr(sizeStr string) SizeType {
	if f.err != nil {
		return SizeType{}
	}
	sizeStr = strings.ToLower(sizeStr)
	size, ok := f.stdPageSizes[sizeStr]
	if ok {
		size.Wd /= f.k
		size.Ht /= f.k
	} else {
		f.err = fmt.Errorf("%w %s", errUnknownPageSize, sizeStr)
	}
	return size
}

// GetPageSizeStr returns the SizeType for the given sizeStr (that is A4, A3, etc..)
func (f *PDF) GetPageSizeStr(sizeStr string) SizeType {
	return f.getpagesizestr(sizeStr)
}

func (f *PDF) beginpage(orientationStr string, size SizeType) {
	if f.err != nil {
		return
	}
	f.page++

	f.pageBoxes[f.page] = make(map[string]PageBox)
	maps.Copy(f.pageBoxes[f.page], f.defPageBoxes)
	f.pages = append(f.pages, bytes.NewBufferString(""))
	f.pageLinks = append(f.pageLinks, make([]linkType, 0))
	f.pageAttachments = append(f.pageAttachments, []annotationAttach{})
	f.state = 2
	f.x = f.lMargin
	f.y = f.tMargin
	f.fontFamily = ""

	if orientationStr == "" {
		orientationStr = f.defOrientation
	} else {
		orientationStr = strings.ToUpper(orientationStr[0:1])
	}
	if orientationStr != f.curOrientation || size.Wd != f.curPageSize.Wd || size.Ht != f.curPageSize.Ht {
		if orientationStr == "P" {
			f.w = size.Wd
			f.h = size.Ht
		} else {
			f.w = size.Ht
			f.h = size.Wd
		}
		f.wPt = f.w * f.k
		f.hPt = f.h * f.k
		f.pageBreakTrigger = f.h - f.bMargin
		f.curOrientation = orientationStr
		f.curPageSize = size
	}
	if orientationStr != f.defOrientation || size.Wd != f.defPageSize.Wd || size.Ht != f.defPageSize.Ht {
		f.pageSizes[f.page] = SizeType{f.wPt, f.hPt}
	}
}

func (f *PDF) endpage() {
	f.EndLayer()
	f.state = 1
}

func (f *PDF) putpages() {
	var wPt, hPt float64
	var pageSize SizeType
	var ok bool
	nb := f.page
	if len(f.aliasNbPagesStr) > 0 {
		f.RegisterAlias(f.aliasNbPagesStr, sprintf("%d", nb))
	}
	f.replaceAliases()
	if f.defOrientation == "P" {
		wPt = f.defPageSize.Wd * f.k
		hPt = f.defPageSize.Ht * f.k
	} else {
		wPt = f.defPageSize.Ht * f.k
		hPt = f.defPageSize.Wd * f.k
	}
	pagesObjectNumbers := make([]int, nb+1)
	for n := 1; n <= nb; n++ {
		f.newobj()
		pagesObjectNumbers[n] = f.n
		f.out("<</Type /Page")
		f.out("/Parent 1 0 R")
		pageSize, ok = f.pageSizes[n]
		if ok {
			f.outf("/MediaBox [0 0 %.2f %.2f]", pageSize.Wd, pageSize.Ht)
		}
		for t, pb := range f.pageBoxes[n] {
			f.outf("/%s [%.2f %.2f %.2f %.2f]", t, pb.X, pb.Y, pb.Wd, pb.Ht)
		}
		f.out("/Resources 2 0 R")

		f.putPageAnnotations(n, hPt)
		if f.pdfVersion > "1.3" {
			f.out("/Group <</Type /Group /S /Transparency /CS /DeviceRGB>>")
		}
		f.outf("/Contents %d 0 R>>", f.n+1)
		f.out("endobj")

		f.newobj()
		if f.compress {
			data := sliceCompress(f.pages[n].Bytes())
			f.outf("<</Filter /FlateDecode /Length %d>>", len(data))
			f.putstream(data)
		} else {
			f.outf("<</Length %d>>", f.pages[n].Len())
			f.putstream(f.pages[n].Bytes())
		}
		f.out("endobj")
	}

	f.offsets[1] = f.buffer.Len()
	f.out("1 0 obj")
	f.out("<</Type /Pages")
	var kids fmtBuffer
	kids.printf("/Kids [")
	for i := 1; i <= nb; i++ {
		kids.printf("%d 0 R ", pagesObjectNumbers[i])
	}
	kids.printf("]")
	f.out(kids.String())
	f.outf("/Count %d", nb)
	f.outf("/MediaBox [0 0 %.2f %.2f]", wPt, hPt)
	f.out(">>")
	f.out("endobj")
}

func (f *PDF) putPageAnnotations(pageNum int, defaultHeight float64) {
	if len(f.pageLinks[pageNum])+len(f.pageAttachments[pageNum]) == 0 {
		return
	}
	var annots fmtBuffer
	annots.printf("/Annots [")
	for _, pl := range f.pageLinks[pageNum] {
		f.putPageLinkAnnotation(&annots, pl, defaultHeight)
	}
	f.putAttachmentAnnotationLinks(&annots, pageNum)
	annots.printf("]")
	f.out(annots.String())
}

func (f *PDF) putPageLinkAnnotation(annots *fmtBuffer, pl linkType, defaultHeight float64) {
	annots.printf("<</Type /Annot /Subtype /Link /Rect [%.2f %.2f %.2f %.2f] /Border [0 0 0] ",
		pl.x, pl.y, pl.x+pl.wd, pl.y-pl.ht)
	if pl.link == 0 {
		annots.printf("/A <</S /URI /URI %s>>>>", f.textstring(pl.linkStr))
		return
	}

	l := f.links[pl.link]
	h := defaultHeight
	if sz, ok := f.pageSizes[l.page]; ok {
		h = sz.Ht
	}
	annots.printf("/Dest [%d 0 R /XYZ 0 %.2f null]>>", 1+2*l.page, h-l.y*f.k)
}
