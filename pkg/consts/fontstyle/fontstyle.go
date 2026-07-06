// Package fontstyle contains all default font styles.
package fontstyle

// Type is a representation of a style DefaultFont.
type Type string

const (
	// Normal represents a normal style.
	Normal Type = ""
	// Bold represents a bold style.
	Bold Type = "B"
	// Semibold represents a medium/semibold style.
	Semibold Type = "M"
	// Italic represents an italic style.
	Italic Type = "I"
	// BoldItalic represents a bold and italic style.
	BoldItalic Type = Bold + Italic
	// SemiboldItalic represents a semibold and italic style.
	SemiboldItalic Type = Semibold + Italic
	// Underline represents underlined style.
	Underline Type = "U"
	// Strikethrough represents strike-through style.
	Strikethrough Type = "S"
)

// IsValid checks if the style is valid.
func (s Type) IsValid() bool {
	for _, r := range s {
		switch Type(r) {
		case
			Normal,
			Italic,
			Bold,
			Semibold,
			BoldItalic,
			SemiboldItalic,
			Underline,
			Strikethrough:
		default:
			return false
		}
	}
	return true
}
