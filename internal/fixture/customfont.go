package fixture

import (
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
)

// TestFont is a stand-in font for tests, returning fixed values instead of
// reading a real font file.
type TestFont struct {
	Family string
	Style  fontstyle.Type
	File   string
	Bytes  []byte
}

// GetFamily returns the font family name.
func (t TestFont) GetFamily() string {
	return t.Family
}

// GetStyle returns the font style.
func (t TestFont) GetStyle() fontstyle.Type {
	return t.Style
}

// GetFile returns the font file path.
func (t TestFont) GetFile() string {
	return t.File
}

// GetBytes returns the embedded font bytes.
func (t TestFont) GetBytes() []byte {
	return t.Bytes
}
