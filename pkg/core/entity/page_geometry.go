package entity

import "fmt"

// PageGeometry configures generated PDF page dictionary geometry entries.
type PageGeometry struct {
	PageIndex int
	Rotate    int
	CropBox   *[4]float64
	BleedBox  *[4]float64
	TrimBox   *[4]float64
	ArtBox    *[4]float64
}

func appendPageGeometriesMap(geometries []PageGeometry, m map[string]any) map[string]any {
	if len(geometries) == 0 {
		return m
	}
	m["config_page_geometries"] = len(geometries)
	for i, geometry := range geometries {
		prefix := fmt.Sprintf("config_page_geometry_%d", i)
		m[prefix+"_page_index"] = geometry.PageIndex
		if geometry.Rotate != 0 {
			m[prefix+"_rotate"] = geometry.Rotate
		}
		if geometry.CropBox != nil {
			m[prefix+"_crop_box"] = *geometry.CropBox
		}
		if geometry.BleedBox != nil {
			m[prefix+"_bleed_box"] = *geometry.BleedBox
		}
		if geometry.TrimBox != nil {
			m[prefix+"_trim_box"] = *geometry.TrimBox
		}
		if geometry.ArtBox != nil {
			m[prefix+"_art_box"] = *geometry.ArtBox
		}
	}
	return m
}

// ClonePageGeometries returns an independent copy of page geometries.
func ClonePageGeometries(geometries []PageGeometry) []PageGeometry {
	if geometries == nil {
		return nil
	}
	clones := make([]PageGeometry, len(geometries))
	for i, geometry := range geometries {
		clones[i] = geometry
		clones[i].CropBox = cloneBox(geometry.CropBox)
		clones[i].BleedBox = cloneBox(geometry.BleedBox)
		clones[i].TrimBox = cloneBox(geometry.TrimBox)
		clones[i].ArtBox = cloneBox(geometry.ArtBox)
	}
	return clones
}

func cloneBox(box *[4]float64) *[4]float64 {
	if box == nil {
		return nil
	}
	return new(*box)
}
