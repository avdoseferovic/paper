package core

// OutlineProvider is a narrow optional capability interface for providers
// that support PDF document outlines (the bookmark sidebar in PDF viewers).
//
// Components with an Outline prop call Bookmark during render; the entry is
// recorded on the current page at the given vertical position.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if op, ok := provider.(core.OutlineProvider); ok { ... }
type OutlineProvider interface {
	// Bookmark records an outline entry titled title at nesting depth level
	// (0 = top level) pointing at vertical position y, in mm measured from
	// the top of the page content area (margins are added by the provider).
	Bookmark(title string, level int, y float64)
}
