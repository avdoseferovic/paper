// Package consts groups the small enumeration types used across Paper:
// alignment, orientation, line styles, break-line strategies, font families,
// barcode types, generation modes, and provider types. Larger constant
// domains (border, pagesize, fontstyle, extension, protection) keep their own
// subpackages.
package consts

// Align is the representation of a horizontal or vertical alignment.
type Align string

const (
	// AlignLeft represents a left horizontal alignment.
	AlignLeft Align = "L"
	// AlignRight represents a right horizontal alignment.
	AlignRight Align = "R"
	// AlignCenter represents a center horizontal/vertical alignment.
	AlignCenter Align = "C"
	// AlignTop represents a top vertical alignment.
	AlignTop Align = "T"
	// AlignBottom represents a bottom vertical alignment.
	AlignBottom Align = "B"
	// AlignMiddle represents a middle vertical alignment.
	AlignMiddle Align = "M"
	// AlignJustify represents a horizontal alignment that distributes
	// extra space between words so both edges of the column are flush.
	AlignJustify Align = "J"
)
