package pdf

import (
	"reflect"
	"testing"
)

func TestStringToCIDsRemapsSupplementaryRunes(t *testing.T) {
	t.Parallel()

	f := &PDF{
		currentFont: fontDefType{
			usedRunes: map[int]int{},
			runeToCID: map[int]int{},
		},
	}

	got := []byte(f.stringToCIDs("A😀"))
	want := []byte{0x00, 0x41, 0xE0, 0x00}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected CID bytes %v, got %v", want, got)
	}
	if gotRune := f.currentFont.usedRunes[0xE000]; gotRune != 0x1F600 {
		t.Fatalf("expected CID 0xE000 to map to U+1F600, got U+%04X", gotRune)
	}
}

func TestColorEmojiToggleRequiresColorFont(t *testing.T) {
	t.Parallel()

	f := &PDF{}
	f.SetColorEmojiEnabled(true)
	if f.HasColorEmoji() {
		t.Fatal("expected no color emoji support without a color font")
	}
	f.currentFont.hasColorGlyphs = true
	if !f.HasColorEmoji() {
		t.Fatal("expected enabled color font to report color emoji support")
	}
}
