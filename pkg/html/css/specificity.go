package css

// Specificity returns an integer encoding the CSS a-b-c-d specificity.
// a=inline, b=#id count, c=class/attr/pseudo-class count, d=element/pseudo-element count.
// Higher = more specific.
func Specificity(a, b, c, d int) int {
	return a*1_000_000 + b*10_000 + c*100 + d
}
