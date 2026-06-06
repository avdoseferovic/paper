package props

// Rect represents properties from a rectangle (Image, QrCode or Barcode) inside a cell.
type Rect struct {
	// Left is the space between the left cell boundary to the rectangle, if center is false.
	Left float64
	// Top is space between the upper cell limit to the barcode, if center is false.
	Top float64
	// Percent is how much the rectangle will occupy the cell,
	// ex 100%: The rectangle will fulfill the entire cell
	// ex 50%: The greater side from the rectangle will have half the size of the cell.
	Percent float64
	// indicate whether only the width should be used as a reference to calculate the component size, disregarding the height
	// ex true: The component will be scaled only based on the available width, disregarding the available height
	JustReferenceWidth bool
	// Center define that the barcode will be vertically and horizontally centralized.
	Center bool
	// ObjectFit defines how an image should fit into its content box.
	ObjectFit string
	// ObjectPosition defines the image alignment inside its content box.
	ObjectPosition string
}

// ToMap from Rect will return a map representation from Rect.
func (r *Rect) ToMap() map[string]any {
	m := make(map[string]any)

	if r.Left != 0 {
		m["prop_left"] = r.Left
	}

	if r.Top != 0 {
		m["prop_top"] = r.Top
	}

	if r.Percent != 0 {
		m["prop_percent"] = r.Percent
	}

	if r.Center {
		m["prop_center"] = r.Center
	}

	if r.JustReferenceWidth {
		m["prop_just_reference_width"] = r.JustReferenceWidth
	}
	if r.ObjectFit != "" {
		m["prop_object_fit"] = r.ObjectFit
	}
	if r.ObjectPosition != "" {
		m["prop_object_position"] = r.ObjectPosition
	}
	return m
}

// MakeValid from Rect will make the properties from a rectangle reliable to fit inside a cell
// and define default values for a rectangle.
func (r *Rect) MakeValid() {
	*r = NormalizeRect(*r)
}

// NormalizeRect returns a defaulted copy of r.
func NormalizeRect(r Rect) Rect {
	minPercentage := 0.0
	maxPercentage := 100.0
	minValue := 0.0

	if r.Percent <= minPercentage || r.Percent > maxPercentage {
		r.Percent = maxPercentage
	}

	if r.Center {
		r.Left = 0
		r.Top = 0
	}

	if r.Left < minValue {
		r.Left = minValue
	}

	if r.Top < minValue {
		r.Top = minValue
	}

	return r
}
