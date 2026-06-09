package pdf

import (
	"reflect"
	"testing"
)

func TestPackAndUnpackUint16(t *testing.T) {
	packed := packUint16(0x1234)

	if want := []byte{0x12, 0x34}; !reflect.DeepEqual(packed, want) {
		t.Fatalf("expected %v, got %v", want, packed)
	}
	if got := unpackUint16(packed); got != 0x1234 {
		t.Fatalf("expected 0x1234, got %#x", got)
	}
}

func TestPackAndUnpackUint32Array(t *testing.T) {
	packed := append(packUint32(0x12345678), packUint32(0x90abcdef)...)

	got := unpackUint32Array(packed)
	want := []int{0, 0x12345678, 0x90abcdef}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestPackHeader(t *testing.T) {
	got := packHeader(0x00010000, 2, 16, 1, 16)
	want := []byte{
		0x00, 0x01, 0x00, 0x00,
		0x00, 0x02,
		0x00, 0x10,
		0x00, 0x01,
		0x00, 0x10,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestKeySortHelpers(t *testing.T) {
	if got, want := keySortStrings(map[string][]byte{"b": nil, "a": nil, "c": nil}), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if got, want := keySortInt(map[int]int{9: 0, 1: 0, 4: 0}), []int{1, 4, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if got, want := keySortArrayRangeMap(map[int][]int{7: nil, 3: nil, 5: nil}), []int{3, 5, 7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
