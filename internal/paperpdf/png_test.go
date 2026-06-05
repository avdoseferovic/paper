package paperpdf

import (
	"bytes"
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

func TestSpotColorKeepsFirstError(t *testing.T) {
	f := NewCustom(&InitType{})
	first := errors.New("first error")
	f.SetError(first)

	f.SetDrawSpotColor("missing", 100)
	f.AddSpotColor("ink", 0, 0, 0, 100)
	f.AddSpotColor("ink", 0, 0, 0, 100)

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
