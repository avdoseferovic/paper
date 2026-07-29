package code

import (
	"errors"
	"fmt"
	"image"
	"slices"
)

var errInvalidDataMatrix = errors.New("invalid Data Matrix code")

// encodeDataMatrix creates an ECC 200 square Data Matrix using ASCII
// encodation. The existing Paper API passes strings, so ASCII digit-pair
// compaction and extended-byte escape handling cover the current surface while
// keeping the byte-level behavior explicit and bounded.
func encodeDataMatrix(value string) (image.Image, error) {
	encoded := dataMatrixASCII([]byte(value))
	symbol, ok := dataMatrixSymbolFor(len(encoded))
	if !ok {
		return nil, fmt.Errorf("%w: payload of %d codewords exceeds square ECC 200 capacity", errInvalidDataMatrix, len(encoded))
	}
	data := dataMatrixPad(encoded, symbol.dataCodewords)
	codewords := dataMatrixAddErrorCorrection(data, symbol)
	placement := newDataMatrixPlacement(symbol.dataRows*symbol.regionRows, symbol.dataCols*symbol.regionCols, codewords)
	placement.place()
	return dataMatrixRender(symbol, placement.bits), nil
}

func dataMatrixASCII(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for index := 0; index < len(data); {
		if index+1 < len(data) && data[index] >= '0' && data[index] <= '9' && data[index+1] >= '0' && data[index+1] <= '9' {
			result = append(result, 130+(data[index]-'0')*10+(data[index+1]-'0'))
			index += 2
			continue
		}
		if data[index] <= 127 {
			result = append(result, data[index]+1)
		} else {
			result = append(result, 235, data[index]-127)
		}
		index++
	}
	return result
}

func dataMatrixSymbolFor(codewords int) (dataMatrixSymbol, bool) {
	for _, symbol := range &dataMatrixSymbols {
		if codewords <= symbol.dataCodewords {
			return symbol, true
		}
	}
	return dataMatrixSymbol{}, false
}

func dataMatrixPad(data []byte, capacity int) []byte {
	result := slices.Clone(data)
	if len(result) < capacity {
		result = append(result, 129)
	}
	for len(result) < capacity {
		position := len(result) + 1 // ECC 200 positions are one-based.
		pad := 129 + ((149*position)%253 + 1)
		if pad > 254 {
			pad -= 254
		}
		result = append(result, byte(pad&0xFF))
	}
	return result
}

func dataMatrixAddErrorCorrection(data []byte, symbol dataMatrixSymbol) []byte {
	blocks := symbol.blocks
	dataBlocks := make([][]byte, blocks)
	eccBlocks := make([][]byte, blocks)
	divisor := dataMatrixReedSolomonDivisor(symbol.eccCodewords / blocks)
	position := 0
	for block := range blocks {
		length := symbol.dataCodewords / blocks
		if block < symbol.dataCodewords%blocks {
			length++
		}
		dataBlocks[block] = slices.Clone(data[position : position+length])
		position += length
		eccBlocks[block] = dataMatrixReedSolomonRemainder(dataBlocks[block], divisor)
	}

	result := make([]byte, 0, symbol.dataCodewords+symbol.eccCodewords)
	for index := 0; ; index++ {
		wrote := false
		for _, block := range dataBlocks {
			if index < len(block) {
				result = append(result, block[index])
				wrote = true
			}
		}
		if !wrote {
			break
		}
	}
	for index := 0; ; index++ {
		wrote := false
		for _, block := range eccBlocks {
			if index < len(block) {
				result = append(result, block[index])
				wrote = true
			}
		}
		if !wrote {
			break
		}
	}
	return result
}

func dataMatrixReedSolomonDivisor(degree int) []byte {
	divisor := make([]byte, degree)
	divisor[degree-1] = 1
	root := byte(2) // ECC 200 uses roots α¹ through αᵈ.
	for range degree {
		for index := range divisor {
			divisor[index] = dataMatrixGaloisMultiply(divisor[index], root)
			if index+1 < len(divisor) {
				divisor[index] ^= divisor[index+1]
			}
		}
		root = dataMatrixGaloisMultiply(root, 2)
	}
	return divisor
}

func dataMatrixReedSolomonRemainder(data, divisor []byte) []byte {
	remainder := make([]byte, len(divisor))
	for _, value := range data {
		factor := value ^ remainder[0]
		copy(remainder, remainder[1:])
		remainder[len(remainder)-1] = 0
		for index := range remainder {
			remainder[index] ^= dataMatrixGaloisMultiply(divisor[index], factor)
		}
	}
	return remainder
}

func dataMatrixGaloisMultiply(x, y byte) byte {
	result := byte(0)
	for range 8 {
		if y&1 != 0 {
			result ^= x
		}
		y >>= 1
		carry := x & 0x80
		x <<= 1
		if carry != 0 {
			x ^= 0x2D
		}
	}
	return result
}

type dataMatrixPlacement struct {
	rows, columns int
	codewords     []byte
	bits          [][]int8 // -1 unset, 0 white, 1 black
}

func newDataMatrixPlacement(rows, columns int, codewords []byte) *dataMatrixPlacement {
	bits := make([][]int8, rows)
	for row := range bits {
		bits[row] = make([]int8, columns)
		for column := range bits[row] {
			bits[row][column] = -1
		}
	}
	return &dataMatrixPlacement{rows: rows, columns: columns, codewords: codewords, bits: bits}
}

func (placement *dataMatrixPlacement) place() {
	row, column, position := 4, 0, 0
	for row < placement.rows || column < placement.columns {
		if row == placement.rows && column == 0 {
			placement.corner1(position)
			position++
		}
		if row == placement.rows-2 && column == 0 && placement.columns%4 != 0 {
			placement.corner2(position)
			position++
		}
		if row == placement.rows-2 && column == 0 && placement.columns%8 == 4 {
			placement.corner3(position)
			position++
		}
		if row == placement.rows+4 && column == 2 && placement.columns%8 == 0 {
			placement.corner4(position)
			position++
		}
		for row >= 0 && column < placement.columns {
			if row < placement.rows && column >= 0 && placement.bits[row][column] < 0 {
				placement.utah(row, column, position)
				position++
			}
			row -= 2
			column += 2
		}
		row++
		column += 3
		for row < placement.rows && column >= 0 {
			if row >= 0 && column < placement.columns && placement.bits[row][column] < 0 {
				placement.utah(row, column, position)
				position++
			}
			row += 2
			column -= 2
		}
		row += 3
		column++
	}
	if placement.bits[placement.rows-1][placement.columns-1] < 0 {
		placement.bits[placement.rows-1][placement.columns-1] = 1
		placement.bits[placement.rows-2][placement.columns-2] = 1
	}
}

func (placement *dataMatrixPlacement) utah(row, column, position int) {
	placement.module(row-2, column-2, position, 1)
	placement.module(row-2, column-1, position, 2)
	placement.module(row-1, column-2, position, 3)
	placement.module(row-1, column-1, position, 4)
	placement.module(row-1, column, position, 5)
	placement.module(row, column-2, position, 6)
	placement.module(row, column-1, position, 7)
	placement.module(row, column, position, 8)
}

func (placement *dataMatrixPlacement) corner1(position int) {
	placement.module(placement.rows-1, 0, position, 1)
	placement.module(placement.rows-1, 1, position, 2)
	placement.module(placement.rows-1, 2, position, 3)
	placement.module(0, placement.columns-2, position, 4)
	placement.module(0, placement.columns-1, position, 5)
	placement.module(1, placement.columns-1, position, 6)
	placement.module(2, placement.columns-1, position, 7)
	placement.module(3, placement.columns-1, position, 8)
}

func (placement *dataMatrixPlacement) corner2(position int) {
	placement.module(placement.rows-3, 0, position, 1)
	placement.module(placement.rows-2, 0, position, 2)
	placement.module(placement.rows-1, 0, position, 3)
	placement.module(0, placement.columns-4, position, 4)
	placement.module(0, placement.columns-3, position, 5)
	placement.module(0, placement.columns-2, position, 6)
	placement.module(0, placement.columns-1, position, 7)
	placement.module(1, placement.columns-1, position, 8)
}

func (placement *dataMatrixPlacement) corner3(position int) {
	placement.module(placement.rows-3, 0, position, 1)
	placement.module(placement.rows-2, 0, position, 2)
	placement.module(placement.rows-1, 0, position, 3)
	placement.module(0, placement.columns-2, position, 4)
	placement.module(0, placement.columns-1, position, 5)
	placement.module(1, placement.columns-1, position, 6)
	placement.module(2, placement.columns-1, position, 7)
	placement.module(3, placement.columns-1, position, 8)
}

func (placement *dataMatrixPlacement) corner4(position int) {
	placement.module(placement.rows-1, 0, position, 1)
	placement.module(placement.rows-1, placement.columns-1, position, 2)
	placement.module(0, placement.columns-3, position, 3)
	placement.module(0, placement.columns-2, position, 4)
	placement.module(0, placement.columns-1, position, 5)
	placement.module(1, placement.columns-3, position, 6)
	placement.module(1, placement.columns-2, position, 7)
	placement.module(1, placement.columns-1, position, 8)
}

func (placement *dataMatrixPlacement) module(row, column, position, bit int) {
	if row < 0 {
		row += placement.rows
		column += 4 - (placement.rows+4)%8
	}
	if column < 0 {
		column += placement.columns
		row += 4 - (placement.columns+4)%8
	}
	value := false
	if position < len(placement.codewords) {
		value = (placement.codewords[position]>>(8-bit))&1 != 0
	}
	if value {
		placement.bits[row][column] = 1
	} else {
		placement.bits[row][column] = 0
	}
}

func dataMatrixRender(symbol dataMatrixSymbol, bits [][]int8) image.Image {
	rows := symbol.regionRows * (symbol.dataRows + 2)
	columns := symbol.regionCols * (symbol.dataCols + 2)
	modules := make([][]bool, rows)
	for row := range modules {
		modules[row] = make([]bool, columns)
	}
	for regionRow := range symbol.regionRows {
		for regionColumn := range symbol.regionCols {
			baseRow := regionRow * (symbol.dataRows + 2)
			baseColumn := regionColumn * (symbol.dataCols + 2)
			for column := range symbol.dataCols + 2 {
				modules[baseRow][baseColumn+column] = column%2 == 0
				modules[baseRow+symbol.dataRows+1][baseColumn+column] = true
			}
			for row := range symbol.dataRows {
				modules[baseRow+row+1][baseColumn] = true
				modules[baseRow+row+1][baseColumn+symbol.dataCols+1] = row%2 == 0
				for column := range symbol.dataCols {
					dataRow := regionRow*symbol.dataRows + row
					dataColumn := regionColumn*symbol.dataCols + column
					modules[baseRow+row+1][baseColumn+column+1] = bits[dataRow][dataColumn] == 1
				}
			}
		}
	}
	return &matrixBarcode{modules: modules}
}

type dataMatrixSymbol struct {
	dataCodewords int
	eccCodewords  int
	regionRows    int
	regionCols    int
	dataRows      int
	dataCols      int
	blocks        int
}

// Square ECC 200 symbols. Rectangular symbols are deliberately excluded
// because the current encoder selects square symbols for Paper's API.
var dataMatrixSymbols = [...]dataMatrixSymbol{
	{3, 5, 1, 1, 8, 8, 1},
	{5, 7, 1, 1, 10, 10, 1},
	{8, 10, 1, 1, 12, 12, 1},
	{12, 12, 1, 1, 14, 14, 1},
	{18, 14, 1, 1, 16, 16, 1},
	{22, 18, 1, 1, 18, 18, 1},
	{30, 20, 1, 1, 20, 20, 1},
	{36, 24, 1, 1, 22, 22, 1},
	{44, 28, 1, 1, 24, 24, 1},
	{62, 36, 2, 2, 14, 14, 1},
	{86, 42, 2, 2, 16, 16, 1},
	{114, 48, 2, 2, 18, 18, 1},
	{144, 56, 2, 2, 20, 20, 1},
	{174, 68, 2, 2, 22, 22, 1},
	{204, 84, 2, 2, 24, 24, 2},
	{280, 112, 4, 4, 14, 14, 2},
	{368, 144, 4, 4, 16, 16, 4},
	{456, 192, 4, 4, 18, 18, 4},
	{576, 224, 4, 4, 20, 20, 4},
	{696, 272, 4, 4, 22, 22, 4},
	{816, 336, 4, 4, 24, 24, 6},
	{1050, 408, 6, 6, 18, 18, 6},
	{1304, 496, 6, 6, 20, 20, 8},
	{1558, 620, 6, 6, 22, 22, 10},
}
