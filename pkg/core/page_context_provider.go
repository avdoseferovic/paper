package core

// PageContextProvider is an optional capability for providers that expose the
// physical page context (current page index, total pages, and running strings
// such as CSS running headers) to components while they render.
//
// Consumers detect support via the safe type-assertion idiom:
//
//	if pcp, ok := provider.(core.PageContextProvider); ok {
//	    pcp.WithPageContextStrings(current, total, runningStrings, fn)
//	}
type PageContextProvider interface {
	// WithPageContext runs fn with the current/total page context installed,
	// restoring the previous context afterwards.
	WithPageContext(current, total int, fn func())
	// WithPageContextStrings is WithPageContext with named running strings
	// (e.g. the active chapter title) additionally in scope.
	WithPageContextStrings(current, total int, runningStrings map[string]string, fn func())
}
