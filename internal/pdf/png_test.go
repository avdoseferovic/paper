package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

func TestPNGMalformedCompressionAndFilterErrorsDoNotStartWithQuote(t *testing.T) {
	tests := []struct {
		name        string
		compression byte
		filter      byte
		want        string
	}{
		{
			name:        "compression",
			compression: 1,
			want:        "unknown compression method in PNG buffer",
		},
		{
			name:   "filter",
			filter: 1,
			want:   "unknown filter method in PNG buffer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewCustom(&InitType{})

			f.parsepngstream(bytes.NewBuffer(malformedPNGHeader(tt.compression, tt.filter)), false)

			if f.Error() == nil {
				t.Fatal("expected PNG parse error")
			}
			got := f.Error().Error()
			if strings.HasPrefix(got, "'") {
				t.Fatalf("expected error without leading quote, got %q", got)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPNGParseKeepsFirstError(t *testing.T) {
	f := NewCustom(&InitType{})
	first := errors.New("first error")
	f.SetError(first)

	f.parsepngstream(bytes.NewBuffer([]byte("bad png")), false)

	if !errors.Is(f.Error(), first) {
		t.Fatalf("expected first error to remain, got %v", f.Error())
	}
}

func TestPNGShortTransparencyChunkReturnsErrorWithoutPanic(t *testing.T) {
	f := NewCustom(&InitType{})

	recovered := recoverPNGParse(f, pngWithChunks(0, pngChunk("tRNS", []byte{0}), pngChunk("IEND", nil)))

	if recovered != nil {
		t.Fatalf("expected short tRNS chunk not to panic, got %v", recovered)
	}
	if f.Error() == nil {
		t.Fatal("expected short tRNS chunk to set an error")
	}
}

func TestPNGShortPhysicalChunkReturnsErrorWithoutPanic(t *testing.T) {
	f := NewCustom(&InitType{})

	recovered := recoverPNGParse(f, pngWithChunks(2, pngChunk("pHYs", nil), pngChunk("IEND", nil)))

	if recovered != nil {
		t.Fatalf("expected short pHYs chunk not to panic, got %v", recovered)
	}
	if f.Error() == nil {
		t.Fatal("expected short pHYs chunk to set an error")
	}
}

func TestPNGShortAlphaDataReturnsErrorWithoutPanic(t *testing.T) {
	f := NewCustom(&InitType{})

	recovered := recoverPNGParse(f, pngWithChunks(6, pngChunk("IDAT", zlibData([]byte{0})), pngChunk("IEND", nil)))

	if recovered != nil {
		t.Fatalf("expected short alpha data not to panic, got %v", recovered)
	}
	if f.Error() == nil {
		t.Fatal("expected short alpha data to set an error")
	}
}

func TestTransformKeepsFirstError(t *testing.T) {
	f := NewCustom(&InitType{})
	first := errors.New("first error")
	f.SetError(first)

	f.TransformScale(0, 100, 0, 0)
	f.TransformSkew(90, 0, 0, 0)
	f.Transform(TransformMatrix{})
	f.TransformEnd()

	if !errors.Is(f.Error(), first) {
		t.Fatalf("expected first error to remain, got %v", f.Error())
	}
}

func malformedPNGHeader(compression, filter byte) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\x0d\x0a\x1a\x0a")
	_ = binary.Write(&b, binary.BigEndian, uint32(13))
	b.WriteString("IHDR")
	_ = binary.Write(&b, binary.BigEndian, uint32(1))
	_ = binary.Write(&b, binary.BigEndian, uint32(1))
	b.WriteByte(8)
	b.WriteByte(2)
	b.WriteByte(compression)
	b.WriteByte(filter)
	b.WriteByte(0)
	_ = binary.Write(&b, binary.BigEndian, uint32(0))
	return b.Bytes()
}

func recoverPNGParse(f *PDF, data []byte) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	f.parsepngstream(bytes.NewBuffer(data), true)
	return nil
}

func pngWithChunks(colorType byte, chunks ...[]byte) []byte {
	var b bytes.Buffer
	b.Write(malformedPNGHeader(0, 0))
	// malformedPNGHeader writes only a valid PNG signature and IHDR.
	// Append caller-provided chunks for targeted parser coverage.
	for _, chunk := range chunks {
		b.Write(chunk)
	}
	data := b.Bytes()
	data[25] = colorType
	return data
}

func pngChunk(name string, data []byte) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
	b.WriteString(name)
	b.Write(data)
	_ = binary.Write(&b, binary.BigEndian, uint32(0))
	return b.Bytes()
}

func zlibData(data []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, _ = w.Write(data)
	_ = w.Close()
	return b.Bytes()
}
