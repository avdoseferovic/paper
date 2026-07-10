package core

// LayeredRow is an optional capability for rows that render out of the normal
// flow paint order. Negative layers paint behind the page's flow rows,
// positive layers in front; 0 (or not implementing the interface) is normal
// flow order. Used by CSS positioned rows carrying a z-index.
type LayeredRow interface {
	// RenderLayer returns the paint layer for the row.
	RenderLayer() int
}
