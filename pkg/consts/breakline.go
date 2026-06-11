package consts

// BreakLineStrategy is the representation of a text line-breaking strategy.
type BreakLineStrategy string

const (
	// BreakLineEmptySpace breaks lines on word boundaries (empty spaces).
	BreakLineEmptySpace BreakLineStrategy = "empty_space_strategy"
	// BreakLineDash breaks lines mid-word, appending a dash at the break.
	BreakLineDash BreakLineStrategy = "dash_strategy"
)
