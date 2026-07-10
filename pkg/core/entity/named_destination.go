package entity

import "fmt"

// DestinationFitType is the PDF destination fit operator for a named destination.
type DestinationFitType string

const (
	// DestinationFit fits the full page in the viewer.
	DestinationFit DestinationFitType = "Fit"
	// DestinationFitH fits the page width at the provided top coordinate.
	DestinationFitH DestinationFitType = "FitH"
	// DestinationXYZ opens the page at the provided left/top/zoom values.
	DestinationXYZ DestinationFitType = "XYZ"
)

// NamedDestination defines a named location in the generated PDF.
type NamedDestination struct {
	Name      string
	PageIndex int
	FitType   DestinationFitType
	Top       float64
	Left      float64
	Zoom      float64
}

// NamedDest is a compatibility alias for Folio-style naming.
type NamedDest = NamedDestination

func appendNamedDestinationsMap(destinations []NamedDestination, m map[string]any) map[string]any {
	for i, destination := range destinations {
		prefix := fmt.Sprintf("config_named_destination_%d", i)
		if destination.Name != "" {
			m[prefix+"_name"] = destination.Name
		}
		m[prefix+"_page_index"] = destination.PageIndex
		if destination.FitType != "" {
			m[prefix+"_fit_type"] = destination.FitType
		}
		if destination.Top != 0 {
			m[prefix+"_top"] = destination.Top
		}
		if destination.Left != 0 {
			m[prefix+"_left"] = destination.Left
		}
		if destination.Zoom != 0 {
			m[prefix+"_zoom"] = destination.Zoom
		}
	}
	return m
}
