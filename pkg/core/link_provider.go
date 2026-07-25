package core

// LinkProvider is a narrow optional capability interface for providers that
// support PDF internal links (named destinations and click-to-jump).
//
// The two-pass anchor flow:
//  1. Translator pre-walks the DOM; for every `id="…"` element it calls
//     AddLink() to reserve a link ID and stores name → id in a map.
//  2. During render, anchorTarget components call SetLink(id, y, page) to
//     register the destination at the correct coordinates.
//  3. anchorSource components / RichRuns with LocalAnchor call Link(...)
//     at their bounding box to make the area clickable.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if lp, ok := provider.(core.LinkProvider); ok { ... }
type LinkProvider interface {
	// AddLink reserves a new internal link target ID. Returns the link ID.
	AddLink() int
	// SetLink registers the target's Y position (mm from page top) and
	// page number (1-based) for a previously reserved link ID.
	SetLink(linkID int, y float64, page int)
	// Link makes a rectangular area clickable, jumping to the named link ID.
	// x and y are in mm from the page's top-left; w and h are the bounding box.
	Link(x, y, w, h float64, linkID int)
}

// NamedLinkProvider is an optional extension of LinkProvider for providers that
// can reserve a link target under a document-level name.
//
// Link IDs are provider-scoped. Under parallel page rendering each worker owns
// a separate document, so the worker that renders an anchor's target and the
// workers that render links to it reserve independent IDs. A name gives those
// reservations a shared identity, so they collapse into one link when the pages
// are spliced. Providers that do not implement this still work; their
// cross-worker links simply cannot be reconciled, which is why callers should
// prefer AddNamedLink whenever the destination has a name.
type NamedLinkProvider interface {
	LinkProvider
	// AddNamedLink reserves the internal link target called name, returning the
	// same link ID for repeated calls with the same name.
	AddNamedLink(name string) int
}
