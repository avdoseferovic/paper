/*
 * Copyright (c) 2013-2014 Kurt Jung (Gmail: kurt.w.jung)
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package paperpdf

import (
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
func (f *Fpdf) SetDisplayMode(zoomStr, layoutStr string) {
	if f.err != nil {
		return
	}
	if layoutStr == "" {
		layoutStr = "default"
	}
	switch zoomStr {
	case "fullpage", "fullwidth", "real", "default":
		f.zoomMode = zoomStr
	default:
		f.err = fmt.Errorf("incorrect zoom display mode: %s", zoomStr)
		return
	}
	switch layoutStr {
	case "single", "continuous", "two", "default", "SinglePage", "OneColumn",
		"TwoColumnLeft", "TwoColumnRight", "TwoPageLeft", "TwoPageRight":
		f.layoutMode = layoutStr
	default:
		f.err = fmt.Errorf("incorrect layout display mode: %s", layoutStr)
		return
	}
}

// SetProducer defines the producer of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetProducer(producerStr string, isUTF8 bool) {
	if isUTF8 {
		producerStr = utf8toutf16(producerStr)
	}
	f.producer = producerStr
}

// SetTitle defines the title of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetTitle(titleStr string, isUTF8 bool) {
	if isUTF8 {
		titleStr = utf8toutf16(titleStr)
	}
	f.title = titleStr
}

// SetSubject defines the subject of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetSubject(subjectStr string, isUTF8 bool) {
	if isUTF8 {
		subjectStr = utf8toutf16(subjectStr)
	}
	f.subject = subjectStr
}

// SetAuthor defines the author of the document. isUTF8 indicates if the string
// is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetAuthor(authorStr string, isUTF8 bool) {
	if isUTF8 {
		authorStr = utf8toutf16(authorStr)
	}
	f.author = authorStr
}

// SetKeywords defines the keywords of the document. keywordStr is a
// space-delimited string, for example "invoice August". isUTF8 indicates if
// the string is encoded
func (f *Fpdf) SetKeywords(keywordsStr string, isUTF8 bool) {
	if isUTF8 {
		keywordsStr = utf8toutf16(keywordsStr)
	}
	f.keywords = keywordsStr
}

// SetCreator defines the creator of the document. isUTF8 indicates if the
// string is encoded in ISO-8859-1 (false) or UTF-8 (true).
func (f *Fpdf) SetCreator(creatorStr string, isUTF8 bool) {
	if isUTF8 {
		creatorStr = utf8toutf16(creatorStr)
	}
	f.creator = creatorStr
}

// SetXmpMetadata defines XMP metadata that will be embedded with the document.
func (f *Fpdf) SetXmpMetadata(xmpStream []byte) {
	f.xmp = xmpStream
}

// AliasNbPages defines an alias for the total number of pages. It will be
// substituted as the document is closed. An empty string is replaced with the
// string "{nb}".
//
// See the example for AddPage() for a demonstration of this method.
func (f *Fpdf) AliasNbPages(aliasStr string) {
	if aliasStr == "" {
		aliasStr = "{nb}"
	}
	f.aliasNbPagesStr = aliasStr
}

// RTL enables right-to-left mode
func (f *Fpdf) RTL() {
	f.isRTL = true
}

// LTR disables right-to-left mode
func (f *Fpdf) LTR() {
	f.isRTL = false
}

// SetCatalogSort sets a flag that will be used, if true, to consistently order
// the document's internal resource catalogs. This method is typically only
// used for test purposes to facilitate PDF comparison.
func (f *Fpdf) SetCatalogSort(flag bool) {
	f.catalogSort = flag
}

// SetCreationDate fixes the document's internal CreationDate value. By
// default, the time when the document is generated is used for this value.
// This method is typically only used for testing purposes to facilitate PDF
// comparison. Specify a zero-value time to revert to the default behavior.
func (f *Fpdf) SetCreationDate(tm time.Time) {
	f.creationDate = tm
}

// SetModificationDate fixes the document's internal ModDate value.
// See `SetCreationDate` for more details.
func (f *Fpdf) SetModificationDate(tm time.Time) {
	f.modDate = tm
}

// SetJavascript adds Adobe JavaScript to the document.
func (f *Fpdf) SetJavascript(script string) {
	f.javascript = &script
}

// RegisterAlias adds an (alias, replacement) pair to the document so we can
// replace all occurrences of that alias after writing but before the document
// is closed. Functions ExampleFpdf_RegisterAlias() and
// ExampleFpdf_RegisterAlias_utf8() in fpdf_test.go demonstrate this method.
func (f *Fpdf) RegisterAlias(alias, replacement string) {

	f.aliasMap[alias] = replacement
}

func (f *Fpdf) replaceAliases() {
	for mode := 0; mode < 2; mode++ {
		for alias, replacement := range f.aliasMap {
			if mode == 1 {
				alias = utf8toutf16(alias, false)
				replacement = utf8toutf16(replacement, false)
			}
			for n := 1; n <= f.page; n++ {
				s := f.pages[n].String()
				if strings.Contains(s, alias) {
					s = strings.Replace(s, alias, replacement, -1)
					f.pages[n].Truncate(0)
					f.pages[n].WriteString(s)
				}
			}
		}
	}
}

func (f *Fpdf) putjavascript() {
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

func (f *Fpdf) putinfo() {
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

func (f *Fpdf) putcatalog() {
	f.out("/Type /Catalog")
	f.out("/Pages 1 0 R")
	switch f.zoomMode {
	case "fullpage":
		f.out("/OpenAction [3 0 R /Fit]")
	case "fullwidth":
		f.out("/OpenAction [3 0 R /FitH null]")
	case "real":
		f.out("/OpenAction [3 0 R /XYZ null null 1]")
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

	f.layerPutCatalog()

	f.out("/Names <<")

	if f.javascript != nil {
		f.outf("/JavaScript %d 0 R", f.nJs)
	}

	f.outf("/EmbeddedFiles %s", f.getEmbeddedFiles())
	f.out(">>")
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
func (f *Fpdf) SetProtection(actionFlag byte, userPassStr, ownerPassStr string) {
	if f.err != nil {
		return
	}
	f.protect.setProtection(actionFlag, userPassStr, ownerPassStr)
}

// OutputAndClose sends the PDF document to the writer specified by w. This
// method will close both f and w, even if an error is detected and no document
// is produced.
func (f *Fpdf) OutputAndClose(w io.WriteCloser) error {
	f.Output(w)
	w.Close()
	return f.err
}

// OutputFileAndClose creates or truncates the file specified by fileStr and
// writes the PDF document to it. This method will close f and the newly
// written file, even if an error is detected and no document is produced.
//
// Most examples demonstrate the use of this method.
func (f *Fpdf) OutputFileAndClose(fileStr string) error {
	if f.err == nil {
		pdfFile, err := os.Create(fileStr)
		if err == nil {
			f.Output(pdfFile)
			pdfFile.Close()
		} else {
			f.err = err
		}
	}
	return f.err
}

// Output sends the PDF document to the writer specified by w. No output will
// take place if an error has occurred in the document generation process. w
// remains open after this function returns. After returning, f is in a closed
// state and its methods should not be called.
func (f *Fpdf) Output(w io.Writer) error {
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
func (f *Fpdf) escape(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "(", "\\(", -1)
	s = strings.Replace(s, ")", "\\)", -1)
	s = strings.Replace(s, "\r", "\\r", -1)
	return s
}

// textstring formats a text string
func (f *Fpdf) textstring(s string) string {
	if f.protect.encrypted {
		b := []byte(s)
		f.protect.rc4(uint32(f.n), &b)
		s = string(b)
	}
	return "(" + f.escape(s) + ")"
}

// newobj begins a new object
func (f *Fpdf) newobj() {

	f.n++
	for j := len(f.offsets); j <= f.n; j++ {
		f.offsets = append(f.offsets, 0)
	}
	f.offsets[f.n] = f.buffer.Len()
	f.outf("%d 0 obj", f.n)
}

func (f *Fpdf) putstream(b []byte) {

	if f.protect.encrypted {
		f.protect.rc4(uint32(f.n), &b)
	}
	f.out("stream")
	f.out(string(b))
	f.out("endstream")
}

// out; Add a line to the document
func (f *Fpdf) out(s string) {
	if f.state == 2 {
		f.pages[f.page].WriteString(s)
		f.pages[f.page].WriteString("\n")
	} else {
		f.buffer.WriteString(s)
		f.buffer.WriteString("\n")
	}
}

// outbuf adds a buffered line to the document
func (f *Fpdf) outbuf(r io.Reader) {
	if f.state == 2 {
		f.pages[f.page].ReadFrom(r)
		f.pages[f.page].WriteString("\n")
	} else {
		f.buffer.ReadFrom(r)
		f.buffer.WriteString("\n")
	}
}

// RawWriteStr writes a string directly to the PDF generation buffer. This is a
// low-level function that is not required for normal PDF construction. An
// understanding of the PDF specification is needed to use this method
// correctly.
func (f *Fpdf) RawWriteStr(str string) {
	f.out(str)
}

// RawWriteBuf writes the contents of the specified buffer directly to the PDF
// generation buffer. This is a low-level function that is not required for
// normal PDF construction. An understanding of the PDF specification is needed
// to use this method correctly.
func (f *Fpdf) RawWriteBuf(r io.Reader) {
	f.outbuf(r)
}

// outf adds a formatted line to the document
func (f *Fpdf) outf(fmtStr string, args ...any) {
	f.out(sprintf(fmtStr, args...))
}

func (f *Fpdf) putheader() {
	if len(f.blendMap) > 0 && f.pdfVersion < "1.4" {
		f.pdfVersion = "1.4"
	}
	f.outf("%%PDF-%s", f.pdfVersion)
}

func (f *Fpdf) puttrailer() {
	f.outf("/Size %d", f.n+1)
	f.outf("/Root %d 0 R", f.n)
	f.outf("/Info %d 0 R", f.n-1)
	if f.protect.encrypted {
		f.outf("/Encrypt %d 0 R", f.protect.objNum)
		f.out("/ID [()()]")
	}
}

func (f *Fpdf) putxmp() {
	if len(f.xmp) == 0 {
		return
	}
	f.newobj()
	f.outf("<< /Type /Metadata /Subtype /XML /Length %d >>", len(f.xmp))
	f.putstream(f.xmp)
	f.out("endobj")
}

func (f *Fpdf) enddoc() {
	if f.err != nil {
		return
	}
	f.layerEndDoc()
	f.putheader()

	f.putAttachments()
	f.putAnnotationsAttachments()
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
	return
}

func (f *Fpdf) putxobjectdict() {
	{
		var image *ImageInfoType
		var key string
		var keyList []string
		for key = range f.images {
			keyList = append(keyList, key)
		}
		if f.catalogSort {
			sort.SliceStable(keyList, func(i, j int) bool { return f.images[keyList[i]].i < f.images[keyList[j]].i })
		}
		for _, key = range keyList {
			image = f.images[key]
			f.outf("/I%s %d 0 R", image.i, image.n)
		}
	}
	{
		var keyList []string
		var key string
		var tpl Template
		keyList = templateKeyList(f.templates, f.catalogSort)
		for _, key = range keyList {
			tpl = f.templates[key]

			id := tpl.ID()
			if objID, ok := f.templateObjects[id]; ok {
				f.outf("/TPL%s %d 0 R", id, objID)
			}
		}
	}
	{
		for tplName, objID := range f.importedTplObjs {

			f.outf("%s %d 0 R", tplName, f.importedTplIDs[objID])
		}
	}
}

func (f *Fpdf) putresourcedict() {
	f.out("/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]")
	f.out("/Font <<")
	{
		var keyList []string
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

	f.layerPutResourceDict()
	f.spotColorPutResourceDict()
}

func (f *Fpdf) putresources() {
	if f.err != nil {
		return
	}
	f.layerPutLayers()
	f.putBlendModes()
	f.putGradients()
	f.putSpotColors()
	f.putfonts()
	if f.err != nil {
		return
	}
	f.putimages()
	f.putTemplates()
	f.putImportedTemplates()

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
		f.out("/V 1")
		f.out("/R 2")
		f.outf("/O (%s)", f.escape(string(f.protect.oValue)))
		f.outf("/U (%s)", f.escape(string(f.protect.uValue)))
		f.outf("/P %d", f.protect.pValue)
		f.out(">>")
		f.out("endobj")
	}
	return
}

// ImportObjects imports objects from gofpdi into current document
func (f *Fpdf) ImportObjects(objs map[string][]byte) {
	for k, v := range objs {
		f.importedObjs[k] = v
	}
}

// ImportObjPos imports object hash positions from gofpdi
func (f *Fpdf) ImportObjPos(objPos map[string]map[int]string) {
	for k, v := range objPos {
		f.importedObjPos[k] = v
	}
}

// putImportedTemplates writes the imported template objects to the PDF
func (f *Fpdf) putImportedTemplates() {
	nOffset := f.n + 1

	objsIDHash := make([]string, len(f.importedObjs))

	objsIDData := make([][]byte, len(f.importedObjs))

	i := 0
	for k, v := range f.importedObjs {
		objsIDHash[i] = k
		objsIDData[i] = v

		i++
	}

	hashToObjID := make(map[string]int, len(f.importedObjs))
	for i = 0; i < len(objsIDHash); i++ {
		hashToObjID[objsIDHash[i]] = i + nOffset
	}

	for i = 0; i < len(objsIDData); i++ {

		hash := objsIDHash[i]

		for pos, h := range f.importedObjPos[hash] {

			objIDPadded := fmt.Sprintf("%40s", fmt.Sprintf("%d", hashToObjID[h]))

			objIDBytes := []byte(objIDPadded)

			for j := pos; j < pos+40; j++ {
				objsIDData[i][j] = objIDBytes[j-pos]
			}
		}

		f.importedTplIDs[hash] = i + nOffset
	}

	for i = 0; i < len(objsIDData); i++ {
		f.newobj()
		f.out(string(objsIDData[i]))
	}
}

// UseImportedTemplate uses imported template from gofpdi. It draws imported
// PDF page onto page.
func (f *Fpdf) UseImportedTemplate(tplName string, scaleX float64, scaleY float64, tX float64, tY float64) {
	f.outf("q 0 J 1 w 0 j 0 G 0 g q %.4F 0 0 %.4F %.4F %.4F cm %s Do Q Q\n", scaleX*f.k, scaleY*f.k, tX*f.k, (tY+f.h)*f.k, tplName)
}

// ImportTemplates imports gofpdi template names into importedTplObjs for
// inclusion in the procset dictionary
func (f *Fpdf) ImportTemplates(tpls map[string]string) {
	for tplName, tplID := range tpls {
		f.importedTplObjs[tplName] = tplID
	}
}

// GetConversionRatio returns the conversion ratio based on the unit given when
// creating the PDF.
func (f *Fpdf) GetConversionRatio() float64 {
	return f.k
}

// GetXY returns the abscissa and ordinate of the current position.
//
// Note: the value returned for the abscissa will be affected by the current
// cell margin. To account for this, you may need to either add the value
// returned by GetCellMargin() to it or call SetCellMargin(0) to remove the
// cell margin.
func (f *Fpdf) GetXY() (float64, float64) {
	return f.x, f.y
}

// GetX returns the abscissa of the current position.
//
// Note: the value returned will be affected by the current cell margin. To
// account for this, you may need to either add the value returned by
// GetCellMargin() to it or call SetCellMargin(0) to remove the cell margin.
func (f *Fpdf) GetX() float64 {
	return f.x
}

// SetX defines the abscissa of the current position. If the passed value is
// negative, it is relative to the right of the page.
func (f *Fpdf) SetX(x float64) {
	if x >= 0 {
		f.x = x
	} else {
		f.x = f.w + x
	}
}

// GetY returns the ordinate of the current position.
func (f *Fpdf) GetY() float64 {
	return f.y
}

// SetY moves the current abscissa back to the left margin and sets the
// ordinate. If the passed value is negative, it is relative to the bottom of
// the page.
func (f *Fpdf) SetY(y float64) {

	f.x = f.lMargin
	if y >= 0 {
		f.y = y
	} else {
		f.y = f.h + y
	}
}

// SetHomeXY is a convenience method that sets the current position to the left
// and top margins.
func (f *Fpdf) SetHomeXY() {
	f.SetY(f.tMargin)
	f.SetX(f.lMargin)
}

// SetXY defines the abscissa and ordinate of the current position. If the
// passed values are negative, they are relative respectively to the right and
// bottom of the page.
func (f *Fpdf) SetXY(x, y float64) {
	f.SetY(y)
	f.SetX(x)
}
