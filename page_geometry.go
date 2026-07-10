package paper

import "github.com/avdoseferovic/paper/pkg/core/entity"

// SetPageGeometry configures generated PDF page dictionary geometry entries.
func (m *Paper) SetPageGeometry(geometry entity.PageGeometry) {
	m.config.PageGeometries = upsertPageGeometry(m.config.PageGeometries, geometry)
}

// SetPageRotation sets a generated page rotation in degrees.
func (m *Paper) SetPageRotation(pageIndex, degrees int) {
	geometry := m.pageGeometry(pageIndex)
	geometry.Rotate = degrees
	m.SetPageGeometry(geometry)
}

// SetCropBox sets a generated page CropBox.
func (m *Paper) SetCropBox(pageIndex int, box [4]float64) {
	geometry := m.pageGeometry(pageIndex)
	geometry.CropBox = &box
	m.SetPageGeometry(geometry)
}

// SetBleedBox sets a generated page BleedBox.
func (m *Paper) SetBleedBox(pageIndex int, box [4]float64) {
	geometry := m.pageGeometry(pageIndex)
	geometry.BleedBox = &box
	m.SetPageGeometry(geometry)
}

// SetTrimBox sets a generated page TrimBox.
func (m *Paper) SetTrimBox(pageIndex int, box [4]float64) {
	geometry := m.pageGeometry(pageIndex)
	geometry.TrimBox = &box
	m.SetPageGeometry(geometry)
}

// SetArtBox sets a generated page ArtBox.
func (m *Paper) SetArtBox(pageIndex int, box [4]float64) {
	geometry := m.pageGeometry(pageIndex)
	geometry.ArtBox = &box
	m.SetPageGeometry(geometry)
}

func (m *Paper) pageGeometry(pageIndex int) entity.PageGeometry {
	for _, geometry := range m.config.PageGeometries {
		if geometry.PageIndex == pageIndex {
			return entity.ClonePageGeometries([]entity.PageGeometry{geometry})[0]
		}
	}
	return entity.PageGeometry{PageIndex: pageIndex}
}

func upsertPageGeometry(geometries []entity.PageGeometry, geometry entity.PageGeometry) []entity.PageGeometry {
	clones := entity.ClonePageGeometries(geometries)
	incoming := entity.ClonePageGeometries([]entity.PageGeometry{geometry})[0]
	for i := range clones {
		if clones[i].PageIndex == incoming.PageIndex {
			clones[i] = incoming
			return clones
		}
	}
	return append(clones, incoming)
}
