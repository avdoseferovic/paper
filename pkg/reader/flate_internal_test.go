package reader

import (
	"bytes"
	"compress/zlib"
	"strings"
	"testing"
)

func zlibCompressed(t *testing.T, payload []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	_, err := writer.Write(payload)
	if err != nil {
		t.Fatalf("write payload: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return compressed.Bytes()
}

// TestFlateDecodeLimit_RefusesOversizedStream covers a stream that decodes past
// its limit. Without the cap, io.ReadAll allocated whatever the stream expanded
// to: 8MB of zeros ship in 8KB of PDF, and the ratio scales with the payload.
func TestFlateDecodeLimit_RefusesOversizedStream(t *testing.T) {
	t.Parallel()

	compressed := zlibCompressed(t, make([]byte, 8<<20))

	_, err := flateDecodeLimit(compressed, 1<<20)

	if err == nil {
		t.Fatal("flateDecodeLimit() error = nil, want an expansion refusal")
	}
	if !strings.Contains(err.Error(), "expands past") {
		t.Fatalf("flateDecodeLimit() error = %q, want an expansion refusal", err)
	}
}

func TestFlateDecodeLimit_AcceptsStreamAtLimit(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("x"), 4096)

	decoded, err := flateDecodeLimit(zlibCompressed(t, payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("flateDecodeLimit() error = %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("flateDecodeLimit() did not round-trip a stream that exactly fills the limit")
	}
}

func TestFlateDecode_AcceptsOrdinaryStream(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("BT /F1 12 Tf (hello) Tj ET "), 64)

	decoded, err := flateDecode(zlibCompressed(t, payload))
	if err != nil {
		t.Fatalf("flateDecode() error = %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("flateDecode() did not round-trip the payload")
	}
}
