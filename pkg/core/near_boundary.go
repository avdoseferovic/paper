package core

// NearBoundarySplitter lets a fitting row move selected content to the next
// page when too little space remains for that content's painted box.
type NearBoundarySplitter interface {
	NearBoundaryThreshold() float64
}
