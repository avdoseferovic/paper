package props

// Shadow holds the resolved parameters of a single CSS shadow entry.
type Shadow struct {
	OffsetX    float64 // mm
	OffsetY    float64 // mm
	BlurRadius float64 // mm (approximated as multi-rect overlay)
	Spread     float64 // mm (expands rect uniformly before blur)
	Color      *Color  // nil = use default black
	Inset      bool    // draw inside cell rather than behind it
}
