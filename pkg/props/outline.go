package props

// Outline marks a component as an entry in the PDF document outline (the
// bookmark sidebar shown by PDF viewers).
type Outline struct {
	// Title is the text shown in the outline panel. When empty, the
	// component's own text content is used.
	Title string
	// Level is the nesting depth of the entry: 0 is the top level, 1 is
	// nested under the previous level-0 entry, and so on. Negative values
	// are treated as 0.
	Level int
}

// NormalizedLevel returns the outline level clamped to >= 0.
func (o *Outline) NormalizedLevel() int {
	if o.Level < 0 {
		return 0
	}
	return o.Level
}

// ResolveTitle returns the explicit title, or fallback when no title is set.
func (o *Outline) ResolveTitle(fallback string) string {
	if o.Title != "" {
		return o.Title
	}
	return fallback
}

// ToMap appends outline attributes to a component attribute map.
func (o *Outline) ToMap(m map[string]any) map[string]any {
	if m == nil {
		m = make(map[string]any)
	}
	m["prop_outline_level"] = o.NormalizedLevel()
	if o.Title != "" {
		m["prop_outline_title"] = o.Title
	}
	return m
}

// CloneOutline returns a deep copy of o, or nil when o is nil.
func CloneOutline(o *Outline) *Outline {
	if o == nil {
		return nil
	}
	clone := *o
	return &clone
}
