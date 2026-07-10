// Package extension contains all image extensions.
package extension

// Type is a representation of a Image extension.
type Type string

const (
	// Jpg represents a jpg extension.
	Jpg Type = "jpg"
	// Jpeg represents a jpeg extension.
	Jpeg Type = "jpeg"
	// Png represents a png extension.
	Png Type = "png"
	// Svg represents a svg extension.
	Svg Type = "svg"
	// Gif represents a gif extension.
	Gif Type = "gif"
	// WebP represents a webp extension.
	WebP Type = "webp"
	// Tif represents a tif extension.
	Tif Type = "tif"
	// Tiff represents a tiff extension.
	Tiff Type = "tiff"
)

// IsValid checks if the extension is valid.
func (t Type) IsValid() bool {
	switch t {
	case Jpg, Jpeg, Png, Svg, Gif, WebP, Tif, Tiff:
		return true
	default:
		return false
	}
}
