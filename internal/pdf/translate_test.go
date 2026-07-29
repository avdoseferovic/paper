package pdf

import (
	"bytes"
	"strings"
	"testing"
)

// TestRepClosure_ASCIIFastPathMatchesMapping guards the zero-allocation shortcut
// for all-ASCII input: it must return exactly what the byte-by-byte mapping
// would have produced, for every ASCII byte and for mixed/non-ASCII input.
func TestRepClosure_ASCIIFastPathMatchesMapping(t *testing.T) {
	t.Parallel()

	m := map[rune]byte{
		'é': 0xE9,
		'ü': 0xFC,
		'€': 0x80,
	}
	translate := repClosure(m)

	var everyASCII strings.Builder
	for c := range 0x80 {
		everyASCII.WriteByte(byte(c))
	}

	inputs := []string{
		"",
		"hello world",
		everyASCII.String(),
		"café",
		"über",
		"€100",
		"mixed ascii and é accents",
		"\x00\x01\x7f",
		"unmapped: 日本語",
	}

	for _, in := range inputs {
		var buf bytes.Buffer
		want := translateNonASCII(&buf, m, in)
		got := translate(in)
		if got != want {
			t.Errorf("input %q: got %q, want %q", in, got, want)
		}
	}
}

// TestRepClosure_ASCIIInputDoesNotAllocate documents the point of the fast
// path: translating ASCII text builds no new string.
// Deliberately not parallel: testing.AllocsPerRun pins GOMAXPROCS to 1 and
// panics outright when called from a parallel test.
func TestRepClosure_ASCIIInputDoesNotAllocate(t *testing.T) {
	translate := repClosure(map[rune]byte{'é': 0xE9})
	in := "plain ascii text"

	if got := translate(in); got != in {
		t.Fatalf("got %q, want %q", got, in)
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = translate(in)
	})
	if allocs != 0 {
		t.Errorf("translating ASCII text allocated %.0f times, want 0", allocs)
	}
}
