package core

// PageBreaker is an optional interface that a Row can implement to signal
// maroto.addRow() to force a page break before placing the row's content.
// The row itself is NOT placed on any page — it is consumed entirely by the
// page-break mechanic. Render() of a PageBreaker row is a no-op.
//
// Usage: implement IsPageBreak() bool on a row type and return true.
// maroto.addRow() detects this via type assertion and calls fillPageToAddNew().
type PageBreaker interface {
	// IsPageBreak returns true when this row signals a hard page break.
	IsPageBreak() bool
}
