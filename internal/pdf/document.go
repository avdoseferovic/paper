package pdf

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// SetDisplayMode sets advisory display directives for the document viewer.
// Pages can be displayed entirely on screen, occupy the full width of the
// window, use real size, be scaled by a specific zooming factor or use viewer
// default (configured in the Preferences menu of Adobe Reader). The page
// layout can be specified so that pages are displayed individually or in
// pairs.
//
// zoomStr can be "fullpage" to display the entire page on screen, "fullwidth"
// to use maximum width of window, "real" to use real size (equivalent to 100%
// zoom) or "default" to use viewer default mode.
//
// layoutStr can be "single" (or "SinglePage") to display one page at once,
// "continuous" (or "OneColumn") to display pages continuously, "two" (or
// "TwoColumnLeft") to display two pages on two columns with odd-numbered pages
// on the left, or "TwoColumnRight" to display two pages on two columns with
// odd-numbered pages on the right, or "TwoPageLeft" to display pages two at a
// time with odd-numbered pages on the left, or "TwoPageRight" to display pages
// two at a time with odd-numbered pages on the right, or "default" to use
// viewer default mode.
func (f *PDF) SetDisplayMode(zoomStr, layoutStr string) {
	if f.err != nil {
		return
	}
	if layoutStr == "" {
		layoutStr = displayModeDefault
	}
	switch zoomStr {
	case "fullpage", "fullwidth", "real", displayModeDefault:
		f.zoomMode = zoomStr
	default:
		f.err = fmt.Errorf("%w: %s", errIncorrectZoomDisplayMode, zoomStr)
		return
	}
	switch layoutStr {
	case "single", "continuous", "two", displayModeDefault, "SinglePage", "OneColumn",
		"TwoColumnLeft", "TwoColumnRight", "TwoPageLeft", "TwoPageRight":
		f.layoutMode = layoutStr
	default:
		f.err = fmt.Errorf("%w: %s", errIncorrectLayoutDisplayMode, layoutStr)
		return
	}
}

// SetProducer defines the producer of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *PDF) SetProducer(producerStr string, isUTF8 bool) {
	if isUTF8 {
		producerStr = utf8toutf16(producerStr)
	}
	f.producer = producerStr
}

// SetTitle defines the title of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *PDF) SetTitle(titleStr string, isUTF8 bool) {
	if isUTF8 {
		titleStr = utf8toutf16(titleStr)
	}
	f.title = titleStr
}

// SetSubject defines the subject of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *PDF) SetSubject(subjectStr string, isUTF8 bool) {
	if isUTF8 {
		subjectStr = utf8toutf16(subjectStr)
	}
	f.subject = subjectStr
}

// SetAuthor defines the author of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *PDF) SetAuthor(authorStr string, isUTF8 bool) {
	if isUTF8 {
		authorStr = utf8toutf16(authorStr)
	}
	f.author = authorStr
}

// SetKeywords defines the keywords of the document. keywordStr is a
// space-delimited string, for example "invoice August". isUTF8 indicates if
// the string is encoded
func (f *PDF) SetKeywords(keywordsStr string, isUTF8 bool) {
	if isUTF8 {
		keywordsStr = utf8toutf16(keywordsStr)
	}
	f.keywords = keywordsStr
}

// SetCreator defines the creator of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *PDF) SetCreator(creatorStr string, isUTF8 bool) {
	if isUTF8 {
		creatorStr = utf8toutf16(creatorStr)
	}
	f.creator = creatorStr
}

// SetXmpMetadata defines XMP metadata that will be embedded with the document.
func (f *PDF) SetXmpMetadata(xmpStream []byte) {
	f.xmp = xmpStream
}

// AliasNbPages defines an alias for the total number of pages. It will be
// substituted as the document is closed. An empty string is replaced with the
// string "{nb}".
//
// See the example for AddPage() for a demonstration of this method.
func (f *PDF) AliasNbPages(aliasStr string) {
	if aliasStr == "" {
		aliasStr = "{nb}"
	}
	f.aliasNbPagesStr = aliasStr
}

// RTL enables right-to-left mode
func (f *PDF) RTL() {
	f.isRTL = true
}

// LTR disables right-to-left mode
func (f *PDF) LTR() {
	f.isRTL = false
}

// SetCatalogSort sets a flag that will be used, if true, to consistently order
// the document's internal resource catalogs. This method is typically only
// used for test purposes to facilitate PDF comparison.
func (f *PDF) SetCatalogSort(flag bool) {
	f.catalogSort = flag
}

// SetCreationDate fixes the document's internal CreationDate value. By
// default, the time when the document is generated is used for this value.
// This method is typically only used for testing purposes to facilitate PDF
// comparison. Specify a zero-value time to revert to the default behavior.
func (f *PDF) SetCreationDate(tm time.Time) {
	f.creationDate = tm
}

// SetModificationDate fixes the document's internal ModDate value.
// See `SetCreationDate` for more details.
func (f *PDF) SetModificationDate(tm time.Time) {
	f.modDate = tm
}

// SetJavascript adds Adobe JavaScript to the document.
func (f *PDF) SetJavascript(script string) {
	f.javascript = &script
}

// RegisterAlias adds an (alias, replacement) pair to the document so we can
// replace all occurrences of that alias after writing but before the document
// is closed. Functions ExamplePDF_RegisterAlias() and
// ExamplePDF_RegisterAlias_utf8() in fpdf_test.go demonstrate this method.
func (f *PDF) RegisterAlias(alias, replacement string) {
	f.aliasMap[alias] = replacement
}

func (f *PDF) replaceAliases() {
	for mode := range 2 {
		for alias, replacement := range f.aliasMap {
			if mode == 1 {
				alias = utf8toutf16(alias, false)
				replacement = utf8toutf16(replacement, false)
			}
			for n := 1; n <= f.page; n++ {
				s := f.pages[n].String()
				if strings.Contains(s, alias) {
					s = strings.ReplaceAll(s, alias, replacement)
					f.pages[n].Truncate(0)
					f.pages[n].WriteString(s)
				}
			}
		}
	}
}

func (f *PDF) putjavascript() {
	if f.javascript == nil {
		return
	}

	f.newobj()
	f.nJs = f.n
	f.out("<<")
	f.outf("/Names [(EmbeddedJS) %d 0 R]", f.n+1)
	f.out(">>")
	f.out("endobj")
	f.newobj()
	f.out("<<")
	f.out("/S /JavaScript")
	f.outf("/JS %s", f.textstring(*f.javascript))
	f.out(">>")
	f.out("endobj")
}

// returns Now() if tm is zero
func timeOrNow(tm time.Time) time.Time {
	if tm.IsZero() {
		return time.Now()
	}
	return tm
}

func (f *PDF) putinfo() {
	if len(f.producer) > 0 {
		f.outf("/Producer %s", f.textstring(f.producer))
	}
	if len(f.title) > 0 {
		f.outf("/Title %s", f.textstring(f.title))
	}
	if len(f.subject) > 0 {
		f.outf("/Subject %s", f.textstring(f.subject))
	}
	if len(f.author) > 0 {
		f.outf("/Author %s", f.textstring(f.author))
	}
	if len(f.keywords) > 0 {
		f.outf("/Keywords %s", f.textstring(f.keywords))
	}
	if len(f.creator) > 0 {
		f.outf("/Creator %s", f.textstring(f.creator))
	}
	creation := timeOrNow(f.creationDate)
	f.outf("/CreationDate %s", f.textstring("D:"+creation.Format("20060102150405")))
	mod := timeOrNow(f.modDate)
	f.outf("/ModDate %s", f.textstring("D:"+mod.Format("20060102150405")))
}

func (f *PDF) putcatalog() {
	f.out("/Type /Catalog")
	f.out("/Pages 1 0 R")
	switch f.zoomMode {
	case "fullpage":
		f.out("/OpenAction [3 0 R /Fit]")
	case "fullwidth":
		f.out("/OpenAction [3 0 R /FitH null]")
	case "real":
		const pdfNullOperand = "null"
		f.outf("/OpenAction [3 0 R /XYZ %s %s 1]", pdfNullOperand, pdfNullOperand)
	}

	switch f.layoutMode {
	case "single", "SinglePage":
		f.out("/PageLayout /SinglePage")
	case "continuous", "OneColumn":
		f.out("/PageLayout /OneColumn")
	case "two", "TwoColumnLeft":
		f.out("/PageLayout /TwoColumnLeft")
	case "TwoColumnRight":
		f.out("/PageLayout /TwoColumnRight")
	case "TwoPageLeft", "TwoPageRight":
		if f.pdfVersion < "1.5" {
			f.pdfVersion = "1.5"
		}
		f.out("/PageLayout /" + f.layoutMode)
	}

	if len(f.outlines) > 0 {
		f.outf("/Outlines %d 0 R", f.outlineRoot)
		f.out("/PageMode /UseOutlines")
	}

	if f.javascript != nil {
		f.out("/Names <<")
		f.outf("/JavaScript %d 0 R", f.nJs)
		f.out(">>")
	}
}

// SetProtection applies certain constraints on the finished PDF document.
//
// actionFlag is a bitflag that controls various document operations.
// CnProtectPrint allows the document to be printed. CnProtectModify allows a
// document to be modified by a PDF editor. CnProtectCopy allows text and
// images to be copied into the system clipboard. CnProtectAnnotForms allows
// annotations and forms to be added by a PDF editor. These values can be
// combined by or-ing them together, for example,
// CnProtectCopy|CnProtectModify. This flag is advisory; not all PDF readers
// implement the constraints that this argument attempts to control.
//
// userPassStr specifies the password that will need to be provided to view the
// contents of the PDF. The permissions specified by actionFlag will apply.
//
// ownerPassStr specifies the password that will need to be provided to gain
// full access to the document regardless of the actionFlag value. An empty
// string for this argument will be replaced with a random value, effectively
// prohibiting full access to the document.
func (f *PDF) SetProtection(actionFlag byte, userPassStr, ownerPassStr string) {
	if f.err != nil {
		return
	}
	f.protect.setProtection(actionFlag, userPassStr, ownerPassStr)
}

// SetProtectionAlgorithm selects the encryption algorithm for protected PDFs.
// It must be called before SetProtection.
func (f *PDF) SetProtectionAlgorithm(algorithm ProtectionAlgorithm) {
	if f.err != nil {
		return
	}
	f.protect.algorithm = algorithm
}

// OutputAndClose sends the PDF document to the writer specified by w. This
// method will close both f and w, even if an error is detected and no document
// is produced.
func (f *PDF) OutputAndClose(w io.WriteCloser) error {
	outErr := f.Output(w)
	closeErr := w.Close()
	if outErr != nil {
		return outErr
	}
	if closeErr != nil {
		f.err = closeErr
	}
	return f.err
}

// OutputFileAndClose creates or truncates the file specified by fileStr and
// writes the PDF document to it. This method will close f and the newly
// written file, even if an error is detected and no document is produced.
//
// Most examples demonstrate the use of this method.
func (f *PDF) OutputFileAndClose(fileStr string) error {
	if f.err != nil {
		return f.err
	}

	pdfFile, err := os.Create(fileStr)
	if err != nil {
		f.err = err
		return f.err
	}
	outErr := f.Output(pdfFile)
	closeErr := pdfFile.Close()
	if outErr != nil {
		f.err = outErr
	} else if closeErr != nil {
		f.err = closeErr
	}

	return f.err
}

// Output sends the PDF document to the writer specified by w. No output will
// take place if an error has occurred in the document generation process. w
// remains open after this function returns. After returning, f is in a closed
// state and its methods should not be called.
func (f *PDF) Output(w io.Writer) error {
	if f.err != nil {
		return f.err
	}

	if f.state < 3 {
		f.Close()
	}
	_, err := f.buffer.WriteTo(w)
	if err != nil {
		f.err = err
	}
	return f.err
}

// Escape special characters in strings
func (f *PDF) escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// textstring formats a text string
func (f *PDF) textstring(s string) string {
	if f.protect.encrypted {
		objectNumber, ok := checkedUint32(f.n)
		if !ok {
			f.err = staticErrorf(errObjectNumberOutOfRange, "%d", f.n)
			return "(" + f.escape(s) + ")"
		}
		b := []byte(s)
		encrypted, err := f.protect.encryptBytes(objectNumber, b)
		if err != nil {
			f.err = err
			return "(" + f.escape(s) + ")"
		}
		b = encrypted
		s = string(b)
	}
	return "(" + f.escape(s) + ")"
}

// newobj begins a new object
func (f *PDF) newobj() {
	f.n++
	for j := len(f.offsets); j <= f.n; j++ {
		f.offsets = append(f.offsets, 0)
	}
	f.offsets[f.n] = f.buffer.Len()
	f.outf("%d 0 obj", f.n)
}

func (f *PDF) encryptedStream(b []byte) []byte {
	if f.protect.encrypted {
		objectNumber, ok := checkedUint32(f.n)
		if !ok {
			f.err = staticErrorf(errObjectNumberOutOfRange, "%d", f.n)
			return nil
		}
		encrypted, err := f.protect.encryptBytes(objectNumber, b)
		if err != nil {
			f.err = err
			return nil
		}
		return encrypted
	}
	return b
}

func (f *PDF) putstream(b []byte) {
	f.out("stream")
	f.out(string(b))
	f.out("endstream")
}

// out; Add a line to the document
func (f *PDF) out(s string) {
	if f.state == 2 {
		f.pages[f.page].WriteString(s)
		f.pages[f.page].WriteString("\n")
	} else {
		f.buffer.WriteString(s)
		f.buffer.WriteString("\n")
	}
}

// outbuf adds a buffered line to the document
func (f *PDF) outbuf(r io.Reader) {
	if f.state == 2 {
		_, err := f.pages[f.page].ReadFrom(r)
		if err != nil {
			f.err = err
			return
		}
		f.pages[f.page].WriteString("\n")
	} else {
		_, err := f.buffer.ReadFrom(r)
		if err != nil {
			f.err = err
			return
		}
		f.buffer.WriteString("\n")
	}
}

// RawWriteStr writes a string directly to the PDF generation buffer. This is a
// low-level function that is not required for normal PDF construction. An
// understanding of the PDF specification is needed to use this method
// correctly.
func (f *PDF) RawWriteStr(str string) {
	f.out(str)
}

// RawWriteBuf writes the contents of the specified buffer directly to the PDF
// generation buffer. This is a low-level function that is not required for
// normal PDF construction. An understanding of the PDF specification is needed
// to use this method correctly.
func (f *PDF) RawWriteBuf(r io.Reader) {
	f.outbuf(r)
}

// outf adds a formatted line to the document
func (f *PDF) outf(fmtStr string, args ...any) {
	// Format directly into the active buffer rather than building an
	// intermediate string with sprintf and copying it in via out. This removed
	// the largest object allocator in the profile (the per-line result string,
	// ~53% of fmt.Sprintf calls). Output is byte-identical; only the unavoidable
	// variadic arg boxing remains.
	buf := &f.buffer.Buffer
	if f.state == 2 {
		buf = f.pages[f.page]
	}
	fmt.Fprintf(buf, fmtStr, args...)
	buf.WriteByte('\n')
}

func (f *PDF) putheader() {
	if len(f.blendMap) > 0 && f.pdfVersion < "1.4" {
		f.pdfVersion = "1.4"
	}
	f.outf("%%PDF-%s", f.pdfVersion)
}

func (f *PDF) puttrailer() {
	f.outf("/Size %d", f.n+1)
	f.outf("/Root %d 0 R", f.n)
	f.outf("/Info %d 0 R", f.n-1)
	if f.protect.encrypted {
		f.outf("/Encrypt %d 0 R", f.protect.objNum)
		if f.protect.algorithm == ProtectionAES128 && len(f.protect.fileID) > 0 {
			id := pdfHexString(f.protect.fileID)
			f.outf("/ID [%s%s]", id, id)
		} else {
			f.out("/ID [()()]")
		}
	}
}

func pdfHexString(data []byte) string {
	return "<" + hex.EncodeToString(data) + ">"
}

func (f *PDF) putxmp() {
	if len(f.xmp) == 0 {
		return
	}
	f.newobj()
	stream := f.encryptedStream(f.xmp)
	if f.err != nil {
		return
	}
	f.outf("<< /Type /Metadata /Subtype /XML /Length %d >>", len(stream))
	f.putstream(stream)
	f.out("endobj")
}

func (f *PDF) enddoc() {
	if f.err != nil {
		return
	}
	f.putheader()

	f.putpages()
	f.putresources()
	if f.err != nil {
		return
	}

	f.putbookmarks()

	f.putxmp()

	f.newobj()
	f.out("<<")
	f.putinfo()
	f.out(">>")
	f.out("endobj")

	f.newobj()
	f.out("<<")
	f.putcatalog()
	f.out(">>")
	f.out("endobj")

	o := f.buffer.Len()
	f.out("xref")
	f.outf("0 %d", f.n+1)
	f.out("0000000000 65535 f ")
	for j := 1; j <= f.n; j++ {
		f.outf("%010d 00000 n ", f.offsets[j])
	}

	f.out("trailer")
	f.out("<<")
	f.puttrailer()
	f.out(">>")
	f.out("startxref")
	f.outf("%d", o)
	f.out("%%EOF")
	f.state = 3
}

func (f *PDF) putxobjectdict() {
	var image *ImageInfoType
	keyList := make([]string, 0, len(f.images))
	for key := range f.images {
		keyList = append(keyList, key)
	}
	if f.catalogSort {
		sort.SliceStable(keyList, func(i, j int) bool { return f.images[keyList[i]].i < f.images[keyList[j]].i })
	}
	for _, key := range keyList {
		image = f.images[key]
		f.outf("/I%s %d 0 R", image.i, image.n)
	}
}

func (f *PDF) putresourcedict() {
	f.out("/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")
	f.out("/Font <<")
	{
		keyList := make([]string, 0, len(f.fonts))
		var font fontDefType
		var key string
		for key = range f.fonts {
			keyList = append(keyList, key)
		}
		if f.catalogSort {
			sort.SliceStable(keyList, func(i, j int) bool { return f.fonts[keyList[i]].i < f.fonts[keyList[j]].i })
		}
		for _, key = range keyList {
			font = f.fonts[key]
			f.outf("/F%s %d 0 R", font.i, font.N)
		}
	}
	f.out(">>")
	f.out("/XObject <<")
	f.putxobjectdict()
	f.out(">>")
	count := len(f.blendList)
	if count > 1 {
		f.out("/ExtGState <<")
		for j := 1; j < count; j++ {
			f.outf("/GS%d %d 0 R", j, f.blendList[j].objNum)
		}
		f.out(">>")
	}
	count = len(f.gradientList)
	if count > 1 {
		f.out("/Shading <<")
		for j := 1; j < count; j++ {
			f.outf("/Sh%d %d 0 R", j, f.gradientList[j].objNum)
		}
		f.out(">>")
	}
}

func (f *PDF) putresources() {
	if f.err != nil {
		return
	}
	f.putBlendModes()
	f.putGradients()
	f.putfonts()
	if f.err != nil {
		return
	}
	f.putimages()

	f.offsets[2] = f.buffer.Len()
	f.out("2 0 obj")
	f.out("<<")
	f.putresourcedict()
	f.out(">>")
	f.out("endobj")
	f.putjavascript()
	if f.protect.encrypted {
		f.newobj()
		f.protect.objNum = f.n
		f.out("<<")
		f.out("/Filter /Standard")
		if f.protect.algorithm == ProtectionAES128 {
			f.out("/V 4")
			f.out("/R 4")
			f.out("/Length 128")
			f.outf("/O %s", pdfHexString(f.protect.oValue))
			f.outf("/U %s", pdfHexString(f.protect.uValue))
			f.outf("/P %d", f.protect.pValue)
			f.out("/CF <</StdCF <</CFM /AESV2 /AuthEvent /DocOpen /Length 128>>>>")
			f.out("/StmF /StdCF")
			f.out("/StrF /StdCF")
		} else {
			f.out("/V 1")
			f.out("/R 2")
			f.outf("/O (%s)", f.escape(string(f.protect.oValue)))
			f.outf("/U (%s)", f.escape(string(f.protect.uValue)))
			f.outf("/P %d", f.protect.pValue)
		}
		f.out(">>")
		f.out("endobj")
	}
}

// GetConversionRatio returns the conversion ratio based on the unit given when
// creating the PDF.
func (f *PDF) GetConversionRatio() float64 {
	return f.k
}

// GetXY returns the abscissa and ordinate of the current position.
//
// Note: the value returned for the abscissa will be affected by the current
// cell margin. To account for this, you may need to either add the value
// returned by GetCellMargin() to it or call SetCellMargin(0) to remove the
// cell margin.
func (f *PDF) GetXY() (float64, float64) {
	return f.x, f.y
}

// GetX returns the abscissa of the current position.
//
// Note: the value returned will be affected by the current cell margin. To
// account for this, you may need to either add the value returned by
// GetCellMargin() to it or call SetCellMargin(0) to remove the cell margin.
func (f *PDF) GetX() float64 {
	return f.x
}

// SetX defines the abscissa of the current position. If the passed value is
// negative, it is relative to the right of the page.
func (f *PDF) SetX(x float64) {
	if x >= 0 {
		f.x = x
	} else {
		f.x = f.w + x
	}
}

// GetY returns the ordinate of the current position.
func (f *PDF) GetY() float64 {
	return f.y
}

// SetY moves the current abscissa back to the left margin and sets the
// ordinate. If the passed value is negative, it is relative to the bottom of
// the page.
func (f *PDF) SetY(y float64) {
	f.x = f.lMargin
	if y >= 0 {
		f.y = y
	} else {
		f.y = f.h + y
	}
}

// SetHomeXY is a convenience method that sets the current position to the left
// and top margins.
func (f *PDF) SetHomeXY() {
	f.SetY(f.tMargin)
	f.SetX(f.lMargin)
}

// SetXY defines the abscissa and ordinate of the current position. If the
// passed values are negative, they are relative respectively to the right and
// bottom of the page.
func (f *PDF) SetXY(x, y float64) {
	f.SetY(y)
	f.SetX(x)
}
