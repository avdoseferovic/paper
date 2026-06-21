package core

// Splittable is an optional interface that a Row can implement to allow
// paper.addRow() to split it across a page boundary during the build phase.
//
// When addRow() determines the row is too tall for the remaining page space
// and the row implements Splittable, it calls SplitAt(remainingHeight):
//   - didSplit==false: the row fits; proceed normally (no split needed)
//   - didSplit==true, first!=nil: place first on the current page, then
//     fillPageToAddNew() + addHeader(), then recursively addRow(rest)
//   - didSplit==true, first==nil: atomic mode — push the whole row to the
//     next page (break-inside: avoid)
//
// width is the page content width the row will actually render at, so the row
// measures its children at the same width used for rendering (a stale or
// fallback width mis-sizes wrapped content and overflows the page).
//
// SplitAt must be side-effect free on the original row.
type Splittable interface {
	SplitAt(provider Provider, remainingHeight float64, width float64) (first, rest Row, didSplit bool)
}
