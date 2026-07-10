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

func TestStringToCIDsMapsNonBreakingSpaceToSpaceGlyph(t *testing.T) {
	t.Parallel()

	f := &PDF{
		currentFont: fontDefType{
			usedRunes: map[int]int{},
			runeToCID: map[int]int{},
		},
	}

	got := []byte(f.stringToCIDs("\u00a0"))
	want := []byte{0x00, 0x20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected NBSP CID bytes %v, got %v", want, got)
	}
	if gotRune := f.currentFont.usedRunes[0x20]; gotRune != ' ' {
		t.Fatalf("expected NBSP to use the space glyph, got U+%04X", gotRune)
	}
}

func TestGetOrAssignCIDClaimsPreseededIdentitySlots(t *testing.T) {
	t.Parallel()

	// makeSubsetRange pre-seeds usedRunes[cid] = 0 for low cids so that alias
	// replacement can inject identity-encoded text. A typed rune whose own
	// slot is merely reserved must claim it instead of moving to the PUA.
	f := &PDF{
		currentFont: fontDefType{
			usedRunes: makeSubsetRange(57),
			runeToCID: map[int]int{},
		},
	}

	cid := f.getOrAssignCID('0') // 0x30 = 48, inside the pre-seeded range
	if cid != '0' {
		t.Fatalf("expected digit to keep identity CID 0x30, got 0x%04X", cid)
	}
	if gotRune := f.currentFont.usedRunes['0']; gotRune != '0' {
		t.Fatalf("expected slot 0x30 to be claimed by rune '0', got U+%04X", gotRune)
	}
}

func TestGetOrAssignCIDStillRemapsConflictingRunes(t *testing.T) {
	t.Parallel()

	f := &PDF{
		currentFont: fontDefType{
			usedRunes: map[int]int{0x30: 0x1F600}, // slot claimed by another rune
			runeToCID: map[int]int{},
		},
	}

	if cid := f.getOrAssignCID(0x30); cid != 0xE000 {
		t.Fatalf("expected conflicting rune to be remapped to the PUA, got 0x%04X", cid)
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
