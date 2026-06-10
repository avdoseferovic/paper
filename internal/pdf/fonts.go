package pdf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// SetFontLocation sets the location in the file system of the font and font
// definition files.
func (f *PDF) SetFontLocation(fontDirStr string) {
	f.fontpath = fontDirStr
}

// SetFontLoader sets a loader used to read font files (.json and .z) from an
// arbitrary source. If a font loader has been specified, it is used to load
// the named font resources when AddFont() is called. If this operation fails,
// an attempt is made to load the resources from the configured font directory
// (see SetFontLocation()).
func (f *PDF) SetFontLoader(loader FontLoader) {
	f.fontLoader = loader
}

// AddFont imports a TrueType, OpenType or Type1 font and makes it available.
// It is necessary to generate a font definition file first with the makefont
// utility. It is not necessary to call this function for the core PDF fonts
// (courier, helvetica, times, zapfdingbats).
//
// The JSON definition file (and the font file itself when embedding) must be
// present in the font directory. If it is not found, the error "Could not
// include font definition file" is set.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// fileStr specifies the base name with ".json" extension of the font
// definition file to be added. The file will be loaded from the font directory
// specified in the call to New() or SetFontLocation().
func (f *PDF) AddFont(familyStr, styleStr, fileStr string) {
	f.addFont(fontFamilyEscape(familyStr), styleStr, fileStr, false)
}

// AddUTF8Font imports a TrueType font with utf-8 symbols and makes it available.
// It is necessary to generate a font definition file first with the makefont
// utility. It is not necessary to call this function for the core PDF fonts
// (courier, helvetica, times, zapfdingbats).
//
// The JSON definition file (and the font file itself when embedding) must be
// present in the font directory. If it is not found, the error "Could not
// include font definition file" is set.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// fileStr specifies the base name with ".json" extension of the font
// definition file to be added. The file will be loaded from the font directory
// specified in the call to New() or SetFontLocation().
func (f *PDF) AddUTF8Font(familyStr, styleStr, fileStr string) {
	f.addFont(fontFamilyEscape(familyStr), styleStr, fileStr, true)
}

func (f *PDF) addFont(familyStr, styleStr, fileStr string, isUTF8 bool) {
	if fileStr == "" {
		fileStr = defaultFontFileName(familyStr, styleStr, isUTF8)
	}
	if isUTF8 {
		f.addUTF8FontFile(familyStr, styleStr, fileStr)
		return
	}
	f.addCoreFontFile(familyStr, styleStr, fileStr)
}

func defaultFontFileName(familyStr, styleStr string, isUTF8 bool) string {
	extension := ".json"
	if isUTF8 {
		extension = ".ttf"
	}
	return strings.ReplaceAll(familyStr, " ", "") + strings.ToLower(styleStr) + extension
}

func (f *PDF) addUTF8FontFile(familyStr, styleStr, fileStr string) {
	fontKey := getFontKey(familyStr, styleStr)
	if _, ok := f.fonts[fontKey]; ok {
		return
	}

	fileStr = path.Join(f.fontpath, fileStr)
	ttfStat, err := os.Stat(fileStr)
	if err != nil {
		f.SetError(err)
		return
	}
	utf8Bytes, err := os.ReadFile(fileStr)
	if err != nil {
		f.SetError(err)
		return
	}
	f.addUTF8FontFromBytes(fontKey, fileStr, ttfStat.Size(), utf8Bytes)
}

func (f *PDF) addCoreFontFile(familyStr, styleStr, fileStr string) {
	if f.addFontFromLoader(familyStr, styleStr, fileStr) {
		return
	}

	file, err := os.Open(path.Join(f.fontpath, fileStr))
	if err != nil {
		f.err = err
		return
	}
	defer file.Close()

	f.AddFontFromReader(familyStr, styleStr, file)
}

func (f *PDF) addFontFromLoader(familyStr, styleStr, fileStr string) bool {
	if f.fontLoader == nil {
		return false
	}
	reader, err := f.fontLoader.Open(fileStr)
	if err != nil {
		return false
	}
	f.AddFontFromReader(familyStr, styleStr, reader)
	if closer, ok := reader.(io.Closer); ok {
		err = closer.Close()
		if err != nil {
			f.SetError(err)
		}
	}
	return true
}

func (f *PDF) addUTF8FontFromBytes(fontKey, fileStr string, originalSize int64, utf8Bytes []byte) {
	reader := fileReader{readerPosition: 0, array: utf8Bytes}
	utf8File := newUTF8Font(&reader)
	err := utf8File.parseFile()
	if err != nil {
		f.SetError(fmt.Errorf("get font metrics: %w", err))
		return
	}

	def := f.newUTF8FontDefinition(fontKey, fileStr, utf8File)
	defID, err := generateFontID(def)
	if err != nil {
		f.SetError(err)
		return
	}
	def.i = defID
	f.fonts[fontKey] = def
	if originalSize > 0 {
		f.fontFiles[fontKey] = fontFileType{
			length1:  originalSize,
			fontType: fontTypeUTF8,
		}
	}
	if fileStr != "" {
		f.fontFiles[fileStr] = fontFileType{
			fontType: fontTypeUTF8,
		}
	}
}

func (f *PDF) newUTF8FontDefinition(fontKey, fileStr string, utf8File *utf8FontFile) fontDefType {
	sbarr := f.initialUTF8Subset()
	def := fontDefType{
		Tp:   utf8FontType(utf8File),
		Name: fontKey,
		Desc: FontDescType{
			Ascent:       utf8File.ascent,
			Descent:      utf8File.descent,
			CapHeight:    utf8File.capHeight,
			Flags:        utf8File.flags,
			FontBBox:     utf8File.bbox,
			ItalicAngle:  utf8File.italicAngle,
			StemV:        utf8File.stemV,
			MissingWidth: round(utf8File.defaultWidth),
		},
		Up:             round(utf8File.underlinePosition),
		Ut:             round(utf8File.underlineThickness),
		Cw:             utf8File.charWidths,
		CwExtra:        utf8File.charWidthExtra,
		usedRunes:      sbarr,
		File:           fileStr,
		utf8File:       utf8File,
		runeToCID:      make(map[int]int),
		hasColorGlyphs: utf8File.hasColorGlyphs,
	}
	for cid, r := range sbarr {
		def.runeToCID[r] = cid
	}
	return def
}

func (f *PDF) initialUTF8Subset() map[int]int {
	if f.aliasNbPagesStr == "" {
		return makeSubsetRange(57)
	}
	return makeSubsetRange(32)
}

func makeSubsetRange(end int) map[int]int {
	answer := make(map[int]int)
	for i := range end {
		answer[i] = 0
	}
	return answer
}

// AddFontFromBytes imports a TrueType, OpenType or Type1 font from static
// bytes within the executable and makes it available for use in the generated
// document.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// jsonFileBytes contain all bytes of JSON file.
//
// zFileBytes contain all bytes of Z file.
func (f *PDF) AddFontFromBytes(familyStr, styleStr string, jsonFileBytes, zFileBytes []byte) {
	f.addFontFromBytes(fontFamilyEscape(familyStr), styleStr, jsonFileBytes, zFileBytes, nil)
}

// AddUTF8FontFromBytes  imports a TrueType font with utf-8 symbols from static
// bytes within the executable and makes it available for use in the generated
// document.
//
// family specifies the font family. The name can be chosen arbitrarily. If it
// is a standard family name, it will override the corresponding font. This
// string is used to subsequently set the font with the SetFont method.
//
// style specifies the font style. Acceptable values are (case insensitive) the
// empty string for regular style, "B" for bold, "I" for italic, or "BI" or
// "IB" for bold and italic combined.
//
// jsonFileBytes contain all bytes of JSON file.
//
// zFileBytes contain all bytes of Z file.
func (f *PDF) AddUTF8FontFromBytes(familyStr, styleStr string, utf8Bytes []byte) {
	f.addFontFromBytes(fontFamilyEscape(familyStr), styleStr, nil, nil, utf8Bytes)
}

func (f *PDF) addFontFromBytes(familyStr, styleStr string, jsonFileBytes, zFileBytes, utf8Bytes []byte) {
	if f.err != nil {
		return
	}

	fontkey := getFontKey(familyStr, styleStr)
	if _, ok := f.fonts[fontkey]; ok {
		return
	}

	if utf8Bytes != nil {
		f.addUTF8FontFromBytes(fontkey, "", 0, utf8Bytes)
		return
	}

	f.addFontDefinitionFromBytes(fontkey, jsonFileBytes, zFileBytes)
}

func (f *PDF) addFontDefinitionFromBytes(fontKey string, jsonFileBytes, zFileBytes []byte) {
	var info fontDefType
	err := json.Unmarshal(jsonFileBytes, &info)
	if err != nil {
		f.err = err
		return
	}

	fontID, err := generateFontID(info)
	if err != nil {
		f.err = err
		return
	}
	info.i = fontID
	f.registerFontDiff(&info)
	f.registerEmbeddedFontFile(info, zFileBytes)
	f.fonts[fontKey] = info
}

func (f *PDF) registerFontDiff(info *fontDefType) {
	if len(info.Diff) == 0 {
		return
	}

	n := -1
	for j, str := range f.diffs {
		if str == info.Diff {
			n = j + 1
			break
		}
	}
	if n < 0 {
		f.diffs = append(f.diffs, info.Diff)
		n = len(f.diffs)
	}
	info.DiffN = n
}

func (f *PDF) registerEmbeddedFontFile(info fontDefType, zFileBytes []byte) {
	if len(info.File) == 0 {
		return
	}

	fontFile := fontFileType{
		length1:  int64(info.Size1),
		length2:  int64(info.Size2),
		embedded: true,
		content:  zFileBytes,
	}
	if info.Tp == fontTypeTrueType {
		fontFile.length1 = int64(info.OriginalSize)
		fontFile.length2 = 0
	}
	f.fontFiles[info.File] = fontFile
}

func utf8FontType(utf8File *utf8FontFile) string {
	if utf8File != nil && utf8File.hasBitmapGlyphs && !utf8File.hasOutlineTables() {
		return fontTypeUTF8Bitmap
	}
	return fontTypeUTF8
}

// getFontKey is used by AddFontFromReader and GetFontDesc
func getFontKey(familyStr, styleStr string) string {
	familyStr = strings.ToLower(familyStr)
	styleStr = strings.ToUpper(styleStr)
	if styleStr == "IB" {
		styleStr = "BI"
	}
	return familyStr + styleStr
}

// AddFontFromReader imports a TrueType, OpenType or Type1 font and makes it
// available using a reader that satisifies the io.Reader interface. See
// AddFont for details about familyStr and styleStr.
func (f *PDF) AddFontFromReader(familyStr, styleStr string, r io.Reader) {
	if f.err != nil {
		return
	}

	familyStr = fontFamilyEscape(familyStr)
	var ok bool
	fontkey := getFontKey(familyStr, styleStr)
	_, ok = f.fonts[fontkey]
	if ok {
		return
	}
	info := f.loadfont(r)
	if f.err != nil {
		return
	}
	if len(info.Diff) > 0 {
		n := -1
		for j, str := range f.diffs {
			if str == info.Diff {
				n = j + 1
				break
			}
		}
		if n < 0 {
			f.diffs = append(f.diffs, info.Diff)
			n = len(f.diffs)
		}
		info.DiffN = n
	}

	if len(info.File) > 0 {
		if info.Tp == fontTypeTrueType {
			f.fontFiles[info.File] = fontFileType{length1: int64(info.OriginalSize)}
		} else {
			f.fontFiles[info.File] = fontFileType{length1: int64(info.Size1), length2: int64(info.Size2)}
		}
	}
	f.fonts[fontkey] = info
}

// GetFontDesc returns the font descriptor, which can be used for
// example to find the baseline of a font. If familyStr is empty
// current font descriptor will be returned.
// See FontDescType for documentation about the font descriptor.
// See AddFont for details about familyStr and styleStr.
func (f *PDF) GetFontDesc(familyStr, styleStr string) FontDescType {
	if familyStr == "" {
		return f.currentFont.Desc
	}
	return f.fonts[getFontKey(fontFamilyEscape(familyStr), styleStr)].Desc
}

// SetFont sets the font used to print character strings. It is mandatory to
// call this method at least once before printing text or the resulting
// document will not be valid.
//
// The font can be either a standard one or a font added via the AddFont()
// method or AddFontFromReader() method. Standard fonts use the Windows
// encoding cp1252 (Western Europe).
//
// The method can be called before the first page is created and the font is
// kept from page to page. If you just wish to change the current font size, it
// is simpler to call SetFontSize().
//
// Note: the font definition file must be accessible. An error is set if the
// file cannot be read.
//
// familyStr specifies the font family. It can be either a name defined by
// AddFont(), AddFontFromReader() or one of the standard families (case
// insensitive): "Courier" for fixed-width, "Helvetica" or "Arial" for sans
// serif, "Times" for serif, "Symbol" or "ZapfDingbats" for symbolic.
//
// styleStr can be "B" (bold), "I" (italic), "U" (underscore), "S" (strike-out)
// or any combination. The default value (specified with an empty string) is
// regular. Bold and italic styles do not apply to Symbol and ZapfDingbats.
//
// size is the font size measured in points. The default value is the current
// size. If no size has been specified since the beginning of the document, the
// value taken is 12.
func (f *PDF) SetFont(familyStr, styleStr string, size float64) {
	if f.err != nil {
		return
	}

	familyStr = f.normalizedFontFamily(familyStr)
	styleStr = f.normalizedFontStyle(styleStr)
	if size == 0.0 {
		size = f.fontSizePt
	}

	fontKey := familyStr + styleStr
	if _, ok := f.fonts[fontKey]; !ok {
		var loaded bool
		familyStr, styleStr, fontKey, loaded = f.loadCoreFont(familyStr, styleStr)
		if !loaded {
			return
		}
	}

	f.fontFamily = familyStr
	f.fontStyle = styleStr
	f.fontSizePt = size
	f.fontSize = size / f.k
	f.currentFont = f.fonts[fontKey]
	if f.currentFont.Tp == fontTypeUTF8 || f.currentFont.Tp == fontTypeUTF8Bitmap {
		f.isCurrentUTF8 = true
	} else {
		f.isCurrentUTF8 = false
	}
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

func (f *PDF) normalizedFontFamily(familyStr string) string {
	familyStr = fontFamilyEscape(familyStr)
	if familyStr == "" {
		return f.fontFamily
	}
	return strings.ToLower(familyStr)
}

func (f *PDF) normalizedFontStyle(styleStr string) string {
	styleStr = strings.ToUpper(styleStr)
	f.underline = strings.Contains(styleStr, "U")
	styleStr = strings.ReplaceAll(styleStr, "U", "")
	f.strikeout = strings.Contains(styleStr, "S")
	styleStr = strings.ReplaceAll(styleStr, "S", "")
	if styleStr == "IB" {
		return "BI"
	}
	return styleStr
}

func (f *PDF) loadCoreFont(familyStr, styleStr string) (string, string, string, bool) {
	if familyStr == "arial" {
		familyStr = "helvetica"
	}
	if _, ok := f.coreFonts[familyStr]; !ok {
		f.err = staticErrorf(errUndefinedFont, "%s %s", familyStr, styleStr)
		return familyStr, styleStr, familyStr + styleStr, false
	}
	if familyStr == "symbol" {
		familyStr = "zapfdingbats"
	}
	if familyStr == "zapfdingbats" {
		styleStr = ""
	}

	fontKey := familyStr + styleStr
	if _, ok := f.fonts[fontKey]; ok {
		return familyStr, styleStr, fontKey, true
	}
	rdr := f.coreFontReader(familyStr, styleStr)
	if f.err == nil {
		f.AddFontFromReader(familyStr, styleStr, rdr)
	}
	return familyStr, styleStr, fontKey, f.err == nil
}

// SetFontStyle sets the style of the current font. See also SetFont()
func (f *PDF) SetFontStyle(styleStr string) {
	f.SetFont(f.fontFamily, styleStr, f.fontSizePt)
}

// SetFontSize defines the size of the current font. Size is specified in
// points (1/ 72 inch). See also SetFontUnitSize().
func (f *PDF) SetFontSize(size float64) {
	f.fontSizePt = size
	f.fontSize = size / f.k
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

// SetFontUnitSize defines the size of the current font. Size is specified in
// the unit of measure specified in New(). See also SetFontSize().
func (f *PDF) SetFontUnitSize(size float64) {
	f.fontSizePt = size * f.k
	f.fontSize = size
	if f.page > 0 {
		f.outf("BT /F%s %.2f Tf ET", f.currentFont.i, f.fontSizePt)
	}
}

// GetFontSize returns the size of the current font in points followed by the
// size in the unit of measure specified in New(). The second value can be used
// as a line height value in drawing operations.
func (f *PDF) GetFontSize() (float64, float64) {
	return f.fontSizePt, f.fontSize
}

// SetUnderlineThickness accepts a multiplier for adjusting the text underline
// thickness, defaulting to 1. See SetUnderlineThickness example.
func (f *PDF) SetUnderlineThickness(thickness float64) {
	f.userUnderlineThickness = thickness
}

// Underline text
func (f *PDF) dounderline(x, y float64, txt string) string {
	up := float64(f.currentFont.Up)
	ut := float64(f.currentFont.Ut) * f.userUnderlineThickness
	w := f.GetStringWidth(txt) + f.ws*float64(blankCount(txt))
	return sprintf("%.2f %.2f %.2f %.2f re f", x*f.k,
		(f.h-(y-up/1000*f.fontSize))*f.k, w*f.k, -ut/1000*f.fontSizePt)
}

func (f *PDF) dostrikeout(x, y float64, txt string) string {
	up := float64(f.currentFont.Up)
	ut := float64(f.currentFont.Ut)
	w := f.GetStringWidth(txt) + f.ws*float64(blankCount(txt))
	return sprintf("%.2f %.2f %.2f %.2f re f", x*f.k,
		(f.h-(y+4*up/1000*f.fontSize))*f.k, w*f.k, -ut/1000*f.fontSizePt)
}

// Load a font definition file from the given Reader
func (f *PDF) loadfont(r io.Reader) fontDefType {
	var def fontDefType
	if f.err != nil {
		return def
	}
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		f.err = fmt.Errorf("read font definition: %w", err)
		return def
	}
	err = json.Unmarshal(buf.Bytes(), &def)
	if err != nil {
		f.err = fmt.Errorf("decode font definition: %w", err)
		return def
	}

	var fontID string
	fontID, err = generateFontID(def)
	if err != nil {
		f.err = err
	}
	def.i = fontID

	return def
}

func (f *PDF) putfonts() {
	if f.err != nil {
		return
	}
	nf := f.n
	f.putFontEncodingObjects()
	f.putFontFileObjects()
	if f.err != nil {
		return
	}
	f.putFontObjects(nf)
}

func (f *PDF) putFontEncodingObjects() {
	for _, diff := range f.diffs {
		f.newobj()
		f.outf("<</Type /Encoding /BaseEncoding /WinAnsiEncoding /Differences [%s]>>", diff)
		f.out("endobj")
	}
}

func (f *PDF) putFontFileObjects() {
	for _, file := range f.sortedFontFileKeys() {
		info := f.fontFiles[file]
		if info.fontType == fontTypeUTF8 {
			continue
		}
		f.putFontFileObject(file, info)
		if f.err != nil {
			return
		}
	}
}

func (f *PDF) sortedFontFileKeys() []string {
	fileList := make([]string, 0, len(f.fontFiles))
	for file := range f.fontFiles {
		fileList = append(fileList, file)
	}
	if f.catalogSort {
		sort.SliceStable(fileList, func(i, j int) bool { return fileList[i] < fileList[j] })
	}
	return fileList
}

func (f *PDF) putFontFileObject(file string, info fontFileType) {
	f.newobj()
	info.n = f.n
	f.fontFiles[file] = info

	font, err := f.fontFileContent(file, info)
	if err != nil {
		f.err = err
		return
	}
	compressed := strings.HasSuffix(file, ".z")
	if !compressed && info.length2 > 0 {
		buf := font[6:info.length1]
		buf = append(buf, font[6+info.length1+6:info.length2]...)
		font = buf
	}
	stream := f.encryptedStream(font)
	if f.err != nil {
		return
	}
	f.outf("<</Length %d", len(stream))
	if compressed {
		f.out("/Filter /FlateDecode")
	}
	f.outf("/Length1 %d", info.length1)
	if info.length2 > 0 {
		f.outf("/Length2 %d /Length3 0", info.length2)
	}
	f.out(">>")
	f.putstream(stream)
	f.out("endobj")
}

func (f *PDF) fontFileContent(file string, info fontFileType) ([]byte, error) {
	if info.embedded {
		return info.content, nil
	}
	return f.loadFontFile(file)
}

func (f *PDF) putFontObjects(nf int) {
	for _, key := range f.sortedFontKeys() {
		font := f.fonts[key]
		font.N = f.n + 1
		f.fonts[key] = font
		f.putFontObject(nf, font)
		if f.err != nil {
			return
		}
	}
}

func (f *PDF) sortedFontKeys() []string {
	keyList := make([]string, 0, len(f.fonts))
	for key := range f.fonts {
		keyList = append(keyList, key)
	}
	if f.catalogSort {
		sort.SliceStable(keyList, func(i, j int) bool { return keyList[i] < keyList[j] })
	}
	return keyList
}

func (f *PDF) putFontObject(nf int, font fontDefType) {
	switch font.Tp {
	case "Core":
		f.putCoreFontObject(font)
	case "Type1", fontTypeTrueType:
		f.putType1OrTrueTypeFontObject(nf, font)
	case fontTypeUTF8, fontTypeUTF8Bitmap:
		f.putUTF8FontObject(font)
	default:
		f.err = fmt.Errorf("%w: %s", errUnsupportedFontType, font.Tp)
	}
}

func (f *PDF) putCoreFontObject(font fontDefType) {
	f.newobj()
	f.out("<</Type /Font")
	f.outf("/BaseFont /%s", font.Name)
	f.out("/Subtype /Type1")
	if font.Name != "Symbol" && font.Name != "ZapfDingbats" {
		f.out("/Encoding /WinAnsiEncoding")
	}
	f.out(">>")
	f.out("endobj")
}

func (f *PDF) putType1OrTrueTypeFontObject(nf int, font fontDefType) {
	f.newobj()
	f.out("<</Type /Font")
	f.outf("/BaseFont /%s", font.Name)
	f.outf("/Subtype /%s", font.Tp)
	f.out("/FirstChar 32 /LastChar 255")
	f.outf("/Widths %d 0 R", f.n+1)
	f.outf("/FontDescriptor %d 0 R", f.n+2)
	if font.DiffN > 0 {
		f.outf("/Encoding %d 0 R", nf+font.DiffN)
	} else {
		f.out("/Encoding /WinAnsiEncoding")
	}
	f.out(">>")
	f.out("endobj")

	f.putFontWidthsObject(font)
	f.putFontDescriptorObject(font)
}

func (f *PDF) putFontWidthsObject(font fontDefType) {
	f.newobj()
	var s fmtBuffer
	s.WriteString("[")
	for j := 32; j < 256; j++ {
		s.printf("%d ", font.Cw[j])
	}
	s.WriteString("]")
	f.out(s.String())
	f.out("endobj")
}

func (f *PDF) putFontDescriptorObject(font fontDefType) {
	f.newobj()
	var s fmtBuffer
	s.printf("<</Type /FontDescriptor /FontName /%s ", font.Name)
	s.printf("/Ascent %d ", font.Desc.Ascent)
	s.printf("/Descent %d ", font.Desc.Descent)
	s.printf("/CapHeight %d ", font.Desc.CapHeight)
	s.printf("/Flags %d ", font.Desc.Flags)
	s.printf("/FontBBox [%d %d %d %d] ", font.Desc.FontBBox.Xmin, font.Desc.FontBBox.Ymin,
		font.Desc.FontBBox.Xmax, font.Desc.FontBBox.Ymax)
	s.printf("/ItalicAngle %d ", font.Desc.ItalicAngle)
	s.printf("/StemV %d ", font.Desc.StemV)
	s.printf("/MissingWidth %d ", font.Desc.MissingWidth)
	s.printf("/FontFile%s %d 0 R>>", fontFileSuffix(font.Tp), f.fontFiles[font.File].n)
	f.out(s.String())
	f.out("endobj")
}

func fontFileSuffix(fontType string) string {
	if fontType == "Type1" {
		return ""
	}
	return "2"
}

func (f *PDF) putUTF8FontObject(font fontDefType) {
	if font.Tp == fontTypeUTF8Bitmap {
		f.newobj()
		f.out("<</Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding>>")
		f.out("endobj")
		return
	}

	fontName := "utf8" + font.Name
	utf8FontStream := f.generateUTF8FontStream(font)
	if f.err != nil {
		return
	}
	utf8FontSize := len(utf8FontStream)
	compressedFontStream := sliceCompress(utf8FontStream)
	codeSignDictionary := font.utf8File.codeSymbolDictionary
	delete(codeSignDictionary, 0)

	f.putUTF8Type0Object(fontName)
	f.putUTF8CIDFontObject(fontName, font)
	f.putUTF8ToUnicodeObject(font.usedRunes)
	f.putUTF8CIDSystemInfoObject()
	f.putUTF8DescriptorObject(fontName, font)
	f.putUTF8CIDToGIDMapObject(codeSignDictionary)
	if f.err != nil {
		return
	}
	f.putUTF8FontStreamObject(compressedFontStream, utf8FontSize)
}

func (f *PDF) generateUTF8FontStream(font fontDefType) []byte {
	usedRunes := font.usedRunes
	delete(usedRunes, 0)
	utf8FontStream := font.utf8File.generateCutFont(usedRunes)
	if font.utf8File.err != nil {
		f.SetError(fmt.Errorf("generate UTF-8 font subset: %w", font.utf8File.err))
		return nil
	}
	if utf8FontStream == nil {
		f.SetErrorf("generate UTF-8 font subset: empty font stream")
		return nil
	}
	return utf8FontStream
}

func (f *PDF) putUTF8Type0Object(fontName string) {
	f.newobj()
	f.out(fmt.Sprintf(
		"<</Type /Font\n/Subtype /Type0\n/BaseFont /%s\n/Encoding /Identity-H\n"+
			"/DescendantFonts [%d 0 R]\n/ToUnicode %d 0 R>>\nendobj",
		fontName, f.n+1, f.n+2,
	))
}

func (f *PDF) putUTF8CIDFontObject(fontName string, font fontDefType) {
	f.newobj()
	f.out("<</Type /Font\n/Subtype /CIDFontType2\n/BaseFont /" + fontName + "\n" +
		"/CIDSystemInfo " + strconv.Itoa(f.n+2) + " 0 R\n/FontDescriptor " + strconv.Itoa(f.n+3) + " 0 R")
	if font.Desc.MissingWidth != 0 {
		f.out("/DW " + strconv.Itoa(font.Desc.MissingWidth) + "")
	}
	f.generateCIDFontMap(&font, font.utf8File.lastRune)
	f.out("/CIDToGIDMap " + strconv.Itoa(f.n+4) + " 0 R>>")
	f.out("endobj")
}

func (f *PDF) putUTF8ToUnicodeObject(usedRunes map[int]int) {
	toUnicodeMap := buildToUnicodeCMap(usedRunes)
	f.newobj()
	stream := f.encryptedStream([]byte(toUnicodeMap))
	if f.err != nil {
		return
	}
	f.out("<</Length " + strconv.Itoa(len(stream)) + ">>")
	f.putstream(stream)
	f.out("endobj")
}

func (f *PDF) putUTF8CIDSystemInfoObject() {
	f.newobj()
	f.out("<</Registry (Adobe)\n/Ordering (UCS)\n/Supplement 0>>")
	f.out("endobj")
}

func (f *PDF) putUTF8DescriptorObject(fontName string, font fontDefType) {
	f.newobj()
	var s fmtBuffer
	s.printf("<</Type /FontDescriptor /FontName /%s\n /Ascent %d", fontName, font.Desc.Ascent)
	s.printf(" /Descent %d", font.Desc.Descent)
	s.printf(" /CapHeight %d", font.Desc.CapHeight)
	v := font.Desc.Flags
	v |= 4
	v &^= 32
	s.printf(" /Flags %d", v)
	s.printf("/FontBBox [%d %d %d %d] ", font.Desc.FontBBox.Xmin, font.Desc.FontBBox.Ymin,
		font.Desc.FontBBox.Xmax, font.Desc.FontBBox.Ymax)
	s.printf(" /ItalicAngle %d", font.Desc.ItalicAngle)
	s.printf(" /StemV %d", font.Desc.StemV)
	s.printf(" /MissingWidth %d", font.Desc.MissingWidth)
	s.printf("/FontFile2 %d 0 R", f.n+2)
	s.printf(">>")
	f.out(s.String())
	f.out("endobj")
}

func (f *PDF) putUTF8CIDToGIDMapObject(codeSignDictionary map[int]int) {
	cidToGidMap := make([]byte, 256*256*2)
	for cc, glyph := range codeSignDictionary {
		glyphID, ok := checkedUint16(glyph)
		if !ok {
			f.SetErrorf("glyph id out of range: %d", glyph)
			return
		}
		cidToGidMap[cc*2] = byte(glyphID >> 8) // #nosec G115 -- high byte of a uint16 glyph id.
		cidToGidMap[cc*2+1] = byte(glyphID)    // #nosec G115 -- low byte of a uint16 glyph id.
	}

	cidToGidMap = sliceCompress(cidToGidMap)
	f.newobj()
	stream := f.encryptedStream(cidToGidMap)
	if f.err != nil {
		return
	}
	f.out("<</Length " + strconv.Itoa(len(stream)) + "/Filter /FlateDecode>>")
	f.putstream(stream)
	f.out("endobj")
}

func (f *PDF) putUTF8FontStreamObject(compressedFontStream []byte, utf8FontSize int) {
	f.newobj()
	stream := f.encryptedStream(compressedFontStream)
	if f.err != nil {
		return
	}
	f.out("<</Length " + strconv.Itoa(len(stream)))
	f.out("/Filter /FlateDecode")
	f.out("/Length1 " + strconv.Itoa(utf8FontSize))
	f.out(">>")
	f.putstream(stream)
	f.out("endobj")
}

func (f *PDF) generateCIDFontMap(font *fontDefType, lastRune int) {
	f.out("/W [" + formatCIDWidthRuns(font, lastRune) + " ]")
}

func (f *PDF) loadFontFile(name string) ([]byte, error) {
	data, ok, err := f.loadFontFileFromLoader(name)
	if ok || err != nil {
		return data, err
	}
	data, err = os.ReadFile(path.Join(f.fontpath, name))
	if err != nil {
		return nil, fmt.Errorf("read font file %q: %w", name, err)
	}
	return data, nil
}

func (f *PDF) loadFontFileFromLoader(name string) ([]byte, bool, error) {
	if f.fontLoader == nil {
		return nil, false, nil
	}
	reader, err := f.fontLoader.Open(name)
	if err == nil {
		return readFontLoaderFile(name, reader)
	}
	return nil, false, nil
}

func readFontLoaderFile(name string, reader io.Reader) ([]byte, bool, error) {
	data, readErr := io.ReadAll(reader)
	closeErr := closeFontReader(name, reader)
	if readErr != nil {
		return nil, true, fmt.Errorf("read font file %q: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, true, closeErr
	}
	return data, true, nil
}

func closeFontReader(name string, reader io.Reader) error {
	closer, ok := reader.(io.Closer)
	if !ok {
		return nil
	}
	err := closer.Close()
	if err != nil {
		return fmt.Errorf("close font file %q: %w", name, err)
	}
	return nil
}

func buildToUnicodeCMap(usedRunes map[int]int) string {
	var b fmtBuffer
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo\n<</Registry (Adobe)\n/Ordering (UCS)\n/Supplement 0\n>> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")

	cids := keySortInt(usedRunes)
	const chunkSize = 100
	for start := 0; start < len(cids); {
		end := min(start+chunkSize, len(cids))
		count := 0
		for _, cid := range cids[start:end] {
			if cid > 0 && cid <= 0xFFFF && usedRunes[cid] > 0 {
				count++
			}
		}
		if count > 0 {
			b.printf("%d beginbfchar\n", count)
			for _, cid := range cids[start:end] {
				r := usedRunes[cid]
				if cid <= 0 || cid > 0xFFFF || r <= 0 {
					continue
				}
				b.printf("<%04X> <%s>\n", cid, utf16Hex(rune(r))) // #nosec G115 -- r is a Unicode scalar value.
			}
			b.WriteString("endbfchar\n")
		}
		start = end
	}

	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend")
	return b.String()
}

func utf16Hex(r rune) string {
	encoded := []byte(utf8toutf16(string(r), false))
	var b strings.Builder
	for _, c := range encoded {
		fmt.Fprintf(&b, "%02X", c)
	}
	return b.String()
}

type cidWidthRun struct {
	start    int
	widths   []int
	interval bool
}

type cidWidthRuns []cidWidthRun

func formatCIDWidthRuns(font *fontDefType, lastRune int) string {
	runs := mergeCIDWidthRuns(buildCIDWidthRuns(font, lastRune))

	var b fmtBuffer
	for _, run := range runs {
		if run.hasSingleWidth() {
			b.printf(" %d %d %d", run.start, run.end(), run.widths[0])
			continue
		}
		b.printf(" %d [ %s ]\n", run.start, joinCIDWidths(run.widths))
	}
	return b.String()
}

func buildCIDWidthRuns(font *fontDefType, lastRune int) cidWidthRuns {
	if font == nil || lastRune < 1 {
		return nil
	}

	runs := make(cidWidthRuns, 0)
	prevCID := -2
	prevWidth := -1
	interval := false

	for cid := 1; cid <= lastRune; cid++ {
		runa := cid
		if used, ok := font.usedRunes[cid]; ok {
			runa = used
		}
		width := fontWidth(font, runa)
		if width == 0 {
			continue
		}
		if width == 65535 {
			width = 0
		}
		if used, ok := font.usedRunes[cid]; cid > 255 && (!ok || used == 0) {
			continue
		}

		if cid == prevCID+1 && len(runs) > 0 {
			current := &runs[len(runs)-1]
			switch {
			case width == prevWidth:
				if width == current.widths[0] {
					current.widths = append(current.widths, width)
				} else {
					current.widths = current.widths[:len(current.widths)-1]
					runs = append(runs, cidWidthRun{
						start:  prevCID,
						widths: []int{prevWidth, width},
					})
					current = &runs[len(runs)-1]
				}
				current.interval = true
				interval = true
			case interval:
				runs = append(runs, cidWidthRun{
					start:  cid,
					widths: []int{width},
				})
				interval = false
			default:
				current.widths = append(current.widths, width)
			}
		} else {
			runs = append(runs, cidWidthRun{
				start:  cid,
				widths: []int{width},
			})
			interval = false
		}

		prevCID = cid
		prevWidth = width
	}

	return runs
}

func fontWidth(font *fontDefType, runa int) int {
	if font == nil {
		return 0
	}
	if runa >= 0 && runa < len(font.Cw) {
		return font.Cw[runa]
	}
	return font.CwExtra[runa]
}

func mergeCIDWidthRuns(runs cidWidthRuns) cidWidthRuns {
	merged := make(cidWidthRuns, 0, len(runs))
	nextStart := -1
	previousWasLongInterval := false

	for _, run := range runs {
		logicalLen := run.logicalLen()
		if run.start == nextStart && !previousWasLongInterval && (!run.interval || logicalLen < 4) {
			merged[len(merged)-1].widths = append(merged[len(merged)-1].widths, run.widths...)
		} else {
			merged = append(merged, run)
		}

		nextStart = run.start + logicalLen
		if run.interval {
			previousWasLongInterval = logicalLen > 3
			nextStart--
		} else {
			previousWasLongInterval = false
		}
	}

	return merged
}

func (run cidWidthRun) logicalLen() int {
	if run.interval {
		return len(run.widths) + 1
	}
	return len(run.widths)
}

func (run cidWidthRun) end() int {
	return run.start + len(run.widths) - 1
}

func (run cidWidthRun) hasSingleWidth() bool {
	for _, width := range run.widths[1:] {
		if width != run.widths[0] {
			return false
		}
	}
	return len(run.widths) > 0
}

func joinCIDWidths(widths []int) string {
	var b strings.Builder
	for i, width := range widths {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(width))
	}
	return b.String()
}

type fileReader struct {
	readerPosition int64
	array          []byte
	err            error
}

func (fr *fileReader) Read(s int) []byte {
	if s < 0 {
		fr.setErrorf("invalid font read length %d", s)
		return nil
	}

	out := make([]byte, s)
	if s == 0 || fr.err != nil {
		return out
	}

	start := fr.readerPosition
	end := start + int64(s)
	if start < 0 || start > int64(len(fr.array)) || end < start {
		fr.setErrorf("invalid font read offset %d", start)
		return out
	}

	if end > int64(len(fr.array)) {
		fr.setErrorf("unexpected EOF reading font data")
		end = int64(len(fr.array))
	}
	copy(out, fr.array[start:end])
	fr.readerPosition = end
	return out
}

func (fr *fileReader) seek(shift int64, flag int) (int64, error) {
	if fr.err != nil {
		return fr.readerPosition, fr.err
	}

	target := fr.readerPosition
	switch flag {
	case 0:
		target = shift
	case 1:
		target += shift
	case 2:
		target = int64(len(fr.array)) - shift
	default:
		fr.setErrorf("invalid font seek mode %d", flag)
		return fr.readerPosition, fr.err
	}

	if target < 0 || target > int64(len(fr.array)) {
		fr.setErrorf("invalid font seek offset %d", target)
		return fr.readerPosition, fr.err
	}

	fr.readerPosition = target
	return fr.readerPosition, nil
}

func (fr *fileReader) setErrorf(format string, args ...any) {
	if fr.err == nil {
		fr.err = staticErrorf(errFontReader, format, args...)
	}
}

func unpackUint16Array(data []byte) []int {
	answer := make([]int, 0, len(data)/2+1)
	answer = append(answer, 0)
	r := bytes.NewReader(data)
	bs := make([]byte, 2)
	var e error
	var c int
	c, e = r.Read(bs)
	for e == nil && c > 0 {
		answer = append(answer, int(binary.BigEndian.Uint16(bs)))
		c, e = r.Read(bs)
	}
	return answer
}

func unpackUint32Array(data []byte) []int {
	answer := make([]int, 0, len(data)/4+1)
	answer = append(answer, 0)
	r := bytes.NewReader(data)
	bs := make([]byte, 4)
	var e error
	var c int
	c, e = r.Read(bs)
	for e == nil && c > 0 {
		answer = append(answer, int(binary.BigEndian.Uint32(bs)))
		c, e = r.Read(bs)
	}
	return answer
}

func unpackUint16(data []byte) int {
	return int(binary.BigEndian.Uint16(data))
}

func packHeader(n uint32, n1, n2, n3, n4 int) []byte {
	answer := make([]byte, 0, 12)
	bs4 := make([]byte, 4)
	binary.BigEndian.PutUint32(bs4, n)
	answer = append(answer, bs4...)
	bs := make([]byte, 2)
	putUint16(bs, n1)
	answer = append(answer, bs...)
	putUint16(bs, n2)
	answer = append(answer, bs...)
	putUint16(bs, n3)
	answer = append(answer, bs...)
	putUint16(bs, n4)
	answer = append(answer, bs...)
	return answer
}

func pack2Uint16(n1, n2 int) []byte {
	answer := make([]byte, 0, 4)
	bs := make([]byte, 2)
	putUint16(bs, n1)
	answer = append(answer, bs...)
	putUint16(bs, n2)
	answer = append(answer, bs...)
	return answer
}

func pack2Uint32(n1, n2 int) []byte {
	answer := make([]byte, 0, 8)
	bs := make([]byte, 4)
	putUint32(bs, n1)
	answer = append(answer, bs...)
	putUint32(bs, n2)
	answer = append(answer, bs...)
	return answer
}

func packUint32(n1 int) []byte {
	bs := make([]byte, 4)
	putUint32(bs, n1)
	return bs
}

func packUint16(n1 int) []byte {
	bs := make([]byte, 2)
	putUint16(bs, n1)
	return bs
}

func putUint16(dst []byte, n int) {
	binary.BigEndian.PutUint16(dst, uint16(n)) // #nosec G115 -- callers pass OpenType uint16 fields.
}

func putUint32(dst []byte, n int) {
	binary.BigEndian.PutUint32(dst, uint32(n)) // #nosec G115 -- callers pass OpenType uint32 fields.
}

func keySortStrings(s map[string][]byte) []string {
	keys := make([]string, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}

func keySortInt(s map[int]int) []int {
	keys := make([]int, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Ints(keys)
	return keys
}

func keySortArrayRangeMap(s map[int][]int) []int {
	keys := make([]int, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Ints(keys)
	return keys
}
