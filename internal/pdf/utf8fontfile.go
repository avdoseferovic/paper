package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// flags
const (
	symbolWords    = 1 << 0
	symbolScale    = 1 << 3
	symbolContinue = 1 << 5
	symbolAllScale = 1 << 6
	symbol2x2      = 1 << 7
)

type colorRecord struct {
	r, g, b, a byte
}

type colorLayerRecord struct {
	glyphID      uint16
	paletteIndex uint16
}

type colorBaseGlyphRecord struct {
	glyphID       uint16
	firstLayerIdx uint16
	numLayers     uint16
}

type colrTable struct {
	version             uint16
	baseGlyphRecords    []colorBaseGlyphRecord
	layerRecords        []colorLayerRecord
	baseGlyphListOffset int
	layerListOffset     int
	clipListOffset      int
}

type cpalTable struct {
	colorRecords []colorRecord
}

type bitmapGlyphLocation struct {
	dataOffset  int
	dataLength  int
	imageFormat int
	ppemX       int
	ppemY       int
	hasMetrics  bool
	width       int
	height      int
	bearingX    int
	bearingY    int
	advance     int
}

type bitmapGlyphImage struct {
	data          []byte
	imageType     string
	width         int
	height        int
	bearingX      int
	bearingY      int
	advance       int
	ppemX         int
	ppemY         int
	originOffsetX int
	originOffsetY int
}

type sbixStrike struct {
	offset int
	ppem   int
	ppi    int
}

const bitmapEmojiResolutionScale = 4.0

type utf8FontFile struct {
	fileReader           *fileReader
	fontOffset           int
	lastRune             int
	tableDescriptions    map[string]*tableDescription
	outTablesData        map[string][]byte
	symbolPosition       []int
	charSymbolDictionary map[int]int
	ascent               int
	descent              int
	fontElementSize      int
	bbox                 fontBoxType
	capHeight            int
	stemV                int
	italicAngle          int
	flags                int
	underlinePosition    float64
	underlineThickness   float64
	charWidths           []int
	charWidthExtra       map[int]int
	defaultWidth         float64
	symbolData           map[int]map[string][]int
	codeSymbolDictionary map[int]int
	err                  error
	colrTable            *colrTable
	cpalTable            *cpalTable
	cbdtGlyphs           map[int][]bitmapGlyphLocation
	sbixStrikes          []sbixStrike
	hasColorGlyphs       bool
	hasBitmapGlyphs      bool
}

type tableDescription struct {
	name     string
	checksum []int
	position int
	size     int
}

func newUTF8Font(reader *fileReader) *utf8FontFile {
	utf := utf8FontFile{
		fileReader: reader,
	}
	return &utf
}

func (utf *utf8FontFile) parseFile() error {
	utf.fileReader.readerPosition = 0
	utf.fileReader.err = nil
	utf.err = nil
	utf.symbolPosition = make([]int, 0)
	utf.charSymbolDictionary = make(map[int]int)
	utf.tableDescriptions = make(map[string]*tableDescription)
	utf.outTablesData = make(map[string][]byte)
	utf.ascent = 0
	utf.descent = 0
	codeType := uint32(utf.readUint32()) // #nosec G115 -- readUint32 returns a four-byte unsigned font tag.
	if utf.err != nil {
		return utf.err
	}
	if codeType == 0x4F54544F {
		return errUnsupportedCFFFont
	}
	if codeType == 0x74746366 {
		err := utf.selectTTCFont()
		if err != nil {
			return err
		}
		codeType = uint32(utf.readUint32()) // #nosec G115 -- readUint32 returns a four-byte unsigned font tag.
	}
	if codeType != 0x00010000 && codeType != 0x74727565 {
		return fmt.Errorf("%w: codeType=%d", errUnexpectedTrueTypeCodeType, codeType)
	}
	utf.generateTableDescriptions()
	if utf.err != nil {
		return utf.err
	}
	utf.parseTables()
	utf.setError(utf.fileReader.err)
	return utf.err
}

func (utf *utf8FontFile) hasOutlineTables() bool {
	return utf.tableDescriptions["glyf"] != nil && utf.tableDescriptions["loca"] != nil
}

func (utf *utf8FontFile) selectTTCFont() error {
	_ = utf.readUint16() // major version
	_ = utf.readUint16() // minor version
	fontCount := utf.readUint32()
	if utf.err != nil {
		return utf.err
	}
	if fontCount < 1 {
		return errTrueTypeCollectionEmpty
	}
	fontOffset := utf.readUint32()
	if utf.err != nil {
		return utf.err
	}
	if fontOffset <= 0 || fontOffset+4 > len(utf.fileReader.array) {
		return errTrueTypeCollectionOffset
	}
	utf.fontOffset = fontOffset
	utf.seek(fontOffset)
	return utf.err
}

func (utf *utf8FontFile) generateTableDescriptions() {
	tablesCount := utf.readUint16()
	_ = utf.readUint16()
	_ = utf.readUint16()
	_ = utf.readUint16()
	if utf.err != nil {
		return
	}
	if 12+(tablesCount*16) > len(utf.fileReader.array) {
		utf.setErrorf("unexpected EOF reading font table directory")
		return
	}
	utf.tableDescriptions = make(map[string]*tableDescription)

	for range tablesCount {
		record := tableDescription{
			name:     utf.readTableName(),
			checksum: []int{utf.readUint16(), utf.readUint16()},
			position: utf.readUint32(),
			size:     utf.readUint32(),
		}
		if utf.err != nil {
			return
		}
		if record.position < 0 || record.size < 0 ||
			record.position > len(utf.fileReader.array) ||
			record.position+record.size > len(utf.fileReader.array) {
			utf.setErrorf("font table %q exceeds font data", record.name)
			return
		}
		utf.tableDescriptions[record.name] = &record
	}
}

func (utf *utf8FontFile) readTableName() string {
	return string(utf.readBytes(4))
}

func (utf *utf8FontFile) readUint16() int {
	s := utf.readBytes(2)
	return (int(s[0]) << 8) + int(s[1])
}

func (utf *utf8FontFile) readCheckedUint16(label string) (uint16, bool) {
	value := utf.readUint16()
	result, ok := checkedUint16(value)
	if !ok {
		utf.setErrorf("%s value %d is out of uint16 range", label, value)
		return 0, false
	}
	return result, true
}

func (utf *utf8FontFile) readByte() int {
	s := utf.readBytes(1)
	return int(s[0])
}

func (utf *utf8FontFile) readInt8Bits(label string) (int, bool) {
	value := utf.readByte()
	result, ok := checkedByte(value)
	if !ok {
		utf.setErrorf("%s value %d is out of byte range", label, value)
		return 0, false
	}
	return signedByteValue(result), true
}

func (utf *utf8FontFile) readOffset24() int {
	s := utf.readBytes(3)
	return (int(s[0]) << 16) + (int(s[1]) << 8) + int(s[2])
}

func (utf *utf8FontFile) readUint32() int {
	s := utf.readBytes(4)
	return (int(s[0]) * 16777216) + (int(s[1]) << 16) + (int(s[2]) << 8) + int(s[3])
}

func (utf *utf8FontFile) calcInt32(x, y []int) []int {
	answer := make([]int, 2)
	if y[1] > x[1] {
		x[1] += 1 << 16
		x[0]++
	}
	answer[1] = x[1] - y[1]
	if y[0] > x[0] {
		x[0] += 1 << 16
	}
	answer[0] = x[0] - y[0]
	answer[0] &= 0xFFFF
	return answer
}

func (utf *utf8FontFile) generateChecksum(data []byte) []int {
	if (len(data) % 4) != 0 {
		for i := 0; (len(data) % 4) != 0; i++ {
			data = append(data, 0)
		}
	}
	answer := []int{0x0000, 0x0000}
	for i := 0; i < len(data); i += 4 {
		answer[0] += (int(data[i]) << 8) + int(data[i+1])
		answer[1] += (int(data[i+2]) << 8) + int(data[i+3])
		answer[0] += answer[1] >> 16
		answer[1] &= 0xFFFF
		answer[0] &= 0xFFFF
	}
	return answer
}

func (utf *utf8FontFile) setErrorf(format string, args ...any) {
	if utf.err == nil {
		utf.err = staticErrorf(errUTF8Font, format, args...)
	}
}

func (utf *utf8FontFile) setError(err error) {
	if utf.err == nil && err != nil {
		utf.err = err
	}
}

func (utf *utf8FontFile) readBytes(length int) []byte {
	data := utf.fileReader.Read(length)
	utf.setError(utf.fileReader.err)
	return data
}

func (utf *utf8FontFile) seek(shift int) int {
	_, err := utf.fileReader.seek(int64(shift), 0)
	utf.setError(err)
	return int(utf.fileReader.readerPosition)
}

func (utf *utf8FontFile) skip(delta int) {
	_, err := utf.fileReader.seek(int64(delta), 1)
	utf.setError(err)
}

func (utf *utf8FontFile) seekTable(name string) int {
	description := utf.tableDescriptions[name]
	if description == nil {
		utf.setErrorf("required font table %q is missing", name)
		return int(utf.fileReader.readerPosition)
	}
	return utf.seek(description.position)
}

func (utf *utf8FontFile) readInt16() int16 {
	s := utf.readBytes(2)
	return int16FromUint16Bits(binary.BigEndian.Uint16(s))
}

func (utf *utf8FontFile) getUint16(pos int) int {
	utf.seek(pos)
	s := utf.readBytes(2)
	return (int(s[0]) << 8) + int(s[1])
}

func (utf *utf8FontFile) getUint32(pos int) int {
	utf.seek(pos)
	return utf.readUint32()
}

func (utf *utf8FontFile) splice(stream []byte, offset int, value []byte) []byte {
	if offset < 0 || offset+len(value) > len(stream) {
		utf.setErrorf("font table splice offset %d is out of range", offset)
		return stream
	}
	stream = append([]byte{}, stream...)
	return append(append(stream[:offset], value...), stream[offset+len(value):]...)
}

func (utf *utf8FontFile) insertUint16(stream []byte, offset int, value int) []byte {
	return utf.splice(stream, offset, packUint16(value))
}

func (utf *utf8FontFile) getRange(pos, length int) []byte {
	if length < 1 {
		return make([]byte, 0)
	}
	if pos < 0 || pos > len(utf.fileReader.array) || pos+length > len(utf.fileReader.array) {
		utf.setErrorf("unexpected EOF reading font data")
		return nil
	}
	utf.seek(pos)
	return utf.readBytes(length)
}

func (utf *utf8FontFile) getTableData(name string) []byte {
	description := utf.tableDescriptions[name]
	if description == nil {
		return nil
	}
	if description.size == 0 {
		return nil
	}
	if description.position < 0 ||
		description.position > len(utf.fileReader.array) ||
		description.position+description.size > len(utf.fileReader.array) {
		utf.setErrorf("font table %q exceeds font data", name)
		return nil
	}
	utf.seek(description.position)
	return utf.readBytes(description.size)
}

func (utf *utf8FontFile) setOutTable(name string, data []byte) {
	if data == nil {
		return
	}
	if name == "head" {
		data = utf.splice(data, 8, []byte{0, 0, 0, 0})
	}
	utf.outTablesData[name] = data
}

func (utf *utf8FontFile) parseNAMETable() int {
	namePosition := utf.seekTable("name")
	format := utf.readUint16()
	if format != 0 {
		utf.setErrorf("unsupported name table format %d", format)
		return format
	}
	nameCount := utf.readUint16()
	stringDataPosition := namePosition + utf.readUint16()
	names := map[int]string{1: "", 2: "", 3: "", 4: "", 6: ""}
	counter := len(names)
	for range nameCount {
		system := utf.readUint16()
		code := utf.readUint16()
		local := utf.readUint16()
		nameID := utf.readUint16()
		size := utf.readUint16()
		position := utf.readUint16()
		if _, ok := names[nameID]; !ok {
			continue
		}
		currentName := ""
		if system == 3 && code == 1 && local == 0x409 {
			oldPos := utf.fileReader.readerPosition
			utf.seek(stringDataPosition + position)
			if size%2 != 0 {
				utf.setErrorf("name table string is not binary byte format")
				return format
			}
			size /= 2
			currentName = ""
			var currentNameSb422 strings.Builder
			for size > 0 {
				char := utf.readUint16()
				currentNameSb422.WriteRune(rune(char)) // #nosec G115 -- UTF-16 code units are valid rune values.
				size--
			}
			currentName += currentNameSb422.String()
			utf.fileReader.readerPosition = oldPos
			utf.seek(int(oldPos))
		} else if system == 1 && code == 0 && local == 0 {
			oldPos := utf.fileReader.readerPosition
			currentName = string(utf.getRange(stringDataPosition+position, size))
			utf.fileReader.readerPosition = oldPos
			utf.seek(int(oldPos))
		}
		if currentName != "" && names[nameID] == "" {
			names[nameID] = currentName
			counter--
			if counter == 0 {
				break
			}
		}
	}
	return format
}

func (utf *utf8FontFile) parseHEADTable() int {
	utf.seekTable("head")
	utf.skip(18)
	utf.fontElementSize = utf.readUint16()
	scale := 1000.0 / float64(utf.fontElementSize)
	utf.skip(16)
	xMin := utf.readInt16()
	yMin := utf.readInt16()
	xMax := utf.readInt16()
	yMax := utf.readInt16()
	utf.bbox = fontBoxType{int(float64(xMin) * scale), int(float64(yMin) * scale), int(float64(xMax) * scale), int(float64(yMax) * scale)}
	utf.skip(3 * 2)
	indexToLocFormat := utf.readUint16()
	symbolDataFormat := utf.readUint16()
	if symbolDataFormat != 0 {
		utf.setErrorf("unknown symbol data format %d", symbolDataFormat)
		return 0
	}
	return indexToLocFormat
}

func (utf *utf8FontFile) parseHHEATable() int {
	metricsCount := 0
	if _, OK := utf.tableDescriptions["hhea"]; OK {
		scale := 1000.0 / float64(utf.fontElementSize)
		utf.seekTable("hhea")
		utf.skip(4)
		hheaAscender := utf.readInt16()
		hheaDescender := utf.readInt16()
		utf.ascent = int(float64(hheaAscender) * scale)
		utf.descent = int(float64(hheaDescender) * scale)
		utf.skip(24)
		metricDataFormat := utf.readUint16()
		if metricDataFormat != 0 {
			utf.setErrorf("unknown horizontal metric data format %d", metricDataFormat)
			return 0
		}
		metricsCount = utf.readUint16()
		if metricsCount == 0 {
			utf.setErrorf("number of horizontal metrics is 0")
			return 0
		}
	}
	return metricsCount
}

func (utf *utf8FontFile) parseOS2Table() int {
	scale := 1000.0 / float64(utf.fontElementSize)
	if _, ok := utf.tableDescriptions["OS/2"]; !ok {
		utf.setFallbackOS2Metrics(scale)
		return 500
	}
	weightType := utf.parseOS2Metrics(scale)
	utf.stemV = 50 + int(math.Pow(float64(weightType)/65.0, 2))
	return weightType
}

func (utf *utf8FontFile) parseOS2Metrics(scale float64) int {
	utf.seekTable("OS/2")
	version := utf.readUint16()
	utf.skip(2)
	weightType := utf.readUint16()
	utf.skip(2)
	fsType := utf.readUint16()
	if fsType == 0x0002 || (fsType&0x0300) != 0 {
		utf.setErrorf("font cannot be embedded because of font embedding restrictions")
		return 0
	}
	utf.skip(20)
	_ = utf.readInt16()

	utf.skip(36)
	sTypoAscender := utf.readInt16()
	sTypoDescender := utf.readInt16()
	if utf.ascent == 0 {
		utf.ascent = int(float64(sTypoAscender) * scale)
	}
	if utf.descent == 0 {
		utf.descent = int(float64(sTypoDescender) * scale)
	}
	utf.setOS2CapHeight(version, scale)
	return weightType
}

func (utf *utf8FontFile) setOS2CapHeight(version int, scale float64) {
	if version <= 1 {
		utf.capHeight = utf.ascent
		return
	}
	utf.skip(16)
	sCapHeight := utf.readInt16()
	utf.capHeight = int(float64(sCapHeight) * scale)
}

func (utf *utf8FontFile) setFallbackOS2Metrics(scale float64) {
	if utf.ascent == 0 {
		utf.ascent = int(float64(utf.bbox.Ymax) * scale)
	}
	if utf.descent == 0 {
		utf.descent = int(float64(utf.bbox.Ymin) * scale)
	}
	utf.capHeight = utf.ascent
}

func (utf *utf8FontFile) parsePOSTTable(weight int) {
	utf.seekTable("post")
	utf.skip(4)
	utf.italicAngle = int(utf.readInt16()) + utf.readUint16()/65536.0
	scale := 1000.0 / float64(utf.fontElementSize)
	utf.underlinePosition = float64(utf.readInt16()) * scale
	utf.underlineThickness = float64(utf.readInt16()) * scale
	fixed := utf.readUint32()

	utf.flags = 4

	if utf.italicAngle != 0 {
		utf.flags |= 64
	}
	if weight >= 600 {
		utf.flags |= 262144
	}
	if fixed != 0 {
		utf.flags |= 1
	}
}

func (utf *utf8FontFile) parseCMAPTable() int {
	cmapPosition := utf.seekTable("cmap")
	utf.skip(2)
	cmapTableCount := utf.readUint16()
	cidCMAPPosition := 0
	for range cmapTableCount {
		system := utf.readUint16()
		coded := utf.readUint16()
		position := utf.readUint32()
		oldReaderPosition := utf.fileReader.readerPosition
		if (system == 3 && (coded == 1 || coded == 10)) || system == 0 { // Microsoft, Unicode
			format := utf.getUint16(cmapPosition + position)
			if format == 12 {
				cidCMAPPosition = cmapPosition + position
				break
			}
			if format == 4 {
				cidCMAPPosition = cmapPosition + position
			}
		}
		utf.seek(int(oldReaderPosition))
	}
	if cidCMAPPosition == 0 {
		utf.setErrorf("font does not have cmap for Unicode")
		return cidCMAPPosition
	}
	return cidCMAPPosition
}

func (utf *utf8FontFile) parseCOLRTable() {
	desc := utf.tableDescriptions["COLR"]
	if desc == nil {
		return
	}

	tableStart := desc.position
	utf.seek(tableStart)
	version, ok := utf.readCheckedUint16("COLR version")
	if !ok {
		return
	}
	if version > 1 {
		return
	}

	numBaseGlyphRecords := utf.readUint16()
	baseGlyphRecordsOffset := utf.readUint32()
	layerRecordsOffset := utf.readUint32()
	numLayerRecords := utf.readUint16()
	if utf.err != nil {
		return
	}

	colr := &colrTable{
		version:          version,
		baseGlyphRecords: make([]colorBaseGlyphRecord, numBaseGlyphRecords),
		layerRecords:     make([]colorLayerRecord, numLayerRecords),
	}
	if version == 1 {
		colr.baseGlyphListOffset = utf.readUint32()
		colr.layerListOffset = utf.readUint32()
		colr.clipListOffset = utf.readUint32()
	}

	if baseGlyphRecordsOffset != 0 {
		utf.seek(tableStart + baseGlyphRecordsOffset)
		for i := range numBaseGlyphRecords {
			glyphID, ok := utf.readCheckedUint16("COLR base glyph ID")
			if !ok {
				return
			}
			firstLayerIdx, ok := utf.readCheckedUint16("COLR first layer index")
			if !ok {
				return
			}
			numLayers, ok := utf.readCheckedUint16("COLR layer count")
			if !ok {
				return
			}
			colr.baseGlyphRecords[i] = colorBaseGlyphRecord{
				glyphID:       glyphID,
				firstLayerIdx: firstLayerIdx,
				numLayers:     numLayers,
			}
		}
		if utf.err != nil {
			return
		}
	}

	if layerRecordsOffset != 0 {
		utf.seek(tableStart + layerRecordsOffset)
		for i := range numLayerRecords {
			glyphID, ok := utf.readCheckedUint16("COLR layer glyph ID")
			if !ok {
				return
			}
			paletteIndex, ok := utf.readCheckedUint16("COLR palette index")
			if !ok {
				return
			}
			colr.layerRecords[i] = colorLayerRecord{
				glyphID:      glyphID,
				paletteIndex: paletteIndex,
			}
		}
		if utf.err != nil {
			return
		}
	}

	utf.colrTable = colr
}

func (utf *utf8FontFile) parseCPALTable() {
	desc := utf.tableDescriptions["CPAL"]
	if desc == nil {
		return
	}

	tableStart := desc.position
	utf.seek(tableStart)
	_ = utf.readUint16() // version
	_ = utf.readUint16() // numPaletteEntries
	_ = utf.readUint16() // numPalettes
	numColorRecords := utf.readUint16()
	colorRecordsArrayOffset := utf.readUint32()
	if utf.err != nil {
		return
	}

	cpal := &cpalTable{
		colorRecords: make([]colorRecord, numColorRecords),
	}
	utf.seek(tableStart + colorRecordsArrayOffset)
	for i := range numColorRecords {
		data := utf.fileReader.Read(4)
		if utf.err != nil || len(data) < 4 {
			return
		}
		cpal.colorRecords[i] = colorRecord{
			b: data[0],
			g: data[1],
			r: data[2],
			a: data[3],
		}
	}

	utf.cpalTable = cpal
}

func (utf *utf8FontFile) colorGlyphLayers(glyphID uint16) []colorLayerRecord {
	if utf.colrTable == nil {
		return nil
	}
	records := utf.colrTable.baseGlyphRecords
	lo, hi := 0, len(records)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		record := records[mid]
		switch {
		case record.glyphID == glyphID:
			start := int(record.firstLayerIdx)
			end := start + int(record.numLayers)
			if start < 0 || end > len(utf.colrTable.layerRecords) {
				return nil
			}
			return utf.colrTable.layerRecords[start:end]
		case record.glyphID < glyphID:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	if utf.colrTable.version == 1 && utf.colrTable.baseGlyphListOffset != 0 {
		return utf.colorGlyphLayersV1(glyphID, 0)
	}
	return nil
}

func (utf *utf8FontFile) paletteColor(paletteIndex uint16) colorRecord {
	if utf.cpalTable == nil || int(paletteIndex) >= len(utf.cpalTable.colorRecords) {
		return colorRecord{a: 255}
	}
	return utf.cpalTable.colorRecords[paletteIndex]
}

func (utf *utf8FontFile) colorGlyphLayersV1(glyphID uint16, depth int) []colorLayerRecord {
	if depth > 16 {
		return nil
	}

	colrStart := utf.tableDescriptions["COLR"].position
	listStart := colrStart + utf.colrTable.baseGlyphListOffset

	utf.seek(listStart)
	numRecords := utf.readUint32()
	lo, hi := 0, numRecords-1
	for lo <= hi {
		mid := (lo + hi) / 2
		utf.seek(listStart + 4 + mid*6)
		recordGlyphID, ok := utf.readCheckedUint16("COLR v1 record glyph ID")
		if !ok {
			return nil
		}
		paintOffset := utf.readUint32()
		if utf.err != nil {
			return nil
		}

		switch {
		case recordGlyphID == glyphID:
			return utf.parseCOLRV1Paint(listStart+paintOffset, depth+1)
		case recordGlyphID < glyphID:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return nil
}

func (utf *utf8FontFile) parseCOLRV1Paint(offset, depth int) []colorLayerRecord {
	if depth > 16 || offset <= 0 {
		return nil
	}

	utf.seek(offset)
	format := utf.readByte()
	if utf.err != nil {
		return nil
	}

	switch format {
	case 1:
		layerCount := utf.readByte()
		firstLayerIndex := utf.readUint32()
		layerListStart := utf.tableDescriptions["COLR"].position + utf.colrTable.layerListOffset
		utf.seek(layerListStart)
		totalLayers := utf.readUint32()
		if utf.err != nil || firstLayerIndex < 0 || firstLayerIndex+layerCount > totalLayers {
			return nil
		}

		layers := make([]colorLayerRecord, 0, layerCount)
		for i := range layerCount {
			utf.seek(layerListStart + 4 + (firstLayerIndex+i)*4)
			paintOffset := utf.readUint32()
			layers = append(layers, utf.parseCOLRV1Paint(layerListStart+paintOffset, depth+1)...)
		}
		return layers
	case 10:
		paintOffset := utf.readOffset24()
		glyphID, ok := utf.readCheckedUint16("COLR v1 glyph ID")
		if !ok {
			return nil
		}
		paletteIndex := utf.parseCOLRV1Color(offset+paintOffset, depth+1)
		if utf.err != nil {
			return nil
		}
		return []colorLayerRecord{{
			glyphID:      glyphID,
			paletteIndex: paletteIndex,
		}}
	case 11:
		reusedGlyphID, ok := utf.readCheckedUint16("COLR v1 reused glyph ID")
		if !ok {
			return nil
		}
		return utf.colorGlyphLayersV1(reusedGlyphID, depth+1)
	case 12, 14, 16, 24, 26, 28, 30:
		paintOffset := utf.readOffset24()
		return utf.parseCOLRV1Paint(offset+paintOffset, depth+1)
	case 13:
		paintOffset := utf.readOffset24()
		return utf.parseCOLRV1Paint(offset+paintOffset, depth+1)
	case 15:
		paintOffset := utf.readOffset24()
		return utf.parseCOLRV1Paint(offset+paintOffset, depth+1)
	case 17, 18, 19, 20, 21, 22, 23, 25, 27, 29, 31:
		paintOffset := utf.readOffset24()
		return utf.parseCOLRV1Paint(offset+paintOffset, depth+1)
	case 32:
		sourceOffset := utf.readOffset24()
		_ = utf.readByte() // composite mode
		backdropOffset := utf.readOffset24()
		layers := utf.parseCOLRV1Paint(offset+backdropOffset, depth+1)
		layers = append(layers, utf.parseCOLRV1Paint(offset+sourceOffset, depth+1)...)
		return layers
	default:
		return nil
	}
}

func (utf *utf8FontFile) parseCOLRV1Color(offset, depth int) uint16 {
	if depth > 16 || offset <= 0 {
		return 0xFFFF
	}

	utf.seek(offset)
	format := utf.readByte()
	if utf.err != nil {
		return 0xFFFF
	}

	switch format {
	case 2:
		paletteIndex, ok := utf.readCheckedUint16("COLR v1 palette index")
		if !ok {
			return 0xFFFF
		}
		return paletteIndex
	case 3:
		paletteIndex, ok := utf.readCheckedUint16("COLR v1 palette index")
		if !ok {
			return 0xFFFF
		}
		return paletteIndex
	case 4, 5, 6, 7, 8, 9:
		colorLineOffset := utf.readOffset24()
		return utf.firstPaletteIndexInColorLine(offset + colorLineOffset)
	case 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31:
		paintOffset := utf.readOffset24()
		return utf.parseCOLRV1Color(offset+paintOffset, depth+1)
	default:
		return 0xFFFF
	}
}

func (utf *utf8FontFile) firstPaletteIndexInColorLine(offset int) uint16 {
	utf.seek(offset)
	_ = utf.readByte() // extend
	stopCount := utf.readUint16()
	if utf.err != nil || stopCount == 0 {
		return 0xFFFF
	}
	_ = utf.readUint16() // stop offset
	paletteIndex, ok := utf.readCheckedUint16("COLR v1 color stop palette index")
	if !ok {
		return 0xFFFF
	}
	return paletteIndex
}

func (utf *utf8FontFile) parseBitmapTables(numGlyphs int) {
	utf.parseCBLCTable()
	utf.parseSBIXTable(numGlyphs)
	utf.hasBitmapGlyphs = len(utf.cbdtGlyphs) > 0 || len(utf.sbixStrikes) > 0
}

func (utf *utf8FontFile) parseCBLCTable() {
	cblc := utf.tableDescriptions["CBLC"]
	cbdt := utf.tableDescriptions["CBDT"]
	if cblc == nil || cbdt == nil {
		return
	}

	utf.seek(cblc.position)
	_ = utf.readUint32() // version
	strikeCount := utf.readUint32()
	if utf.err != nil {
		return
	}

	utf.cbdtGlyphs = make(map[int][]bitmapGlyphLocation)
	for strikeIndex := range strikeCount {
		strikeOffset := cblc.position + 8 + strikeIndex*48
		utf.seek(strikeOffset)
		indexSubTableArrayOffset := utf.readUint32()
		_ = utf.readUint32() // indexTablesSize
		subTableCount := utf.readUint32()
		_ = utf.readUint32() // colorRef
		utf.skip(24)         // horizontal and vertical SbitLineMetrics
		_ = utf.readUint16() // startGlyphIndex
		_ = utf.readUint16() // endGlyphIndex
		ppemX := utf.readByte()
		ppemY := utf.readByte()
		_ = utf.readByte() // bitDepth
		_ = utf.readByte() // flags
		if utf.err != nil {
			return
		}

		arrayStart := cblc.position + indexSubTableArrayOffset
		for i := range subTableCount {
			utf.seek(arrayStart + i*8)
			firstGlyph := utf.readUint16()
			lastGlyph := utf.readUint16()
			subtableOffset := utf.readUint32()
			if utf.err != nil {
				return
			}
			utf.parseCBLCIndexSubTable(cblc.position+indexSubTableArrayOffset+subtableOffset, firstGlyph, lastGlyph, ppemX, ppemY)
		}
	}
}

func (utf *utf8FontFile) parseCBLCIndexSubTable(offset, firstGlyph, lastGlyph, ppemX, ppemY int) {
	if lastGlyph < firstGlyph {
		return
	}

	utf.seek(offset)
	indexFormat := utf.readUint16()
	imageFormat := utf.readUint16()
	imageDataOffset := utf.readUint32()
	if utf.err != nil {
		return
	}
	if !isCBDTPNGImageFormat(imageFormat) {
		return
	}

	cbdt := utf.tableDescriptions["CBDT"]
	if cbdt == nil {
		return
	}
	imageDataStart := cbdt.position + imageDataOffset
	glyphCount := lastGlyph - firstGlyph + 1

	switch indexFormat {
	case 1:
		utf.parseCBLCFormat1(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, glyphCount)
	case 2:
		utf.parseCBLCFormat2(firstGlyph, lastGlyph, imageDataStart, imageFormat, ppemX, ppemY)
	case 3:
		utf.parseCBLCFormat3(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, glyphCount)
	case 4:
		utf.parseCBLCFormat4(imageDataStart, imageFormat, ppemX, ppemY)
	case 5:
		utf.parseCBLCFormat5(imageDataStart, imageFormat, ppemX, ppemY)
	}
}

func (utf *utf8FontFile) parseCBLCFormat1(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, glyphCount int) {
	offsets := make([]int, glyphCount+1)
	for i := range offsets {
		offsets[i] = utf.readUint32()
	}
	if utf.err != nil {
		return
	}
	utf.addCBDTOffsetLocations(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, offsets, nil)
}

func (utf *utf8FontFile) parseCBLCFormat2(firstGlyph, lastGlyph, imageDataStart, imageFormat, ppemX, ppemY int) {
	imageSize := utf.readUint32()
	metrics := utf.readBigGlyphMetrics()
	if utf.err != nil || imageSize <= 0 {
		return
	}
	for glyphID := firstGlyph; glyphID <= lastGlyph; glyphID++ {
		idx := glyphID - firstGlyph
		utf.addCBDTGlyphLocation(glyphID, bitmapGlyphLocation{
			dataOffset:  imageDataStart + idx*imageSize,
			dataLength:  imageSize,
			imageFormat: imageFormat,
			ppemX:       ppemX,
			ppemY:       ppemY,
			hasMetrics:  true,
			width:       metrics.width,
			height:      metrics.height,
			bearingX:    metrics.bearingX,
			bearingY:    metrics.bearingY,
			advance:     metrics.advance,
		})
	}
}

func (utf *utf8FontFile) parseCBLCFormat3(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, glyphCount int) {
	offsets := make([]int, glyphCount+1)
	for i := range offsets {
		offsets[i] = utf.readUint16()
	}
	if utf.err != nil {
		return
	}
	utf.addCBDTOffsetLocations(firstGlyph, imageDataStart, imageFormat, ppemX, ppemY, offsets, nil)
}

func (utf *utf8FontFile) parseCBLCFormat4(imageDataStart, imageFormat, ppemX, ppemY int) {
	pairCount := utf.readUint32()
	if utf.err != nil || pairCount < 1 {
		return
	}
	pairs := make([]bitmapGlyphOffsetPair, pairCount+1)
	for i := range pairs {
		pairs[i] = bitmapGlyphOffsetPair{
			glyphID: utf.readUint16(),
			offset:  utf.readUint16(),
		}
	}
	if utf.err != nil {
		return
	}
	for i := range pairCount {
		dataLength := pairs[i+1].offset - pairs[i].offset
		if dataLength <= 0 {
			continue
		}
		utf.addCBDTGlyphLocation(pairs[i].glyphID, bitmapGlyphLocation{
			dataOffset:  imageDataStart + pairs[i].offset,
			dataLength:  dataLength,
			imageFormat: imageFormat,
			ppemX:       ppemX,
			ppemY:       ppemY,
		})
	}
}

func (utf *utf8FontFile) parseCBLCFormat5(imageDataStart, imageFormat, ppemX, ppemY int) {
	imageSize := utf.readUint32()
	metrics := utf.readBigGlyphMetrics()
	sparseGlyphCount := utf.readUint32()
	if utf.err != nil || imageSize <= 0 {
		return
	}
	for i := range sparseGlyphCount {
		glyphID := utf.readUint16()
		utf.addCBDTGlyphLocation(glyphID, bitmapGlyphLocation{
			dataOffset:  imageDataStart + i*imageSize,
			dataLength:  imageSize,
			imageFormat: imageFormat,
			ppemX:       ppemX,
			ppemY:       ppemY,
			hasMetrics:  true,
			width:       metrics.width,
			height:      metrics.height,
			bearingX:    metrics.bearingX,
			bearingY:    metrics.bearingY,
			advance:     metrics.advance,
		})
	}
}

type bitmapGlyphOffsetPair struct {
	glyphID int
	offset  int
}

type bitmapGlyphMetrics struct {
	width    int
	height   int
	bearingX int
	bearingY int
	advance  int
}

func isCBDTPNGImageFormat(imageFormat int) bool {
	return imageFormat == 17 || imageFormat == 18 || imageFormat == 19
}

func (utf *utf8FontFile) addCBDTOffsetLocations(
	firstGlyph, imageDataStart, imageFormat, ppemX, ppemY int,
	offsets []int,
	metrics *bitmapGlyphMetrics,
) {
	for idx := 0; idx+1 < len(offsets); idx++ {
		dataLength := offsets[idx+1] - offsets[idx]
		if dataLength <= 0 {
			continue
		}
		location := bitmapGlyphLocation{
			dataOffset:  imageDataStart + offsets[idx],
			dataLength:  dataLength,
			imageFormat: imageFormat,
			ppemX:       ppemX,
			ppemY:       ppemY,
		}
		if metrics != nil {
			location.hasMetrics = true
			location.width = metrics.width
			location.height = metrics.height
			location.bearingX = metrics.bearingX
			location.bearingY = metrics.bearingY
			location.advance = metrics.advance
		}
		utf.addCBDTGlyphLocation(firstGlyph+idx, location)
	}
}

func (utf *utf8FontFile) addCBDTGlyphLocation(glyphID int, location bitmapGlyphLocation) {
	utf.cbdtGlyphs[glyphID] = append(utf.cbdtGlyphs[glyphID], location)
}

func (utf *utf8FontFile) readBigGlyphMetrics() bitmapGlyphMetrics {
	height := utf.readByte()
	width := utf.readByte()
	bearingX, ok := utf.readInt8Bits("big glyph bearingX")
	if !ok {
		return bitmapGlyphMetrics{}
	}
	bearingY, ok := utf.readInt8Bits("big glyph bearingY")
	if !ok {
		return bitmapGlyphMetrics{}
	}
	advance := utf.readByte()
	_ = utf.readByte() // vertBearingX
	_ = utf.readByte() // vertBearingY
	_ = utf.readByte() // vertAdvance
	return bitmapGlyphMetrics{
		width:    width,
		height:   height,
		bearingX: bearingX,
		bearingY: bearingY,
		advance:  advance,
	}
}

func (utf *utf8FontFile) parseSBIXTable(numGlyphs int) {
	sbix := utf.tableDescriptions["sbix"]
	if sbix == nil || numGlyphs < 1 {
		return
	}

	utf.seek(sbix.position)
	_ = utf.readUint16() // version
	_ = utf.readUint16() // flags
	strikeCount := utf.readUint32()
	if utf.err != nil {
		return
	}

	utf.sbixStrikes = make([]sbixStrike, 0, strikeCount)
	strikeOffsets := make([]int, strikeCount)
	for i := range strikeOffsets {
		strikeOffsets[i] = utf.readUint32()
		if utf.err != nil {
			return
		}
	}

	for _, strikeOffset := range strikeOffsets {
		strikeStart := sbix.position + strikeOffset
		if strikeStart+4 > sbix.position+sbix.size {
			continue
		}
		utf.seek(strikeStart)
		ppem := utf.readUint16()
		ppi := utf.readUint16()
		if utf.err != nil {
			return
		}
		if strikeStart+4+(numGlyphs+1)*4 > sbix.position+sbix.size {
			continue
		}
		utf.sbixStrikes = append(utf.sbixStrikes, sbixStrike{
			offset: strikeOffset,
			ppem:   ppem,
			ppi:    ppi,
		})
	}
}

func (utf *utf8FontFile) bitmapGlyphImage(glyphID uint16, sizePt float64) *bitmapGlyphImage {
	if glyph := utf.cbdtGlyphImage(int(glyphID), sizePt); glyph != nil {
		return glyph
	}
	return utf.sbixGlyphImage(int(glyphID), sizePt)
}

func (utf *utf8FontFile) cbdtGlyphImage(glyphID int, sizePt float64) *bitmapGlyphImage {
	locations := utf.cbdtGlyphs[glyphID]
	if len(locations) == 0 {
		return nil
	}
	location := locations[0]
	for _, candidate := range locations[1:] {
		if betterBitmapStrike(candidate.ppemY, location.ppemY, sizePt) {
			location = candidate
		}
	}
	if !isCBDTPNGImageFormat(location.imageFormat) || location.dataLength < 4 {
		return nil
	}
	raw := utf.getRange(location.dataOffset, location.dataLength)
	if utf.err != nil {
		return nil
	}

	metrics, pngData := cbdtPNGImageData(location, raw)
	if len(pngData) == 0 {
		return nil
	}
	if !bytes.HasPrefix(pngData, []byte("\x89PNG\r\n\x1a\n")) {
		return nil
	}
	return &bitmapGlyphImage{
		data:      append([]byte(nil), pngData...),
		imageType: "png",
		width:     metrics.width,
		height:    metrics.height,
		bearingX:  metrics.bearingX,
		bearingY:  metrics.bearingY,
		advance:   metrics.advance,
		ppemX:     location.ppemX,
		ppemY:     location.ppemY,
	}
}

func cbdtPNGImageData(location bitmapGlyphLocation, raw []byte) (bitmapGlyphMetrics, []byte) {
	switch location.imageFormat {
	case 17:
		if len(raw) < 9 {
			return bitmapGlyphMetrics{}, nil
		}
		metrics := bitmapGlyphMetrics{
			height:   int(raw[0]),
			width:    int(raw[1]),
			bearingX: signedByteValue(raw[2]),
			bearingY: signedByteValue(raw[3]),
			advance:  int(raw[4]),
		}
		return metrics, sizedPNGData(raw, 5)
	case 18:
		if len(raw) < 12 {
			return bitmapGlyphMetrics{}, nil
		}
		metrics := bitmapGlyphMetrics{
			height:   int(raw[0]),
			width:    int(raw[1]),
			bearingX: signedByteValue(raw[2]),
			bearingY: signedByteValue(raw[3]),
			advance:  int(raw[4]),
		}
		return metrics, sizedPNGData(raw, 8)
	case 19:
		if !location.hasMetrics {
			return bitmapGlyphMetrics{}, nil
		}
		metrics := bitmapGlyphMetrics{
			width:    location.width,
			height:   location.height,
			bearingX: location.bearingX,
			bearingY: location.bearingY,
			advance:  location.advance,
		}
		return metrics, sizedPNGData(raw, 0)
	default:
		return bitmapGlyphMetrics{}, nil
	}
}

func sizedPNGData(raw []byte, offset int) []byte {
	if offset < 0 || offset+4 > len(raw) {
		return nil
	}
	dataLength := int(binary.BigEndian.Uint32(raw[offset : offset+4]))
	dataOffset := offset + 4
	if dataLength < 0 || dataOffset+dataLength > len(raw) {
		return nil
	}
	return raw[dataOffset : dataOffset+dataLength]
}

func (utf *utf8FontFile) sbixGlyphImage(glyphID int, sizePt float64) *bitmapGlyphImage {
	if len(utf.sbixStrikes) == 0 {
		return nil
	}
	strike := utf.sbixStrikes[0]
	for _, candidate := range utf.sbixStrikes[1:] {
		if betterBitmapStrike(candidate.ppem, strike.ppem, sizePt) {
			strike = candidate
		}
	}

	sbix := utf.tableDescriptions["sbix"]
	if sbix == nil {
		return nil
	}
	strikeStart := sbix.position + strike.offset
	glyphOffsetPos := strikeStart + 4 + glyphID*4
	nextGlyphOffsetPos := glyphOffsetPos + 4
	if nextGlyphOffsetPos+4 > sbix.position+sbix.size {
		return nil
	}
	glyphOffset := utf.getUint32(glyphOffsetPos)
	nextGlyphOffset := utf.getUint32(nextGlyphOffsetPos)
	if utf.err != nil || nextGlyphOffset <= glyphOffset || glyphOffset < 4 {
		return nil
	}
	dataOffset := strikeStart + glyphOffset
	dataLength := nextGlyphOffset - glyphOffset
	if dataLength < 8 || dataOffset+dataLength > sbix.position+sbix.size {
		return nil
	}
	raw := utf.getRange(dataOffset, dataLength)
	if utf.err != nil || len(raw) < 8 {
		return nil
	}
	originOffsetX := int(int16FromUint16Bits(binary.BigEndian.Uint16(raw[0:2])))
	originOffsetY := int(int16FromUint16Bits(binary.BigEndian.Uint16(raw[2:4])))
	imageType := strings.TrimSpace(string(raw[4:8]))
	imageData := raw[8:]
	switch imageType {
	case imageTypePNG:
		if !bytes.HasPrefix(imageData, []byte("\x89PNG\r\n\x1a\n")) {
			return nil
		}
	default:
		return nil
	}
	width, height := pngSize(imageData)
	if width == 0 || height == 0 {
		return nil
	}
	return &bitmapGlyphImage{
		data:          append([]byte(nil), imageData...),
		imageType:     imageType,
		width:         width,
		height:        height,
		ppemX:         strike.ppem,
		ppemY:         strike.ppem,
		originOffsetX: originOffsetX,
		originOffsetY: originOffsetY,
	}
}

func betterBitmapStrike(candidate, current int, sizePt float64) bool {
	target := max(bitmapGlyphTargetPPEm(sizePt), 1)
	if current < target && candidate >= target {
		return true
	}
	if current >= target && candidate >= target {
		return candidate < current
	}
	if current < target && candidate < target {
		return candidate > current
	}
	return false
}

func bitmapGlyphTargetPPEm(sizePt float64) int {
	return int(sizePt*bitmapEmojiResolutionScale + 0.5)
}

func pngSize(data []byte) (int, int) {
	if len(data) < 24 || !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return 0, 0
	}
	return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24]))
}

func (utf *utf8FontFile) parseTables() {
	utf.parseNAMETable()
	if utf.err != nil {
		return
	}
	locaFormat := utf.parseHEADTable()
	if utf.err != nil {
		return
	}
	n := utf.parseHHEATable()
	if utf.err != nil {
		return
	}
	w := utf.parseOS2Table()
	if utf.err != nil {
		return
	}
	utf.parsePOSTTable(w)
	if utf.err != nil {
		return
	}
	runeCMAPPosition := utf.parseCMAPTable()
	if utf.err != nil {
		return
	}

	utf.parseCOLRTable()
	utf.parseCPALTable()

	utf.seekTable("maxp")
	utf.skip(4)
	numSymbols := utf.readUint16()
	if utf.err != nil {
		return
	}
	utf.parseBitmapTables(numSymbols)
	utf.hasColorGlyphs = (utf.colrTable != nil && utf.cpalTable != nil) || utf.hasBitmapGlyphs

	symbolCharDictionary := make(map[int][]int)
	charSymbolDictionary := make(map[int]int)
	utf.generateSCCSDictionaries(runeCMAPPosition, symbolCharDictionary, charSymbolDictionary)
	utf.charSymbolDictionary = charSymbolDictionary

	scale := 1000.0 / float64(utf.fontElementSize)
	utf.parseHMTXTable(n, numSymbols, symbolCharDictionary, scale)
	if utf.err != nil {
		return
	}
	if utf.tableDescriptions["loca"] != nil && utf.tableDescriptions["glyf"] != nil {
		utf.parseLOCATable(locaFormat, numSymbols)
	} else if !utf.hasBitmapGlyphs {
		utf.setErrorf("required font table %q is missing", "loca")
	}
}

func (utf *utf8FontFile) generateCMAP() map[int][]int {
	cmapPosition := utf.seekTable("cmap")
	utf.skip(2)
	cmapTableCount := utf.readUint16()
	runeCmapPosition := 0
	for range cmapTableCount {
		system := utf.readUint16()
		coder := utf.readUint16()
		position := utf.readUint32()
		oldPosition := utf.fileReader.readerPosition
		if (system == 3 && (coder == 1 || coder == 10)) || system == 0 {
			format := utf.getUint16(cmapPosition + position)
			if format == 12 {
				runeCmapPosition = cmapPosition + position
				break
			}
			if format == 4 {
				runeCmapPosition = cmapPosition + position
			}
		}
		utf.seek(int(oldPosition))
	}

	if runeCmapPosition == 0 {
		utf.setErrorf("font does not have cmap for Unicode")
		return nil
	}

	symbolCharDictionary := make(map[int][]int)
	charSymbolDictionary := make(map[int]int)
	utf.generateSCCSDictionaries(runeCmapPosition, symbolCharDictionary, charSymbolDictionary)

	utf.charSymbolDictionary = charSymbolDictionary

	return symbolCharDictionary
}

func (utf *utf8FontFile) parseSymbols(usedRunes map[int]int) (map[int]int, map[int]int, map[int]int, []int) {
	symbolCollection := map[int]int{0: 0}
	charSymbolPairCollection := make(map[int]int)
	for cid, char := range usedRunes {
		glyphID, ok := utf.charSymbolDictionary[char]
		if ok {
			utf.addSymbol(char, cid, glyphID, symbolCollection, charSymbolPairCollection)
		}
		utf.lastRune = max(utf.lastRune, cid)
	}

	begin := utf.tableDescriptions["glyf"].position

	symbolArray := make(map[int]int)
	symbolCollectionKeys := keySortInt(symbolCollection)

	symbolCounter := 0
	maxRune := 0
	for _, oldSymbolIndex := range symbolCollectionKeys {
		maxRune = max(maxRune, symbolCollection[oldSymbolIndex])
		symbolArray[oldSymbolIndex] = symbolCounter
		symbolCounter++
	}
	charSymbolPairCollectionKeys := keySortInt(charSymbolPairCollection)
	runeSymbolPairCollection := make(map[int]int)
	for _, runa := range charSymbolPairCollectionKeys {
		runeSymbolPairCollection[runa] = symbolArray[charSymbolPairCollection[runa]]
	}
	utf.codeSymbolDictionary = runeSymbolPairCollection

	symbolCollectionKeys = keySortInt(symbolCollection)
	for _, oldSymbolIndex := range symbolCollectionKeys {
		symbolArray, symbolCollection, symbolCollectionKeys = utf.getSymbols(
			oldSymbolIndex,
			&begin,
			symbolArray,
			symbolCollection,
			symbolCollectionKeys,
		)
	}

	return runeSymbolPairCollection, symbolArray, symbolCollection, symbolCollectionKeys
}

func (utf *utf8FontFile) addSymbol(
	char, cid, glyphID int,
	symbolCollection map[int]int,
	charSymbolPairCollection map[int]int,
) {
	symbolCollection[glyphID] = char
	charSymbolPairCollection[cid] = glyphID
	if !utf.hasColorGlyphs {
		return
	}
	colorGlyphID, ok := checkedUint16(glyphID)
	if !ok {
		return
	}
	for _, layer := range utf.colorGlyphLayers(colorGlyphID) {
		layerGlyphID := int(layer.glyphID)
		if _, exists := symbolCollection[layerGlyphID]; !exists {
			symbolCollection[layerGlyphID] = 0
		}
	}
}

func (utf *utf8FontFile) generateCMAPTable(cidSymbolPairCollection map[int]int, numSymbols int) []byte {
	cidSymbolPairCollectionKeys := keySortInt(cidSymbolPairCollection)
	cidID := 0
	cidArray := make(map[int][]int)
	prevCid := -2
	prevSymbol := -1
	for _, cid := range cidSymbolPairCollectionKeys {
		if cid == (prevCid+1) && cidSymbolPairCollection[cid] == (prevSymbol+1) {
			if n, OK := cidArray[cidID]; !OK || n == nil {
				cidArray[cidID] = make([]int, 0)
			}
			cidArray[cidID] = append(cidArray[cidID], cidSymbolPairCollection[cid])
		} else {
			cidID = cid
			cidArray[cidID] = make([]int, 0)
			cidArray[cidID] = append(cidArray[cidID], cidSymbolPairCollection[cid])
		}
		prevCid = cid
		prevSymbol = cidSymbolPairCollection[cid]
	}
	cidArrayKeys := keySortArrayRangeMap(cidArray)
	segCount := len(cidArray) + 1

	searchRange := 1
	entrySelector := 0
	for searchRange*2 <= segCount {
		searchRange *= 2
		entrySelector++
	}
	searchRange *= 2
	rangeShift := segCount*2 - searchRange
	length := 16 + (8 * segCount) + (numSymbols + 1)
	cmap := make([]int, 0, 13+len(cidArrayKeys)*4+len(cidArray)+numSymbols+3)
	cmap = append(cmap, 0, 1, 3, 1, 0, 12, 4, length, 0, segCount*2, searchRange, entrySelector, rangeShift)

	for _, start := range cidArrayKeys {
		endCode := start + (len(cidArray[start]) - 1)
		cmap = append(cmap, endCode)
	}
	cmap = append(cmap, 0xFFFF)
	cmap = append(cmap, 0)

	cmap = append(cmap, cidArrayKeys...)
	cmap = append(cmap, 0xFFFF)
	for _, cidKey := range cidArrayKeys {
		idDelta := -(cidKey - cidArray[cidKey][0])
		cmap = append(cmap, idDelta)
	}
	cmap = append(cmap, 1)
	for range cidArray {
		cmap = append(cmap, 0)
	}
	cmap = append(cmap, 0)
	for _, start := range cidArrayKeys {
		cmap = append(cmap, cidArray[start]...)
	}
	cmap = append(cmap, 0)
	cmapstr := make([]byte, 0, len(cmap)*2)
	for _, cm := range cmap {
		cmapstr = append(cmapstr, packUint16(cm)...)
	}
	return cmapstr
}

// generateCutFont fill utf8FontFile from .utf file, only with runes from usedRunes
func (utf *utf8FontFile) generateCutFont(usedRunes map[int]int) []byte {
	utf.resetCutFontState()
	metadata, ok := utf.readCutFontMetadata()
	if !ok {
		return nil
	}

	cidSymbolPairCollection, symbolArray, symbolCollection, symbolCollectionKeys := utf.parseSymbols(usedRunes)
	metricsCount := len(symbolCollection)
	numSymbols := metricsCount

	if !utf.copyCutFontStaticTables() {
		return nil
	}

	delete(cidSymbolPairCollection, 0)
	utf.setOutTable("cmap", utf.generateCMAPTable(cidSymbolPairCollection, numSymbols))

	glyphBuilder, ok := utf.newCutFontGlyphBuilder(symbolArray, metadata.oldMetrics)
	if !ok {
		return nil
	}
	for _, originalSymbolIdx := range symbolCollectionKeys {
		if !glyphBuilder.appendGlyph(originalSymbolIdx) {
			return nil
		}
	}
	locaFormat := glyphBuilder.finish()
	if !utf.updateCutFontHeaderTables(locaFormat, metricsCount, numSymbols) {
		return nil
	}

	return utf.assembleTables()
}

type cutFontMetadata struct {
	oldMetrics int
}

func (utf *utf8FontFile) resetCutFontState() {
	utf.fileReader.readerPosition = int64(utf.fontOffset)
	utf.fileReader.err = nil
	utf.err = nil
	utf.symbolPosition = make([]int, 0)
	utf.charSymbolDictionary = make(map[int]int)
	utf.tableDescriptions = make(map[string]*tableDescription)
	utf.outTablesData = make(map[string][]byte)
	utf.ascent = 0
	utf.descent = 0
	utf.skip(4)
	utf.lastRune = 0
	utf.generateTableDescriptions()
	if utf.err != nil {
		return
	}
	utf.parseCOLRTable()
	utf.parseCPALTable()
	utf.hasColorGlyphs = utf.colrTable != nil && utf.cpalTable != nil
}

func (utf *utf8FontFile) readCutFontMetadata() (cutFontMetadata, bool) {
	if utf.err != nil {
		return cutFontMetadata{}, false
	}
	utf.seekTable("head")
	utf.skip(50)
	locaFormat := utf.readUint16()
	if utf.err != nil {
		return cutFontMetadata{}, false
	}

	utf.seekTable("hhea")
	utf.skip(34)
	metricsCount := utf.readUint16()
	oldMetrics := metricsCount
	if utf.err != nil {
		return cutFontMetadata{}, false
	}

	utf.seekTable("maxp")
	utf.skip(4)
	numSymbols := utf.readUint16()
	if utf.err != nil {
		return cutFontMetadata{}, false
	}

	symbolCharDictionary := utf.generateCMAP()
	if symbolCharDictionary == nil || utf.err != nil {
		return cutFontMetadata{}, false
	}

	utf.parseHMTXTable(metricsCount, numSymbols, symbolCharDictionary, 1.0)
	if utf.err != nil {
		return cutFontMetadata{}, false
	}

	utf.parseLOCATable(locaFormat, numSymbols)
	if utf.err != nil {
		return cutFontMetadata{}, false
	}

	return cutFontMetadata{oldMetrics: oldMetrics}, true
}

func (utf *utf8FontFile) copyCutFontStaticTables() bool {
	utf.setOutTable("name", utf.getTableData("name"))
	utf.setOutTable("cvt ", utf.getTableData("cvt "))
	utf.setOutTable("fpgm", utf.getTableData("fpgm"))
	utf.setOutTable("prep", utf.getTableData("prep"))
	utf.setOutTable("gasp", utf.getTableData("gasp"))
	return utf.setCutFontPostTable()
}

func (utf *utf8FontFile) setCutFontPostTable() bool {
	postTable := utf.getTableData("post")
	if utf.err != nil {
		return false
	}
	if len(postTable) < 16 {
		utf.setErrorf("post table is truncated")
		return false
	}
	utf.setOutTable("post", cutFontPostTable(postTable))
	return true
}

func cutFontPostTable(postTable []byte) []byte {
	rewritten := make([]byte, 0, 32)
	rewritten = append(rewritten, 0x00, 0x03, 0x00, 0x00)
	rewritten = append(rewritten, postTable[4:16]...)
	return append(rewritten, make([]byte, 16)...)
}

type cutFontGlyphBuilder struct {
	utf         *utf8FontFile
	symbolData  []byte
	symbolArray map[int]int
	oldMetrics  int
	offsets     []int
	glyfData    []byte
	hmtxData    []byte
	pos         int
}

func (utf *utf8FontFile) newCutFontGlyphBuilder(symbolArray map[int]int, oldMetrics int) (cutFontGlyphBuilder, bool) {
	symbolData := utf.getTableData("glyf")
	if utf.err != nil {
		return cutFontGlyphBuilder{}, false
	}
	utf.symbolData = make(map[int]map[string][]int, 0)
	return cutFontGlyphBuilder{
		utf:         utf,
		symbolData:  symbolData,
		symbolArray: symbolArray,
		oldMetrics:  oldMetrics,
		offsets:     make([]int, 0, len(symbolArray)+1),
		hmtxData:    make([]byte, 0, len(symbolArray)*4),
	}, true
}

func (builder *cutFontGlyphBuilder) appendGlyph(originalSymbolIdx int) bool {
	builder.hmtxData = append(builder.hmtxData, builder.utf.getMetrics(builder.oldMetrics, originalSymbolIdx)...)
	builder.offsets = append(builder.offsets, builder.pos)

	symbolPos := builder.utf.symbolPosition[originalSymbolIdx]
	symbolLen := builder.utf.symbolPosition[originalSymbolIdx+1] - symbolPos
	data, ok := builder.utf.glyphData(builder.symbolData, symbolPos, symbolLen)
	if !ok {
		return false
	}
	if builder.isCompositeGlyph(data, symbolLen) {
		data = builder.utf.rewriteCompositeGlyph(data, originalSymbolIdx, builder.symbolArray)
		if builder.utf.err != nil {
			return false
		}
	}

	builder.glyfData = append(builder.glyfData, data...)
	builder.pos += symbolLen
	builder.pad()
	return true
}

func (builder *cutFontGlyphBuilder) isCompositeGlyph(data []byte, symbolLen int) bool {
	if symbolLen <= 2 {
		return false
	}
	return unpackUint16(data[0:2])&(1<<15) != 0
}

func (builder *cutFontGlyphBuilder) pad() {
	if builder.pos%4 == 0 {
		return
	}
	padding := 4 - (builder.pos % 4)
	builder.glyfData = append(builder.glyfData, make([]byte, padding)...)
	builder.pos += padding
}

func (builder *cutFontGlyphBuilder) finish() int {
	builder.offsets = append(builder.offsets, builder.pos)
	builder.utf.setOutTable("glyf", builder.glyfData)
	builder.utf.setOutTable("hmtx", builder.hmtxData)
	return builder.utf.setCutFontLocaTable(builder.pos, builder.offsets)
}

func (utf *utf8FontFile) setCutFontLocaTable(pos int, offsets []int) int {
	if ((pos + 1) >> 1) > 0xFFFF {
		locaData := make([]byte, 0, len(offsets)*4)
		for _, offset := range offsets {
			locaData = append(locaData, packUint32(offset)...)
		}
		utf.setOutTable("loca", locaData)
		return 1
	}

	locaData := make([]byte, 0, len(offsets)*2)
	for _, offset := range offsets {
		locaData = append(locaData, packUint16(offset/2)...)
	}
	utf.setOutTable("loca", locaData)
	return 0
}

func (utf *utf8FontFile) updateCutFontHeaderTables(locaFormat, metricsCount, numSymbols int) bool {
	if !utf.updateCutFontUint16Table("head", 50, locaFormat) {
		return false
	}
	if !utf.updateCutFontUint16Table("hhea", 34, metricsCount) {
		return false
	}
	if !utf.updateCutFontUint16Table("maxp", 4, numSymbols) {
		return false
	}
	utf.setOutTable("OS/2", utf.getTableData("OS/2"))
	return true
}

func (utf *utf8FontFile) updateCutFontUint16Table(name string, offset, value int) bool {
	data := utf.getTableData(name)
	if utf.err != nil {
		return false
	}
	data = utf.insertUint16(data, offset, value)
	if utf.err != nil {
		return false
	}
	utf.setOutTable(name, data)
	return true
}

func (utf *utf8FontFile) glyphData(symbolData []byte, symbolPos, symbolLen int) ([]byte, bool) {
	if symbolPos < 0 || symbolLen < 0 || symbolPos+symbolLen > len(symbolData) {
		utf.setErrorf("glyph data is truncated")
		return nil, false
	}
	if symbolLen == 1 {
		utf.setErrorf("glyph data is truncated")
		return nil, false
	}
	return symbolData[symbolPos : symbolPos+symbolLen], true
}

func (utf *utf8FontFile) rewriteCompositeGlyph(data []byte, originalSymbolIdx int, symbolArray map[int]int) []byte {
	posInSymbol := 10
	flags := symbolContinue
	for flags&symbolContinue != 0 {
		if posInSymbol+4 > len(data) {
			utf.setErrorf("composite glyph data is truncated")
			return data
		}

		flags = unpackUint16(data[posInSymbol : posInSymbol+2])
		symbolIdx := unpackUint16(data[posInSymbol+2 : posInSymbol+4])

		componentEnd := posInSymbol + 4
		if flags&symbolWords != 0 {
			componentEnd += 4
		} else {
			componentEnd += 2
		}
		switch {
		case flags&symbolScale != 0:
			componentEnd += 2
		case flags&symbolAllScale != 0:
			componentEnd += 4
		case flags&symbol2x2 != 0:
			componentEnd += 8
		}
		if componentEnd > len(data) {
			utf.setErrorf("composite glyph data is truncated")
			return data
		}

		if utf.symbolData == nil {
			utf.symbolData = make(map[int]map[string][]int)
		}
		if _, ok := utf.symbolData[originalSymbolIdx]; !ok {
			utf.symbolData[originalSymbolIdx] = make(map[string][]int)
		}
		if _, ok := utf.symbolData[originalSymbolIdx]["compSymbols"]; !ok {
			utf.symbolData[originalSymbolIdx]["compSymbols"] = make([]int, 0)
		}
		utf.symbolData[originalSymbolIdx]["compSymbols"] = append(utf.symbolData[originalSymbolIdx]["compSymbols"], symbolIdx)
		data = utf.insertUint16(data, posInSymbol+2, symbolArray[symbolIdx])
		posInSymbol = componentEnd
	}
	return data
}

func (utf *utf8FontFile) getSymbols(
	originalSymbolIdx int,
	start *int,
	symbolSet map[int]int,
	symbolsCollection map[int]int,
	symbolsCollectionKeys []int,
) (map[int]int, map[int]int, []int) {
	if originalSymbolIdx < 0 || originalSymbolIdx+1 >= len(utf.symbolPosition) {
		utf.setErrorf("glyph location index %d is out of range", originalSymbolIdx)
		return symbolSet, symbolsCollection, symbolsCollectionKeys
	}
	symbolPos := utf.symbolPosition[originalSymbolIdx]
	symbolSize := utf.symbolPosition[originalSymbolIdx+1] - symbolPos
	if symbolSize == 0 {
		return symbolSet, symbolsCollection, symbolsCollectionKeys
	}
	utf.seek(*start + symbolPos)

	lineCount := utf.readInt16()

	if lineCount < 0 {
		utf.skip(8)
		flags := symbolContinue
		for flags&symbolContinue != 0 {
			flags = utf.readUint16()
			symbolIndex := utf.readUint16()
			if _, OK := symbolSet[symbolIndex]; !OK {
				symbolSet[symbolIndex] = len(symbolsCollection)
				symbolsCollection[symbolIndex] = 1
				symbolsCollectionKeys = append(symbolsCollectionKeys, symbolIndex)
			}
			oldPosition, _ := utf.fileReader.seek(0, 1)
			_, _, symbolsCollectionKeys = utf.getSymbols(
				symbolIndex,
				start,
				symbolSet,
				symbolsCollection,
				symbolsCollectionKeys,
			)
			utf.seek(int(oldPosition))
			if flags&symbolWords != 0 {
				utf.skip(4)
			} else {
				utf.skip(2)
			}
			switch {
			case flags&symbolScale != 0:
				utf.skip(2)
			case flags&symbolAllScale != 0:
				utf.skip(4)
			case flags&symbol2x2 != 0:
				utf.skip(8)
			}
		}
	}
	return symbolSet, symbolsCollection, symbolsCollectionKeys
}

func (utf *utf8FontFile) parseHMTXTable(numberOfHMetrics, numSymbols int, symbolToChar map[int][]int, scale float64) {
	start := utf.seekTable("hmtx")
	arrayWidths := 0
	var arr []int
	utf.charWidths = make([]int, 256*256)
	utf.charWidthExtra = make(map[int]int)
	charCount := 0
	arr = unpackUint16Array(utf.getRange(start, numberOfHMetrics*4))
	if utf.err != nil {
		return
	}
	if len(arr) <= numberOfHMetrics*2 {
		utf.setErrorf("hmtx table is truncated")
		return
	}
	for symbol := range numberOfHMetrics {
		arrayWidths = arr[(symbol*2)+1]
		charCount += utf.applyHMTXWidth(symbol, arrayWidths, symbolToChar, scale)
	}
	diff := numSymbols - numberOfHMetrics
	for pos := range diff {
		symbol := pos + numberOfHMetrics
		charCount += utf.applyHMTXWidth(symbol, arrayWidths, symbolToChar, scale)
	}
	utf.charWidths[0] = charCount
}

func (utf *utf8FontFile) applyHMTXWidth(symbol, arrayWidth int, symbolToChar map[int][]int, scale float64) int {
	chars, ok := symbolToChar[symbol]
	if !ok && symbol != 0 {
		return 0
	}
	if arrayWidth >= (1 << 15) {
		arrayWidth = 0
	}
	if symbol == 0 {
		utf.defaultWidth = scale * float64(arrayWidth)
		return 0
	}

	width := scaledGlyphWidth(arrayWidth, scale)
	count := 0
	for _, char := range chars {
		if char == 0 || char == 65535 {
			continue
		}
		utf.setCharWidth(char, width)
		count++
	}
	return count
}

func scaledGlyphWidth(arrayWidth int, scale float64) int {
	width := int(math.Round(scale * float64(arrayWidth)))
	if width == 0 {
		return 65535
	}
	return width
}

func (utf *utf8FontFile) setCharWidth(char, width int) {
	if char >= 0 && char < len(utf.charWidths) {
		utf.charWidths[char] = width
		return
	}
	if utf.charWidthExtra == nil {
		utf.charWidthExtra = make(map[int]int)
	}
	utf.charWidthExtra[char] = width
}

func (utf *utf8FontFile) getMetrics(metricCount, gid int) []byte {
	start := utf.seekTable("hmtx")
	var metrics []byte
	if gid < metricCount {
		utf.seek(start + (gid * 4))
		metrics = utf.readBytes(4)
	} else {
		utf.seek(start + ((metricCount - 1) * 4))
		metrics = utf.readBytes(2)
		utf.seek(start + (metricCount * 2) + (gid * 2))
		metrics = append(metrics, utf.readBytes(2)...)
	}
	return metrics
}

func (utf *utf8FontFile) parseLOCATable(format, numSymbols int) {
	start := utf.seekTable("loca")
	utf.symbolPosition = make([]int, 0)
	switch format {
	case 0:
		data := utf.getRange(start, (numSymbols*2)+2)
		if utf.err != nil {
			return
		}
		arr := unpackUint16Array(data)
		if len(arr) <= numSymbols+1 {
			utf.setErrorf("loca table is truncated")
			return
		}
		for n := 0; n <= numSymbols; n++ {
			utf.symbolPosition = append(utf.symbolPosition, arr[n+1]*2)
		}
	case 1:
		data := utf.getRange(start, (numSymbols*4)+4)
		if utf.err != nil {
			return
		}
		arr := unpackUint32Array(data)
		if len(arr) <= numSymbols+1 {
			utf.setErrorf("loca table is truncated")
			return
		}
		for n := 0; n <= numSymbols; n++ {
			utf.symbolPosition = append(utf.symbolPosition, arr[n+1])
		}
	default:
		utf.setErrorf("unknown loca table format %d", format)
	}
}

func (utf *utf8FontFile) generateSCCSDictionaries(
	runeCmapPosition int,
	symbolCharDictionary map[int][]int,
	charSymbolDictionary map[int]int,
) {
	utf.seek(runeCmapPosition)
	format := utf.readUint16()
	if format == 12 {
		utf.skip(2)
		_ = utf.readUint32()
		utf.skip(4)
		groupCount := utf.readUint32()
		for range groupCount {
			startCharCode := utf.readUint32()
			endCharCode := utf.readUint32()
			startGlyphID := utf.readUint32()
			if utf.err != nil {
				return
			}
			for char := startCharCode; char <= endCharCode; char++ {
				symbol := startGlyphID + (char - startCharCode)
				charSymbolDictionary[char] = symbol
				symbolCharDictionary[symbol] = append(symbolCharDictionary[symbol], char)
			}
		}
		return
	}

	maxRune := 0
	utf.seek(runeCmapPosition + 2)
	size := utf.readUint16()
	rim := runeCmapPosition + size
	utf.skip(2)

	segmentSize := utf.readUint16() / 2
	utf.skip(6)
	completers := make([]int, 0)
	for range segmentSize {
		completers = append(completers, utf.readUint16())
	}
	utf.skip(2)
	beginners := make([]int, 0)
	for range segmentSize {
		beginners = append(beginners, utf.readUint16())
	}
	sizes := make([]int, 0)
	for range segmentSize {
		sizes = append(sizes, int(utf.readInt16()))
	}
	readerPositionStart := utf.fileReader.readerPosition
	positions := make([]int, 0)
	for range segmentSize {
		positions = append(positions, utf.readUint16())
	}
	for n := range segmentSize {
		completePosition := completers[n] + 1
		for char := beginners[n]; char < completePosition; char++ {
			symbol := utf.cmapFormat4Symbol(char, n, readerPositionStart, rim, beginners, sizes, positions)
			charSymbolDictionary[char] = symbol
			if char < 196608 {
				maxRune = max(char, maxRune)
			}
			symbolCharDictionary[symbol] = append(symbolCharDictionary[symbol], char)
		}
	}
}

func (utf *utf8FontFile) cmapFormat4Symbol(
	char, segment int,
	readerPositionStart int64,
	rim int,
	beginners, sizes, positions []int,
) int {
	if positions[segment] == 0 {
		return (char + sizes[segment]) & 0xFFFF
	}
	position := (char-beginners[segment])*2 + positions[segment]
	position = int(readerPositionStart) + 2*segment + position
	if position >= rim {
		return 0
	}
	symbol := utf.getUint16(position)
	if symbol != 0 {
		return (symbol + sizes[segment]) & 0xFFFF
	}
	return 0
}

func (utf *utf8FontFile) assembleTables() []byte {
	answer := make([]byte, 0)
	tablesCount := len(utf.outTablesData)
	findSize := 1
	writer := 0
	for findSize*2 <= tablesCount {
		findSize *= 2
		writer++
	}
	findSize *= 16
	rOffset := tablesCount*16 - findSize

	answer = append(answer, packHeader(0x00010000, tablesCount, findSize, writer, rOffset)...)

	tables := utf.outTablesData
	tablesNames := keySortStrings(tables)

	offset := 12 + tablesCount*16
	begin := 0

	for _, name := range tablesNames {
		if name == "head" {
			begin = offset
		}
		answer = append(answer, []byte(name)...)
		checksum := utf.generateChecksum(tables[name])
		answer = append(answer, pack2Uint16(checksum[0], checksum[1])...)
		answer = append(answer, pack2Uint32(offset, len(tables[name]))...)
		paddedLength := (len(tables[name]) + 3) &^ 3
		offset += paddedLength
	}

	for _, key := range tablesNames {
		data := append([]byte{}, tables[key]...)
		data = append(data, []byte{0, 0, 0}...)
		answer = append(answer, data[:(len(data)&^3)]...)
	}

	checksum := utf.generateChecksum(answer)
	checksum = utf.calcInt32([]int{0xB1B0, 0xAFBA}, checksum)
	answer = utf.splice(answer, (begin + 8), pack2Uint16(checksum[0], checksum[1]))
	return answer
}
