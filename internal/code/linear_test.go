package code

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestEncodeEAN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    string
		wantLen int
	}{
		{
			name:    "EAN-8 adds a check digit",
			value:   "5512345",
			want:    "1010110001011000100110010010011010101000010101110010011101000100101",
			wantLen: 67,
		},
		{
			name:    "EAN-13 adds a check digit",
			value:   "590123412345",
			want:    "10100010110100111011001100100110111101001110101010110011011011001000010101110010011101000100101",
			wantLen: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := encodeEAN(tt.value)
			if err != nil {
				t.Fatalf("encodeEAN(%q) error = %v", tt.value, err)
			}
			if got.Bounds().Dx() != tt.wantLen {
				t.Fatalf("encodeEAN(%q) width = %d, want %d", tt.value, got.Bounds().Dx(), tt.wantLen)
			}
			if modules := linearModules(got); modules != tt.want {
				t.Fatalf("encodeEAN(%q) modules = %s, want %s", tt.value, modules, tt.want)
			}
		})
	}
}

func TestEncodeEANRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "123456", "123456789", "1234567x", "55123456", "5901234123458"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if _, err := encodeEAN(value); err == nil {
				t.Fatalf("encodeEAN(%q) error = nil, want error", value)
			}
		})
	}
}

func TestEncodeCode128(t *testing.T) {
	t.Parallel()

	barcode, err := encodeCode128("Hi")
	if err != nil {
		t.Fatalf("encodeCode128() error = %v", err)
	}
	if got, want := barcode.Bounds().Dx(), 57; got != want {
		t.Fatalf("encodeCode128() width = %d, want %d", got, want)
	}

	if _, err := encodeCode128("\x1f"); err == nil {
		t.Fatal("encodeCode128() accepted a control byte")
	}
	if _, err := encodeCode128(strings.Repeat("A", maxCode128Input+1)); err == nil {
		t.Fatal("encodeCode128() accepted an oversized input")
	}
}

func TestEncodeQR(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		value string
		size  int
	}{
		{value: "HELLO WORLD", size: 21},
		{value: strings.Repeat("a", 50), size: 33},
	} {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			barcode, err := encodeQR(tt.value)
			if err != nil {
				t.Fatalf("encodeQR(%q) error = %v", tt.value, err)
			}
			if got := barcode.Bounds(); got.Dx() != tt.size || got.Dy() != tt.size {
				t.Fatalf("encodeQR(%q) bounds = %v, want %dx%d", tt.value, got, tt.size, tt.size)
			}
		})
	}

	barcode, err := encodeQR("HELLO WORLD")
	if err != nil {
		t.Fatalf("encodeQR() error = %v", err)
	}
	for y := range 7 {
		for x := range 7 {
			wantBlack := max(abs(x-3), abs(y-3)) != 2
			gray := color.GrayModel.Convert(barcode.At(x, y)).(color.Gray)
			if gotBlack := gray.Y == 0; gotBlack != wantBlack {
				t.Fatalf("finder module (%d, %d) black = %t, want %t", x, y, gotBlack, wantBlack)
			}
		}
	}

	if _, err := encodeQR(strings.Repeat("a", 5000)); err == nil {
		t.Fatal("encodeQR() accepted an oversized payload")
	}
}

func TestEncodeDataMatrix(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		value string
		size  int
	}{
		{value: "HELLO WORLD", size: 16},
		{value: strings.Repeat("a", 50), size: 32},
	} {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			barcode, err := encodeDataMatrix(tt.value)
			if err != nil {
				t.Fatalf("encodeDataMatrix(%q) error = %v", tt.value, err)
			}
			if got := barcode.Bounds(); got.Dx() != tt.size || got.Dy() != tt.size {
				t.Fatalf("encodeDataMatrix(%q) bounds = %v, want %dx%d", tt.value, got, tt.size, tt.size)
			}
			for index := range tt.size {
				if gray := color.GrayModel.Convert(barcode.At(0, index)).(color.Gray); gray.Y != 0 {
					t.Fatalf("left finder at row %d is not black", index)
				}
				if gray := color.GrayModel.Convert(barcode.At(index, tt.size-1)).(color.Gray); gray.Y != 0 {
					t.Fatalf("bottom finder at column %d is not black", index)
				}
			}
		})
	}

	if _, err := encodeDataMatrix(strings.Repeat("a", 5000)); err == nil {
		t.Fatal("encodeDataMatrix() accepted an oversized payload")
	}
}

func linearModules(barcode image.Image) string {
	var modules strings.Builder
	modules.Grow(barcode.Bounds().Dx())
	for x := barcode.Bounds().Min.X; x < barcode.Bounds().Max.X; x++ {
		gray := color.GrayModel.Convert(barcode.At(x, barcode.Bounds().Min.Y)).(color.Gray)
		if gray.Y == 0 {
			modules.WriteByte('1')
			continue
		}
		modules.WriteByte('0')
	}
	return modules.String()
}
