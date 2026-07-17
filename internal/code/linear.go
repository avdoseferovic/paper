package code

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

const maxCode128Input = 80

var (
	errInvalidEAN     = errors.New("invalid EAN code")
	errInvalidCode128 = errors.New("invalid Code 128 code")
)

// linearBarcode stores one boolean for each narrowest barcode module. A true
// module is a black bar; a false module is white. Keeping this representation
// before raster scaling prevents a narrow bar from being lost during resizing.
type linearBarcode struct {
	modules []bool
}

func (barcode *linearBarcode) ColorModel() color.Model { return color.GrayModel }

func (barcode *linearBarcode) Bounds() image.Rectangle {
	return image.Rect(0, 0, len(barcode.modules), 1)
}

func (barcode *linearBarcode) At(x, y int) color.Color {
	if y != 0 || x < 0 || x >= len(barcode.modules) || !barcode.modules[x] {
		return color.White
	}
	return color.Black
}

func encodeEAN(value string) (image.Image, error) {
	if len(value) != 7 && len(value) != 8 && len(value) != 12 && len(value) != 13 {
		return nil, fmt.Errorf("%w: expected 7, 8, 12, or 13 digits", errInvalidEAN)
	}
	if !decimalDigits(value) {
		return nil, fmt.Errorf("%w: contains a non-digit", errInvalidEAN)
	}

	payload := value
	if len(value) == 8 || len(value) == 13 {
		payload = value[:len(value)-1]
		if got, want := value[len(value)-1]-'0', eanCheckDigit(payload); got != want {
			return nil, fmt.Errorf("%w: check digit %d, want %d", errInvalidEAN, got, want)
		}
	} else {
		payload += string('0' + eanCheckDigit(payload))
	}

	if len(payload) == 8 {
		return encodeEAN8(payload), nil
	}
	return encodeEAN13(payload), nil
}

func encodeEAN8(value string) image.Image {
	modules := make([]bool, 0, 67)
	modules = appendBinary(modules, "101")
	for i := range 4 {
		modules = appendBinary(modules, eanLeft[value[i]-'0'])
	}
	modules = appendBinary(modules, "01010")
	for i := 4; i < 8; i++ {
		modules = appendBinary(modules, eanRight[value[i]-'0'])
	}
	modules = appendBinary(modules, "101")
	return &linearBarcode{modules: modules}
}

func encodeEAN13(value string) image.Image {
	modules := make([]bool, 0, 95)
	modules = appendBinary(modules, "101")
	parity := ean13Parity[value[0]-'0']
	for i := 1; i <= 6; i++ {
		digit := value[i] - '0'
		if parity[i-1] == 'G' {
			modules = appendBinary(modules, eanEven[digit])
			continue
		}
		modules = appendBinary(modules, eanLeft[digit])
	}
	modules = appendBinary(modules, "01010")
	for i := 7; i < 13; i++ {
		modules = appendBinary(modules, eanRight[value[i]-'0'])
	}
	modules = appendBinary(modules, "101")
	return &linearBarcode{modules: modules}
}

func eanCheckDigit(value string) byte {
	sum := 0
	for i := len(value) - 1; i >= 0; i-- {
		digit := int(value[i] - '0')
		if (len(value)-1-i)%2 == 0 {
			sum += digit * 3
		} else {
			sum += digit
		}
	}
	return byte((10 - sum%10) % 10)
}

func encodeCode128(value string) (image.Image, error) {
	if value == "" {
		return nil, fmt.Errorf("%w: value is empty", errInvalidCode128)
	}
	if len(value) > maxCode128Input {
		return nil, fmt.Errorf("%w: input exceeds %d bytes", errInvalidCode128, maxCode128Input)
	}

	// Paper historically exposes Code 128 through its printable ASCII surface.
	// Code Set B encodes that surface directly and deterministically.
	codewords := make([]int, 0, len(value)+3)
	codewords = append(codewords, 104) // Start Code B.
	for i := range value {
		if value[i] < 32 || value[i] > 126 {
			return nil, fmt.Errorf("%w: byte %d is outside Code Set B", errInvalidCode128, value[i])
		}
		codewords = append(codewords, int(value[i])-32)
	}

	checksum := codewords[0]
	for i := 1; i < len(codewords); i++ {
		checksum += codewords[i] * i
	}
	codewords = append(codewords, checksum%103, 106)

	modules := make([]bool, 0, 11*(len(value)+2)+13)
	for _, codeword := range codewords {
		modules = appendCode128Pattern(modules, code128Patterns[codeword])
	}
	return &linearBarcode{modules: modules}, nil
}

func decimalDigits(value string) bool {
	for i := range value {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func appendBinary(dst []bool, pattern string) []bool {
	for i := range pattern {
		dst = append(dst, pattern[i] == '1')
	}
	return dst
}

func appendCode128Pattern(dst []bool, pattern string) []bool {
	black := true
	for i := range pattern {
		for width := pattern[i] - '0'; width > 0; width-- {
			dst = append(dst, black)
		}
		black = !black
	}
	return dst
}

var eanLeft = [...]string{
	"0001101", "0011001", "0010011", "0111101", "0100011",
	"0110001", "0101111", "0111011", "0110111", "0001011",
}

var eanEven = [...]string{
	"0100111", "0110011", "0011011", "0100001", "0011101",
	"0111001", "0000101", "0010001", "0001001", "0010111",
}

var eanRight = [...]string{
	"1110010", "1100110", "1101100", "1000010", "1011100",
	"1001110", "1010000", "1000100", "1001000", "1110100",
}

var ean13Parity = [...]string{
	"LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG",
	"LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL",
}

var code128Patterns = [...]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213",
	"221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132",
	"221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211",
	"212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331",
	"231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111",
	"314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214",
	"112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141",
	"214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141",
	"114131", "311141", "411131", "211412", "211214", "211232", "2331112",
}
