package pdf

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// craftPNG builds a structurally valid PNG whose IHDR declares width, height and
// colorType, independent of what idat actually decodes to.
func craftPNG(width, height uint32, colorType byte, idat []byte) []byte {
	var header bytes.Buffer
	_ = binary.Write(&header, binary.BigEndian, width)
	_ = binary.Write(&header, binary.BigEndian, height)
	header.Write([]byte{8, colorType, 0, 0, 0})

	var out bytes.Buffer
	out.WriteString("\x89PNG\x0d\x0a\x1a\x0a")
	out.Write(pngChunk("IHDR", header.Bytes()))
	out.Write(pngChunk("IDAT", idat))
	out.Write(pngChunk("IEND", nil))
	return out.Bytes()
}

func newTestPDF() *PDF {
	return NewCustom(&InitType{OrientationStr: "P", UnitStr: "mm", SizeStr: "A4"})
}

// TestRegisterImageOptionsReader_RejectsMalformedPNGDimensions covers PNGs whose
// IHDR disagrees with the pixel data. The overflow case used to bypass the
// truncation guard (height*(1+4*width) wraps negative) and panic with an index
// out of range while splitting the alpha channel.
func TestRegisterImageOptionsReader_RejectsMalformedPNGDimensions(t *testing.T) {
	t.Parallel()

	idat := sliceCompress(bytes.Repeat([]byte{0}, 64))
	for name, tc := range map[string]struct {
		width, height uint32
		wantErr       string
	}{
		"dimension product overflows": {0x7fffffff, 0x7fffffff, "does not match declared"},
		"width beyond int32":          {0xffffffff, 4, "invalid dimensions"},
		"zero width":                  {0, 4, "invalid dimensions"},
		"zero height":                 {4, 0, "invalid dimensions"},
		"truncated pixel data":        {64, 64, "alpha channel data is truncated"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newTestPDF()
			data := craftPNG(tc.width, tc.height, 6, idat)

			f.RegisterImageOptionsReader(name, ImageOptions{ImageType: "PNG"}, bytes.NewReader(data))

			err := f.Error()
			if err == nil {
				t.Fatalf("RegisterImageOptionsReader() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("RegisterImageOptionsReader() error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// TestRegisterImageOptionsReader_RejectsPNGAlphaBomb keeps a tiny IDAT that
// expands to hundreds of megabytes from being decoded at all.
func TestRegisterImageOptionsReader_RejectsPNGAlphaBomb(t *testing.T) {
	t.Parallel()

	bomb := sliceCompress(make([]byte, 400<<20))
	if len(bomb) > 1<<20 {
		t.Fatalf("compressed bomb = %d bytes, want a small stream", len(bomb))
	}
	f := newTestPDF()
	// Declared dimensions need far more data than this stream can hold.
	data := craftPNG(40_000, 40_000, 6, bomb)

	f.RegisterImageOptionsReader("bomb", ImageOptions{ImageType: "PNG"}, bytes.NewReader(data))

	if f.Error() == nil {
		t.Fatal("RegisterImageOptionsReader() error = nil, want a mismatch error")
	}
}

func TestPNGAlphaRowBytes(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		colorType     byte
		width, height int
		want          int64
		wantOK        bool
	}{
		"gray alpha":       {4, 10, 3, 3 * (1 + 2*10), true},
		"rgba":             {6, 10, 3, 3 * (1 + 4*10), true},
		"zero width":       {6, 0, 3, 0, false},
		"negative height":  {6, 10, -1, 0, false},
		"overflowing size": {6, 0x7fffffff, 0x7fffffff, 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, ok := pngAlphaRowBytes(tc.colorType, tc.width, tc.height)

			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("pngAlphaRowBytes() = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
