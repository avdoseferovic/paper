package code

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

var errInvalidQR = errors.New("invalid QR code")

// matrixBarcode is a one-module-per-pixel, black-and-white image. Matrix-code
// encoders deliberately return modules rather than a pre-scaled bitmap so the
// caller can preserve square module geometry in the generated PNG.
type matrixBarcode struct {
	modules [][]bool
}

func (barcode *matrixBarcode) ColorModel() color.Model { return color.GrayModel }

func (barcode *matrixBarcode) Bounds() image.Rectangle {
	if len(barcode.modules) == 0 {
		return image.Rectangle{}
	}
	return image.Rect(0, 0, len(barcode.modules[0]), len(barcode.modules))
}

func (barcode *matrixBarcode) At(x, y int) color.Color {
	if y < 0 || y >= len(barcode.modules) || x < 0 || x >= len(barcode.modules[y]) || !barcode.modules[y][x] {
		return color.White
	}
	return color.Black
}

// encodeQR encodes arbitrary bytes using QR Model 2 byte mode and error
// correction level M. The encoder chooses the smallest version that holds the
// input, writes standard Reed-Solomon blocks, and evaluates all eight masks.
func encodeQR(value string) (image.Image, error) {
	data := []byte(value)
	version := qrVersionForByteData(len(data))
	if version == 0 {
		return nil, fmt.Errorf("%w: payload of %d bytes exceeds level M capacity", errInvalidQR, len(data))
	}

	dataCodewords, err := qrByteDataCodewords(data, version)
	if err != nil {
		return nil, err
	}
	codewords := qrAddErrorCorrection(dataCodewords, version)
	matrix := newQRMatrix(version)
	matrix.drawFunctionPatterns()
	matrix.drawCodewords(codewords)

	bestMask := 0
	bestPenalty := int(^uint(0) >> 1)
	for mask := range 8 {
		matrix.applyMask(mask)
		matrix.drawFormatBits(mask)
		penalty := matrix.penalty()
		if penalty < bestPenalty {
			bestPenalty = penalty
			bestMask = mask
		}
		matrix.applyMask(mask)
	}
	matrix.applyMask(bestMask)
	matrix.drawFormatBits(bestMask)

	return &matrixBarcode{modules: matrix.modules}, nil
}

func qrVersionForByteData(length int) int {
	for version := 1; version <= 40; version++ {
		countBits := 8
		if version >= 10 {
			countBits = 16
		}
		if length >= 1<<countBits {
			continue
		}
		if 4+countBits+length*8 <= qrDataCodewords(version)*8 {
			return version
		}
	}
	return 0
}

func qrByteDataCodewords(data []byte, version int) ([]byte, error) {
	capacity := qrDataCodewords(version)
	countBits := 8
	if version >= 10 {
		countBits = 16
	}
	if 4+countBits+len(data)*8 > capacity*8 {
		return nil, fmt.Errorf("%w: payload does not fit version %d", errInvalidQR, version)
	}

	bits := qrBitBuffer{}
	bits.append(0x4, 4)                       // Byte mode.
	bits.append(uint32(len(data)), countBits) // #nosec G115 -- QR byte capacity is at most 2,953 bytes.
	for _, b := range data {
		bits.append(uint32(b), 8)
	}
	bits.append(0, min(4, capacity*8-bits.len()))
	for bits.len()%8 != 0 {
		bits.append(0, 1)
	}
	result := append([]byte(nil), bits.bytes...)
	for pad := byte(0xEC); len(result) < capacity; pad ^= 0xEC ^ 0x11 {
		result = append(result, pad)
	}
	return result, nil
}

type qrBitBuffer struct {
	bytes []byte
	bits  int
}

func (buffer *qrBitBuffer) append(value uint32, length int) {
	for i := length - 1; i >= 0; i-- {
		if buffer.bits%8 == 0 {
			buffer.bytes = append(buffer.bytes, 0)
		}
		if value&(1<<i) != 0 {
			buffer.bytes[len(buffer.bytes)-1] |= 1 << (7 - buffer.bits%8)
		}
		buffer.bits++
	}
}

func (buffer *qrBitBuffer) len() int { return buffer.bits }

func qrAddErrorCorrection(data []byte, version int) []byte {
	blocks := qrMediumBlocks[version]
	eccLen := qrMediumECCCodewords[version]
	rawCodewords := qrRawDataModules(version) / 8
	shortBlockLen := rawCodewords / blocks
	numShortBlocks := blocks - rawCodewords%blocks
	shortDataLen := shortBlockLen - eccLen
	divisor := qrReedSolomonDivisor(eccLen)

	encodedBlocks := make([][]byte, blocks)
	position := 0
	for block := range blocks {
		dataLen := shortDataLen
		if block >= numShortBlocks {
			dataLen++
		}
		blockData := append([]byte(nil), data[position:position+dataLen]...)
		position += dataLen
		ecc := qrReedSolomonRemainder(blockData, divisor)
		if block < numShortBlocks {
			blockData = append(blockData, 0) // aligns short blocks for interleaving.
		}
		encodedBlocks[block] = append(blockData, ecc...)
	}

	result := make([]byte, 0, rawCodewords)
	for i := range len(encodedBlocks[0]) {
		for block := range encodedBlocks {
			if i != shortDataLen || block >= numShortBlocks {
				result = append(result, encodedBlocks[block][i])
			}
		}
	}
	return result
}

func qrReedSolomonDivisor(degree int) []byte {
	divisor := make([]byte, degree)
	divisor[degree-1] = 1
	root := byte(1)
	for range degree {
		for i := range divisor {
			divisor[i] = qrGaloisMultiply(divisor[i], root)
			if i+1 < len(divisor) {
				divisor[i] ^= divisor[i+1]
			}
		}
		root = qrGaloisMultiply(root, 0x02)
	}
	return divisor
}

func qrReedSolomonRemainder(data, divisor []byte) []byte {
	remainder := make([]byte, len(divisor))
	for _, value := range data {
		factor := value ^ remainder[0]
		copy(remainder, remainder[1:])
		remainder[len(remainder)-1] = 0
		for i := range remainder {
			remainder[i] ^= qrGaloisMultiply(divisor[i], factor)
		}
	}
	return remainder
}

func qrGaloisMultiply(x, y byte) byte {
	result := byte(0)
	for range 8 {
		if y&1 != 0 {
			result ^= x
		}
		y >>= 1
		carry := x & 0x80
		x <<= 1
		if carry != 0 {
			x ^= 0x1D
		}
	}
	return result
}

type qrMatrix struct {
	version  int
	modules  [][]bool
	function [][]bool
}

func newQRMatrix(version int) *qrMatrix {
	size := version*4 + 17
	modules := make([][]bool, size)
	function := make([][]bool, size)
	for i := range modules {
		modules[i] = make([]bool, size)
		function[i] = make([]bool, size)
	}
	return &qrMatrix{version: version, modules: modules, function: function}
}

func (matrix *qrMatrix) size() int { return len(matrix.modules) }

func (matrix *qrMatrix) setFunction(x, y int, dark bool) {
	matrix.modules[y][x] = dark
	matrix.function[y][x] = true
}

func (matrix *qrMatrix) drawFunctionPatterns() {
	size := matrix.size()
	for i := range size {
		matrix.setFunction(6, i, i%2 == 0)
		matrix.setFunction(i, 6, i%2 == 0)
	}
	matrix.drawFinderPattern(3, 3)
	matrix.drawFinderPattern(size-4, 3)
	matrix.drawFinderPattern(3, size-4)

	positions := qrAlignmentPatternPositions(matrix.version)
	for i, x := range positions {
		for j, y := range positions {
			if (i == 0 && j == 0) || (i == 0 && j == len(positions)-1) || (i == len(positions)-1 && j == 0) {
				continue
			}
			matrix.drawAlignmentPattern(x, y)
		}
	}
	matrix.drawFormatBits(0)
	matrix.drawVersion()
}

func (matrix *qrMatrix) drawFinderPattern(x, y int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			xx, yy := x+dx, y+dy
			if xx < 0 || yy < 0 || xx >= matrix.size() || yy >= matrix.size() {
				continue
			}
			distance := max(abs(dx), abs(dy))
			matrix.setFunction(xx, yy, distance != 2 && distance != 4)
		}
	}
}

func (matrix *qrMatrix) drawAlignmentPattern(x, y int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			matrix.setFunction(x+dx, y+dy, max(abs(dx), abs(dy)) != 1)
		}
	}
}

func (matrix *qrMatrix) drawFormatBits(mask int) {
	// Level M's format indicator is binary 00.
	data := mask
	remainder := data
	for range 10 {
		remainder = (remainder << 1) ^ ((remainder >> 9) * 0x537)
	}
	bits := ((data << 10) | remainder) ^ 0x5412

	for i := range 6 {
		matrix.setFunction(8, i, qrBit(bits, i))
	}
	matrix.setFunction(8, 7, qrBit(bits, 6))
	matrix.setFunction(8, 8, qrBit(bits, 7))
	matrix.setFunction(7, 8, qrBit(bits, 8))
	for i := 9; i < 15; i++ {
		matrix.setFunction(14-i, 8, qrBit(bits, i))
	}

	for i := range 8 {
		matrix.setFunction(matrix.size()-1-i, 8, qrBit(bits, i))
	}
	for i := 8; i < 15; i++ {
		matrix.setFunction(8, matrix.size()-15+i, qrBit(bits, i))
	}
	matrix.setFunction(8, matrix.size()-8, true)
}

func (matrix *qrMatrix) drawVersion() {
	if matrix.version < 7 {
		return
	}
	remainder := matrix.version
	for range 12 {
		remainder = (remainder << 1) ^ ((remainder >> 11) * 0x1F25)
	}
	bits := matrix.version<<12 | remainder
	for i := range 18 {
		bit := qrBit(bits, i)
		a := matrix.size() - 11 + i%3
		b := i / 3
		matrix.setFunction(a, b, bit)
		matrix.setFunction(b, a, bit)
	}
}

func (matrix *qrMatrix) drawCodewords(codewords []byte) {
	bit := 0
	for right := matrix.size() - 1; right >= 1; right -= 2 {
		if right == 6 {
			right--
		}
		for vertical := range matrix.size() {
			y := vertical
			if ((right + 1) & 2) == 0 {
				y = matrix.size() - 1 - vertical
			}
			for column := range 2 {
				x := right - column
				if matrix.function[y][x] || bit >= len(codewords)*8 {
					continue
				}
				matrix.modules[y][x] = (codewords[bit>>3]>>(7-uint(bit&7)))&1 != 0
				bit++
			}
		}
	}
}

func (matrix *qrMatrix) applyMask(mask int) {
	for y := range matrix.modules {
		for x := range matrix.modules[y] {
			if !matrix.function[y][x] && qrMask(mask, x, y) {
				matrix.modules[y][x] = !matrix.modules[y][x]
			}
		}
	}
}

func (matrix *qrMatrix) penalty() int {
	penalty := 0
	for y := range matrix.modules {
		penalty += qrRunPenalty(matrix.modules[y])
	}
	for x := range matrix.size() {
		column := make([]bool, matrix.size())
		for y := range column {
			column[y] = matrix.modules[y][x]
		}
		penalty += qrRunPenalty(column)
	}
	for y := 0; y+1 < matrix.size(); y++ {
		for x := 0; x+1 < matrix.size(); x++ {
			module := matrix.modules[y][x]
			rightMatches := module == matrix.modules[y][x+1]
			bottomMatches := module == matrix.modules[y+1][x]
			diagonalMatches := module == matrix.modules[y+1][x+1]
			if rightMatches && bottomMatches && diagonalMatches {
				penalty += 3
			}
		}
	}
	for y := range matrix.modules {
		penalty += qrFinderPenalty(matrix.modules[y])
	}
	for x := range matrix.size() {
		column := make([]bool, matrix.size())
		for y := range column {
			column[y] = matrix.modules[y][x]
		}
		penalty += qrFinderPenalty(column)
	}
	dark := 0
	for _, row := range matrix.modules {
		for _, value := range row {
			if value {
				dark++
			}
		}
	}
	total := matrix.size() * matrix.size()
	penalty += abs(dark*20-total*10) / total * 10
	return penalty
}

func qrRunPenalty(line []bool) int {
	penalty, run := 0, 1
	for i := 1; i < len(line); i++ {
		if line[i] == line[i-1] {
			run++
			continue
		}
		if run >= 5 {
			penalty += 3 + run - 5
		}
		run = 1
	}
	if run >= 5 {
		penalty += 3 + run - 5
	}
	return penalty
}

func qrFinderPenalty(line []bool) int {
	penalty := 0
	for i := 0; i+10 < len(line); i++ {
		if qrPatternAt(line, i, [11]bool{true, false, true, true, true, false, true, false, false, false, false}) {
			penalty += 40
		}
		if qrPatternAt(line, i, [11]bool{false, false, false, false, true, false, true, true, true, false, true}) {
			penalty += 40
		}
	}
	return penalty
}

func qrPatternAt(line []bool, start int, pattern [11]bool) bool {
	for index, value := range pattern {
		if line[start+index] != value {
			return false
		}
	}
	return true
}

func qrMask(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (x/3+y/2)%2 == 0
	case 5:
		return x*y%2+x*y%3 == 0
	case 6:
		return (x*y%2+x*y%3)%2 == 0
	case 7:
		return ((x*y)%3+(x+y)%2)%2 == 0
	default:
		panic("invalid QR mask")
	}
}

func qrAlignmentPatternPositions(version int) []int {
	if version == 1 {
		return nil
	}
	count := version/7 + 2
	step := 0
	if version == 32 {
		step = 26
	} else {
		step = (version*4 + count*2 + 1) / (count*2 - 2) * 2
	}
	positions := make([]int, count)
	positions[0] = 6
	for i := count - 1; i >= 1; i-- {
		positions[i] = version*4 + 10 - (count-1-i)*step
	}
	return positions
}

func qrDataCodewords(version int) int {
	return qrRawDataModules(version)/8 - qrMediumECCCodewords[version]*qrMediumBlocks[version]
}

func qrRawDataModules(version int) int {
	result := (16*version+128)*version + 64
	if version >= 2 {
		alignment := version/7 + 2
		result -= (25*alignment-10)*alignment - 55
		if version >= 7 {
			result -= 36
		}
	}
	return result
}

func qrBit(value, index int) bool { return (value>>index)&1 != 0 }

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// QR Model 2 error-correction level M parameters, indexed by version.
var qrMediumECCCodewords = [...]int{
	-1,
	10, 16, 26, 18, 24, 16, 18, 22, 22, 26,
	30, 22, 22, 24, 24, 28, 28, 28, 26, 26,
	26, 28, 28, 28, 28, 28, 28, 28, 28, 28,
	28, 28, 28, 28, 28, 28, 28, 28, 28, 28,
}

var qrMediumBlocks = [...]int{
	-1,
	1, 1, 1, 2, 2, 4, 4, 4, 5, 5,
	5, 8, 9, 9, 10, 10, 11, 13, 14, 16,
	17, 17, 18, 20, 21, 23, 25, 26, 28, 29,
	31, 33, 35, 37, 38, 40, 43, 45, 47, 49,
}
