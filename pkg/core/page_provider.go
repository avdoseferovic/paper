package core

// PageProvider is an optional capability for providers that can ensure the
// physical output page matches Maroto's logical page during rendering.
type PageProvider interface {
	EnsurePage(pageNumber int)
}
