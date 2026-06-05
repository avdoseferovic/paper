package paperpdf

// SVGBasicSegmentType describes a single curve or position segment.
type SVGBasicSegmentType struct {
	Cmd byte
	Arg [6]float64
}

// SVGBasicType aggregates the paths needed to describe a multi-segment SVG
// image for SVGBasicWrite.
type SVGBasicType struct {
	Segments [][]SVGBasicSegmentType
}
