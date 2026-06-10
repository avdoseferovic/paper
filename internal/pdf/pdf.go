package pdf

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
	"time"
)

var gl struct {
	catalogSort  bool
	noCompress   bool // Initial zero value indicates compression
	creationDate time.Time
	modDate      time.Time
}

type fmtBuffer struct {
	bytes.Buffer
}

func (b *fmtBuffer) printf(fmtStr string, args ...any) {
	// Format straight into the buffer instead of allocating an intermediate
	// result string via Sprintf. fmt.Fprintf uses a pooled internal printer and
	// writes directly through bytes.Buffer.Write, so only the unavoidable
	// variadic arg boxing remains.
	fmt.Fprintf(&b.Buffer, fmtStr, args...)
}

func pdfNew(orientationStr, unitStr, sizeStr, fontDirStr string, size SizeType) *PDF {
	f := new(PDF)
	if orientationStr == "" {
		orientationStr = "p"
	} else {
		orientationStr = strings.ToLower(orientationStr)
	}
	if unitStr == "" {
		unitStr = "mm"
	}
	if sizeStr == "" {
		sizeStr = "A4"
	}
	if fontDirStr == "" {
		fontDirStr = "."
	}
	f.page = 0
	f.n = 2
	f.pages = make([]*bytes.Buffer, 0, 8)
	f.pages = append(f.pages, bytes.NewBufferString(""))
	f.pageSizes = make(map[int]SizeType)
	f.pageBoxes = make(map[int]map[string]PageBox)
	f.defPageBoxes = make(map[string]PageBox)
	f.state = 0
	f.fonts = make(map[string]fontDefType)
	f.fontFiles = make(map[string]fontFileType)
	f.diffs = make([]string, 0, 8)
	f.images = make(map[string]*ImageInfoType)
	f.pageLinks = make([][]linkType, 0, 8)
	f.pageLinks = append(f.pageLinks, make([]linkType, 0))
	f.links = make([]intLinkType, 0, 8)
	f.links = append(f.links, intLinkType{})
	f.aliasMap = make(map[string]string)
	f.inHeader = false
	f.inFooter = false
	f.lasth = 0
	f.fontFamily = ""
	f.fontStyle = ""
	f.SetFontSize(12)
	f.underline = false
	f.strikeout = false
	f.setDrawColor(0, 0, 0)
	f.setFillColor(0, 0, 0)
	f.setTextColor(0, 0, 0)
	f.colorFlag = false
	f.ws = 0
	f.fontpath = fontDirStr
	f.coreFonts = cloneCoreFontSet()

	switch unitStr {
	case "pt", "point":
		f.k = 1.0
	case "mm":
		f.k = 72.0 / 25.4
	case "cm":
		f.k = 72.0 / 2.54
	case "in", "inch":
		f.k = 72.0
	default:
		f.err = fmt.Errorf("%w %s", errIncorrectUnit, unitStr)
		return f
	}
	f.unitStr = unitStr
	f.stdPageSizes = cloneStandardPageSizes()
	if size.Wd > 0 && size.Ht > 0 {
		f.defPageSize = size
	} else {
		f.defPageSize = f.getpagesizestr(sizeStr)
		if f.err != nil {
			return f
		}
	}
	f.curPageSize = f.defPageSize

	switch orientationStr {
	case "p", "portrait":
		f.defOrientation = "P"
		f.w = f.defPageSize.Wd
		f.h = f.defPageSize.Ht

	case "l", "landscape":
		f.defOrientation = "L"
		f.w = f.defPageSize.Ht
		f.h = f.defPageSize.Wd
	default:
		f.err = fmt.Errorf("%w: %s", errIncorrectOrientation, orientationStr)
		return f
	}
	f.curOrientation = f.defOrientation
	f.wPt = f.w * f.k
	f.hPt = f.h * f.k

	margin := 28.35 / f.k
	f.SetMargins(margin, margin, margin)

	f.cMargin = margin / 10

	f.lineWidth = 0.567 / f.k

	f.SetAutoPageBreak(true, 2*margin)

	f.SetDisplayMode(displayModeDefault, displayModeDefault)
	if f.err != nil {
		return f
	}
	f.acceptPageBreak = func() bool {
		return f.autoPageBreak
	}

	f.SetCompression(!gl.noCompress)
	f.blendList = make([]blendModeType, 0, 8)
	f.blendList = append(f.blendList, blendModeType{})
	f.blendMap = make(map[string]int)
	f.blendMode = blendModeNormal
	f.alpha = 1
	f.gradientList = make([]gradientType, 0, 8)
	f.gradientList = append(f.gradientList, gradientType{})

	f.pdfVersion = "1.3"
	f.SetProducer("FPDF "+cnPDFVersion, true)
	f.catalogSort = gl.catalogSort
	f.creationDate = gl.creationDate
	f.modDate = gl.modDate
	f.userUnderlineThickness = 1
	return f
}

// NewCustom returns a pointer to a new PDF instance. Its methods are
// subsequently called to produce a single PDF document. NewCustom() is an
// alternative to New() that provides additional customization. The PageSize()
// example demonstrates this method.
func NewCustom(init *InitType) *PDF {
	return pdfNew(init.OrientationStr, init.UnitStr, init.SizeStr, init.FontDirStr, init.Size)
}

// Ok returns true if no processing errors have occurred.
func (f *PDF) Ok() bool {
	return f.err == nil
}

// Err returns true if a processing error has occurred.
func (f *PDF) Err() bool {
	return f.err != nil
}

// ClearError unsets the internal PDF error. This method should be used with
// care, as an internal error condition usually indicates an unrecoverable
// problem with the generation of a document. It is intended to deal with cases
// in which an error is used to select an alternate form of the document.
func (f *PDF) ClearError() {
	f.err = nil
}

// SetErrorf sets the internal PDF error with formatted text to halt PDF
// generation; this may facilitate error handling by application. If an error
// condition is already set, this call is ignored.
//
// See the documentation for printing in the standard fmt package for details
// about fmtStr and args.
func (f *PDF) SetErrorf(fmtStr string, args ...any) {
	if f.err == nil {
		f.err = dynamicErrorf(fmtStr, args...)
	}
}

// String satisfies the fmt.Stringer interface and summarizes the PDF
// instance.
func (f *PDF) String() string {
	return "PDF " + cnPDFVersion
}

// SetError sets an error to halt PDF generation. This may facilitate error
// handling by application. See also Ok(), Err() and Error().
func (f *PDF) SetError(err error) {
	if f.err == nil && err != nil {
		f.err = err
	}
}

// Error returns the internal PDF error; this will be nil if no error has occurred.
func (f *PDF) Error() error {
	return f.err
}

// SetCompression activates or deactivates page compression with zlib. When
// activated, the internal representation of each page is compressed, which
// leads to a compression ratio of about 2 for the resulting document.
// Compression is on by default.
func (f *PDF) SetCompression(compress bool) {
	f.compress = compress
}

// open begins a document
func (f *PDF) open() {
	f.state = 1
}

// Close terminates the PDF document. It is not necessary to call this method
// explicitly because Output(), OutputAndClose() and OutputFileAndClose() do it
// automatically. If the document contains no page, AddPage() is called to
// prevent the generation of an invalid document.
func (f *PDF) Close() {
	if f.err == nil {
		if f.clipNest > 0 {
			f.err = errClipProcedureOpen
		} else if f.transformNest > 0 {
			f.err = errTransformationProcedureOpen
		}
	}
	if f.err != nil {
		return
	}
	if f.state == 3 {
		return
	}
	if f.page == 0 {
		f.AddPage()
		if f.err != nil {
			return
		}
	}

	f.inFooter = true
	if f.footerFnc != nil {
		f.footerFnc()
	} else if f.footerFncLpi != nil {
		f.footerFncLpi(true)
	}
	f.inFooter = false

	f.endpage()

	f.enddoc()
}

type coreFontSet map[string]bool

var standardPageSizes = map[string]SizeType{
	"a3":      {841.89, 1190.55},
	"a4":      {595.28, 841.89},
	"a5":      {420.94, 595.28},
	"a6":      {297.64, 420.94},
	"a2":      {1190.55, 1683.78},
	"a1":      {1683.78, 2383.94},
	"letter":  {612, 792},
	"legal":   {612, 1008},
	"tabloid": {792, 1224},
}

var coreFontNames = []string{
	"courier",
	"helvetica",
	"times",
	"symbol",
	"zapfdingbats",
}

func cloneStandardPageSizes() map[string]SizeType {
	pageSizes := make(map[string]SizeType, len(standardPageSizes))
	maps.Copy(pageSizes, standardPageSizes)
	return pageSizes
}

func cloneCoreFontSet() coreFontSet {
	fonts := make(coreFontSet, len(coreFontNames))
	for _, name := range coreFontNames {
		fonts[name] = true
	}
	return fonts
}
