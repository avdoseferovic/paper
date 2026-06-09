package pdf

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
)

func int16FromUint16Bits(v uint16) int16 {
	return int16(v) // #nosec G115 -- TrueType stores signed int16 values as uint16 bits.
}

func signedByteValue(v byte) int {
	if v < 0x80 {
		return int(v)
	}
	return int(v) - 0x100
}

// SetColorEmojiEnabled enables or disables color emoji rendering for COLR/CPAL,
// CBDT/CBLC, and sbix emoji fonts. When disabled, emoji-capable outline fonts
// render through the normal monochrome text path.
func (f *PDF) SetColorEmojiEnabled(enabled bool) {
	f.colorEmojiEnabled = enabled
}

// HasColorEmoji reports whether color emoji rendering is enabled for the
// current font.
func (f *PDF) HasColorEmoji() bool {
	return f.colorEmojiEnabled && f.currentFont.hasColorGlyphs
}

func (f *PDF) stringToCIDs(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		cid := f.getOrAssignCID(int(r))
		cid16, ok := checkedUint16(cid)
		if !ok {
			continue
		}
		b.WriteByte(byte(cid16 >> 8)) // #nosec G115 -- high byte of a uint16 CID.
		b.WriteByte(byte(cid16))      // #nosec G115 -- low byte of a uint16 CID.
	}
	return b.String()
}

func (f *PDF) getOrAssignCID(r int) int {
	if f.currentFont.runeToCID == nil {
		f.currentFont.runeToCID = make(map[int]int)
	}
	if f.currentFont.usedRunes == nil {
		f.currentFont.usedRunes = make(map[int]int)
	}
	if cid, ok := f.currentFont.runeToCID[r]; ok {
		f.currentFont.usedRunes[cid] = r
		return cid
	}

	cid := r
	if r > 0xFFFF {
		cid = f.findNextFreeCID()
	} else if original, used := f.currentFont.usedRunes[r]; used && original != r {
		cid = f.findNextFreeCID()
	}

	f.currentFont.runeToCID[r] = cid
	f.currentFont.usedRunes[cid] = r
	return cid
}

func (f *PDF) findNextFreeCID() int {
	for cid := 0xE000; cid <= 0xF8FF; cid++ {
		if _, used := f.currentFont.usedRunes[cid]; !used {
			return cid
		}
	}
	for cid := 32; cid <= 0xFFFF; cid++ {
		if _, used := f.currentFont.usedRunes[cid]; !used {
			return cid
		}
	}
	return 0
}

func (f *PDF) isColorGlyph(r rune) bool {
	if !f.HasColorEmoji() || f.currentFont.utf8File == nil {
		return false
	}
	glyphID, ok := f.currentFont.utf8File.charSymbolDictionary[int(r)]
	if !ok {
		return false
	}
	glyphID16, ok := checkedUint16(glyphID)
	if !ok {
		return false
	}
	if f.currentFont.utf8File.bitmapGlyphImage(glyphID16, f.fontSizePt) != nil {
		return true
	}
	return f.currentFont.utf8File.colorGlyphLayers(glyphID16) != nil
}

func (f *PDF) textContainsColorEmoji(txtStr string) bool {
	for _, r := range txtStr {
		if f.isColorGlyph(r) {
			return true
		}
	}
	return false
}

func (f *PDF) textWithColorEmoji(x, y float64, txtStr string) {
	if f.isRTL {
		txtStr = reverseText(txtStr)
		x -= f.GetStringWidth(txtStr)
	}

	var s strings.Builder
	currentX := x
	for _, r := range txtStr {
		charWidth := f.GetStringWidth(string(r))
		if f.isColorGlyph(r) {
			s.WriteString(f.colorGlyphTextSegment(r, currentX, y))
		} else {
			s.WriteString(f.monochromeGlyphTextSegment(r, currentX, y))
		}
		currentX += charWidth
	}

	if f.underline && txtStr != "" {
		s.WriteByte(' ')
		s.WriteString(f.dounderline(x, y, txtStr))
	}
	if f.strikeout && txtStr != "" {
		s.WriteByte(' ')
		s.WriteString(f.dostrikeout(x, y, txtStr))
	}
	f.out(s.String())
}

func (f *PDF) colorGlyphTextSegment(r rune, x, y float64) string {
	var s strings.Builder
	if colorPath := f.renderColorGlyph(r, x, y); colorPath != "" {
		s.WriteString(colorPath)
	}
	if f.currentFont.Tp == fontTypeUTF8Bitmap {
		return s.String()
	}
	txt := f.escape(f.stringToCIDs(string(r)))
	s.WriteString(sprintf("q 3 Tr BT %.2f %.2f Td (%s) Tj ET Q ", x*f.k, (f.h-y)*f.k, txt))
	return s.String()
}

func (f *PDF) monochromeGlyphTextSegment(r rune, x, y float64) string {
	if f.currentFont.Tp == fontTypeUTF8Bitmap {
		return ""
	}
	txt := f.escape(f.stringToCIDs(string(r)))
	textOp := sprintf("BT %.2f %.2f Td (%s) Tj ET", x*f.k, (f.h-y)*f.k, txt)
	if f.colorFlag {
		textOp = sprintf("q %s %s Q", f.color.text.str, textOp)
	}
	return textOp + " "
}

func (f *PDF) renderColorGlyph(r rune, x, y float64) string {
	if f.currentFont.utf8File == nil {
		return ""
	}
	glyphID, ok := f.currentFont.utf8File.charSymbolDictionary[int(r)]
	if !ok {
		return ""
	}
	glyphID16, ok := checkedUint16(glyphID)
	if !ok {
		return ""
	}
	if bitmapGlyph := f.currentFont.utf8File.bitmapGlyphImage(glyphID16, f.fontSizePt); bitmapGlyph != nil {
		return f.renderBitmapGlyph(glyphID16, bitmapGlyph, x, y)
	}
	renderer := colorEmojiRenderer{
		utf8File:   f.currentFont.utf8File,
		unitsPerEm: f.currentFont.utf8File.fontElementSize,
	}
	return renderer.renderColorGlyph(glyphID16, x, f.h-y, f.fontSize, f.k)
}

func (f *PDF) renderBitmapGlyph(glyphID uint16, glyph *bitmapGlyphImage, x, baselineY float64) string {
	if glyph == nil || len(glyph.data) == 0 || glyph.width == 0 || glyph.height == 0 {
		return ""
	}
	ppemX := glyph.ppemX
	if ppemX == 0 {
		ppemX = glyph.width
	}
	ppemY := glyph.ppemY
	if ppemY == 0 {
		ppemY = glyph.height
	}

	scaleX := f.fontSize / float64(ppemX)
	scaleY := f.fontSize / float64(ppemY)
	var drawX, drawY float64
	if glyph.advance > 0 || glyph.bearingY != 0 || glyph.bearingX != 0 {
		drawX = x + float64(glyph.bearingX)*scaleX
		drawY = baselineY - float64(glyph.bearingY)*scaleY
	} else {
		drawX = x + float64(glyph.originOffsetX)*scaleX
		drawY = baselineY - float64(glyph.originOffsetY+glyph.height)*scaleY
	}
	w := float64(glyph.width) * scaleX
	h := float64(glyph.height) * scaleY

	hash := sha256.Sum256(glyph.data)
	name := fmt.Sprintf("emoji-%s-%d-%x", f.currentFont.Name, glyphID, hash[:6])
	info := f.RegisterImageOptionsReader(name, ImageOptions{ImageType: glyph.imageType}, bytes.NewReader(glyph.data))
	if f.err != nil || info == nil {
		return ""
	}
	return sprintf("q %.5f 0 0 %.5f %.5f %.5f cm /I%s Do Q ", w*f.k, h*f.k, drawX*f.k, (f.h-(drawY+h))*f.k, info.i)
}

type colorEmojiRenderer struct {
	utf8File   *utf8FontFile
	unitsPerEm int
}

func (r colorEmojiRenderer) renderColorGlyph(glyphID uint16, x, y, fontSize, k float64) string {
	if r.utf8File == nil || !r.utf8File.hasColorGlyphs || r.unitsPerEm == 0 {
		return ""
	}
	layers := r.utf8File.colorGlyphLayers(glyphID)
	if len(layers) == 0 {
		return ""
	}

	var result strings.Builder
	scale := fontSize / float64(r.unitsPerEm)
	for _, layer := range layers {
		color := r.utf8File.paletteColor(layer.paletteIndex)
		outline := r.utf8File.parseGlyphOutline(layer.glyphID)
		if outline == nil || len(outline.contours) == 0 {
			continue
		}

		result.WriteString("q ")
		fmt.Fprintf(&result, "%.3f %.3f %.3f rg ", float64(color.r)/255, float64(color.g)/255, float64(color.b)/255)
		result.WriteString(glyphOutlineToPDFPath(outline, x, y, scale, k))
		result.WriteString("f Q ")
	}
	return result.String()
}

type glyphPoint struct {
	x, y    float64
	onCurve bool
}

type glyphContour []glyphPoint

type glyphOutline struct {
	contours []glyphContour
	bounds   [4]int16
}

const (
	glyphOnCurve         = 1 << 0
	glyphXShortVector    = 1 << 1
	glyphYShortVector    = 1 << 2
	glyphRepeat          = 1 << 3
	glyphXSameOrPosShort = 1 << 4
	glyphYSameOrPosShort = 1 << 5
)

func (utf *utf8FontFile) parseGlyphOutline(glyphID uint16) *glyphOutline {
	if len(utf.symbolPosition) == 0 || int(glyphID) >= len(utf.symbolPosition)-1 {
		return nil
	}
	glyfData := utf.getTableData("glyf")
	if glyfData == nil {
		return nil
	}
	symbolPos := utf.symbolPosition[glyphID]
	symbolLen := utf.symbolPosition[glyphID+1] - symbolPos
	if symbolLen < 0 || symbolPos < 0 || symbolPos+symbolLen > len(glyfData) {
		return nil
	}
	if symbolLen == 0 {
		return &glyphOutline{}
	}
	return utf.parseGlyphData(glyfData[symbolPos:symbolPos+symbolLen], glyfData)
}

func (utf *utf8FontFile) parseGlyphData(data []byte, glyfData []byte) *glyphOutline {
	if len(data) < 10 {
		return nil
	}
	numContours := int16FromUint16Bits(binary.BigEndian.Uint16(data[0:2]))
	outline := &glyphOutline{
		bounds: [4]int16{
			int16FromUint16Bits(binary.BigEndian.Uint16(data[2:4])),
			int16FromUint16Bits(binary.BigEndian.Uint16(data[4:6])),
			int16FromUint16Bits(binary.BigEndian.Uint16(data[6:8])),
			int16FromUint16Bits(binary.BigEndian.Uint16(data[8:10])),
		},
	}
	if numContours >= 0 {
		utf.parseSimpleGlyph(data[10:], int(numContours), outline)
		return outline
	}
	utf.parseCompositeGlyph(data[10:], glyfData, outline)
	return outline
}

func (utf *utf8FontFile) parseSimpleGlyph(data []byte, numContours int, outline *glyphOutline) {
	if numContours == 0 || len(data) < numContours*2 {
		return
	}

	endPtsOfContours := make([]uint16, numContours)
	for i := range numContours {
		endPtsOfContours[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}
	numPoints := int(endPtsOfContours[numContours-1]) + 1
	offset := numContours * 2
	if offset+2 > len(data) {
		return
	}
	instructionLength := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2 + instructionLength
	if offset > len(data) {
		return
	}

	flags, offset, ok := readSimpleGlyphFlags(data, offset, numPoints)
	if !ok {
		return
	}
	xCoords, offset, ok := readSimpleGlyphCoords(data, flags, offset, glyphXShortVector, glyphXSameOrPosShort)
	if !ok {
		return
	}
	yCoords, _, ok := readSimpleGlyphCoords(data, flags, offset, glyphYShortVector, glyphYSameOrPosShort)
	if !ok {
		return
	}
	outline.contours = buildSimpleGlyphContours(endPtsOfContours, flags, xCoords, yCoords)
}

func readSimpleGlyphFlags(data []byte, offset, numPoints int) ([]byte, int, bool) {
	flags := make([]byte, numPoints)
	for i := 0; i < numPoints; {
		if offset >= len(data) {
			return nil, offset, false
		}
		flag := data[offset]
		offset++
		flags[i] = flag
		i++
		if flag&glyphRepeat == 0 {
			continue
		}
		if offset >= len(data) {
			return nil, offset, false
		}
		repeatCount := int(data[offset])
		offset++
		for range repeatCount {
			if i >= numPoints {
				break
			}
			flags[i] = flag
			i++
		}
	}
	return flags, offset, true
}

func readSimpleGlyphCoords(data []byte, flags []byte, offset int, shortFlag, sameFlag byte) ([]int, int, bool) {
	coords := make([]int, len(flags))
	current := 0
	for i, flag := range flags {
		delta, nextOffset, ok := readSimpleGlyphCoordDelta(data, offset, flag, shortFlag, sameFlag)
		if !ok {
			return nil, offset, false
		}
		current += delta
		offset = nextOffset
		coords[i] = current
	}
	return coords, offset, true
}

func readSimpleGlyphCoordDelta(data []byte, offset int, flag, shortFlag, sameFlag byte) (int, int, bool) {
	switch {
	case flag&shortFlag != 0:
		if offset >= len(data) {
			return 0, offset, false
		}
		delta := int(data[offset])
		if flag&sameFlag == 0 {
			delta = -delta
		}
		return delta, offset + 1, true
	case flag&sameFlag == 0:
		if offset+2 > len(data) {
			return 0, offset, false
		}
		delta := int(int16FromUint16Bits(binary.BigEndian.Uint16(data[offset : offset+2])))
		return delta, offset + 2, true
	default:
		return 0, offset, true
	}
}

func buildSimpleGlyphContours(endPtsOfContours []uint16, flags []byte, xCoords, yCoords []int) []glyphContour {
	contours := make([]glyphContour, len(endPtsOfContours))
	pointIdx := 0
	for c, endPtRaw := range endPtsOfContours {
		endPt := int(endPtRaw)
		contourLen := endPt - pointIdx + 1
		contour := make(glyphContour, contourLen)
		for i := range contourLen {
			contour[i] = glyphPoint{
				x:       float64(xCoords[pointIdx]),
				y:       float64(yCoords[pointIdx]),
				onCurve: flags[pointIdx]&glyphOnCurve != 0,
			}
			pointIdx++
		}
		contours[c] = contour
	}
	return contours
}

type glyphTransform struct {
	a, b, c, d float64
	e, f       float64
}

func (utf *utf8FontFile) parseCompositeGlyph(data []byte, glyfData []byte, outline *glyphOutline) {
	offset := 0
	flags := uint16(symbolContinue)
	for flags&symbolContinue != 0 {
		if offset+4 > len(data) {
			return
		}
		flags = binary.BigEndian.Uint16(data[offset : offset+2])
		glyphIndex := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		offset += 4

		transform, nextOffset, ok := readCompositeGlyphTransform(data, offset, flags)
		if !ok {
			return
		}
		offset = nextOffset

		if glyphIndex >= len(utf.symbolPosition)-1 {
			continue
		}
		compPos := utf.symbolPosition[glyphIndex]
		compLen := utf.symbolPosition[glyphIndex+1] - compPos
		if compLen <= 0 || compPos < 0 || compPos+compLen > len(glyfData) {
			continue
		}
		compOutline := utf.parseGlyphData(glyfData[compPos:compPos+compLen], glyfData)
		if compOutline == nil {
			continue
		}
		appendTransformedContours(outline, compOutline, transform)
	}
}

func readCompositeGlyphTransform(data []byte, offset int, flags uint16) (glyphTransform, int, bool) {
	arg1, arg2, nextOffset, ok := readCompositeGlyphArgs(data, offset, flags)
	if !ok {
		return glyphTransform{}, offset, false
	}
	transform := glyphTransform{a: 1, d: 1, e: float64(arg1), f: float64(arg2)}
	transform, nextOffset, ok = readCompositeGlyphScale(data, nextOffset, flags, transform)
	return transform, nextOffset, ok
}

func readCompositeGlyphArgs(data []byte, offset int, flags uint16) (int, int, int, bool) {
	if flags&symbolWords != 0 {
		if offset+4 > len(data) {
			return 0, 0, offset, false
		}
		arg1 := int(int16FromUint16Bits(binary.BigEndian.Uint16(data[offset : offset+2])))
		arg2 := int(int16FromUint16Bits(binary.BigEndian.Uint16(data[offset+2 : offset+4])))
		return arg1, arg2, offset + 4, true
	}
	if offset+2 > len(data) {
		return 0, 0, offset, false
	}
	arg1 := signedByteValue(data[offset])
	arg2 := signedByteValue(data[offset+1])
	return arg1, arg2, offset + 2, true
}

func readCompositeGlyphScale(data []byte, offset int, flags uint16, transform glyphTransform) (glyphTransform, int, bool) {
	switch {
	case flags&symbolScale != 0:
		if offset+2 > len(data) {
			return transform, offset, false
		}
		scale := read2Dot14(data[offset : offset+2])
		transform.a, transform.d = scale, scale
		return transform, offset + 2, true
	case flags&symbolAllScale != 0:
		if offset+4 > len(data) {
			return transform, offset, false
		}
		transform.a = read2Dot14(data[offset : offset+2])
		transform.d = read2Dot14(data[offset+2 : offset+4])
		return transform, offset + 4, true
	case flags&symbol2x2 != 0:
		if offset+8 > len(data) {
			return transform, offset, false
		}
		transform.a = read2Dot14(data[offset : offset+2])
		transform.b = read2Dot14(data[offset+2 : offset+4])
		transform.c = read2Dot14(data[offset+4 : offset+6])
		transform.d = read2Dot14(data[offset+6 : offset+8])
		return transform, offset + 8, true
	default:
		return transform, offset, true
	}
}

func appendTransformedContours(outline, compOutline *glyphOutline, transform glyphTransform) {
	for _, contour := range compOutline.contours {
		transformed := make(glyphContour, len(contour))
		for i, pt := range contour {
			transformed[i] = glyphPoint{
				x:       transform.a*pt.x + transform.c*pt.y + transform.e,
				y:       transform.b*pt.x + transform.d*pt.y + transform.f,
				onCurve: pt.onCurve,
			}
		}
		outline.contours = append(outline.contours, transformed)
	}
}

func read2Dot14(data []byte) float64 {
	return float64(int16FromUint16Bits(binary.BigEndian.Uint16(data))) / 16384
}

func glyphOutlineToPDFPath(outline *glyphOutline, x, y, scale, k float64) string {
	if outline == nil || len(outline.contours) == 0 {
		return ""
	}

	var result strings.Builder
	for _, contour := range outline.contours {
		result.WriteString(contourToPDFOps(contour, x, y, scale, k))
	}
	return result.String()
}

func contourToPDFOps(contour glyphContour, baseX, baseY, scale, k float64) string {
	if len(contour) < 2 {
		return ""
	}

	transform := func(pt glyphPoint) (float64, float64) {
		return (baseX + pt.x*scale) * k, (baseY + pt.y*scale) * k
	}

	var result strings.Builder
	startIdx := -1
	for i, pt := range contour {
		if pt.onCurve {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		p0, p1 := contour[0], contour[1]
		px, py := transform(glyphPoint{x: (p0.x + p1.x) / 2, y: (p0.y + p1.y) / 2, onCurve: true})
		fmt.Fprintf(&result, "%.2f %.2f m ", px, py)
		startIdx = 0
	} else {
		px, py := transform(contour[startIdx])
		fmt.Fprintf(&result, "%.2f %.2f m ", px, py)
	}

	n := len(contour)
	i := (startIdx + 1) % n
	for count := 0; count < n; count++ {
		curr := contour[i]
		next := contour[(i+1)%n]
		if curr.onCurve {
			px, py := transform(curr)
			fmt.Fprintf(&result, "%.2f %.2f l ", px, py)
		} else {
			prev := contour[(i-1+n)%n]
			p0x, p0y := prev.x, prev.y
			if !prev.onCurve {
				p0x = (prev.x + curr.x) / 2
				p0y = (prev.y + curr.y) / 2
			}
			p2x, p2y := next.x, next.y
			if !next.onCurve {
				p2x = (curr.x + next.x) / 2
				p2y = (curr.y + next.y) / 2
			}
			c1x := p0x + 2.0/3.0*(curr.x-p0x)
			c1y := p0y + 2.0/3.0*(curr.y-p0y)
			c2x := p2x + 2.0/3.0*(curr.x-p2x)
			c2y := p2y + 2.0/3.0*(curr.y-p2y)
			c1px, c1py := transform(glyphPoint{x: c1x, y: c1y})
			c2px, c2py := transform(glyphPoint{x: c2x, y: c2y})
			epx, epy := transform(glyphPoint{x: p2x, y: p2y})
			fmt.Fprintf(&result, "%.2f %.2f %.2f %.2f %.2f %.2f c ", c1px, c1py, c2px, c2py, epx, epy)
			if next.onCurve {
				i = (i + 1) % n
				count++
			}
		}
		i = (i + 1) % n
	}
	result.WriteString("h ")
	return result.String()
}
