package paper

import (
	"slices"

	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// AddAnnotation adds a page annotation to the generated PDF.
func (m *Paper) AddAnnotation(annotation entity.PageAnnotation) {
	m.config.Annotations = append(m.config.Annotations, entity.CloneAnnotations([]entity.PageAnnotation{annotation})...)
}

// AddLinkAnnotation adds an external URI link annotation to a generated page.
func (m *Paper) AddLinkAnnotation(pageIndex int, rect [4]float64, uri string) {
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex: pageIndex,
		Type:      entity.AnnotationLink,
		Rect:      rect,
		URI:       uri,
	})
}

// AddInternalLinkAnnotation adds an internal named-destination link annotation.
func (m *Paper) AddInternalLinkAnnotation(pageIndex int, rect [4]float64, destName string) {
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex: pageIndex,
		Type:      entity.AnnotationLink,
		Rect:      rect,
		DestName:  destName,
	})
}

// AddPageLinkAnnotation adds an internal link annotation to a zero-based page index.
func (m *Paper) AddPageLinkAnnotation(pageIndex int, rect [4]float64, destPage int) {
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex: pageIndex,
		Type:      entity.AnnotationLink,
		Rect:      rect,
		DestPage:  &destPage,
	})
}

// AddTextAnnotation adds a sticky-note annotation to a generated page.
func (m *Paper) AddTextAnnotation(pageIndex int, rect [4]float64, text, icon string) {
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex: pageIndex,
		Type:      entity.AnnotationText,
		Rect:      rect,
		Contents:  text,
		Icon:      icon,
	})
}

// AddTextAnnotationOpen adds an initially open sticky-note annotation.
func (m *Paper) AddTextAnnotationOpen(pageIndex int, rect [4]float64, text, icon string) {
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex: pageIndex,
		Type:      entity.AnnotationText,
		Rect:      rect,
		Contents:  text,
		Icon:      icon,
		Open:      true,
	})
}

// AddHighlight adds a highlight text-markup annotation.
func (m *Paper) AddHighlight(pageIndex int, rect [4]float64, color [3]float64, quadPoints [][8]float64) {
	m.AddTextMarkup(pageIndex, entity.MarkupHighlight, rect, color, quadPoints)
}

// AddUnderline adds an underline text-markup annotation.
func (m *Paper) AddUnderline(pageIndex int, rect [4]float64, color [3]float64, quadPoints [][8]float64) {
	m.AddTextMarkup(pageIndex, entity.MarkupUnderline, rect, color, quadPoints)
}

// AddSquiggly adds a squiggly text-markup annotation.
func (m *Paper) AddSquiggly(pageIndex int, rect [4]float64, color [3]float64, quadPoints [][8]float64) {
	m.AddTextMarkup(pageIndex, entity.MarkupSquiggly, rect, color, quadPoints)
}

// AddStrikeOut adds a strikeout text-markup annotation.
func (m *Paper) AddStrikeOut(pageIndex int, rect [4]float64, color [3]float64, quadPoints [][8]float64) {
	m.AddTextMarkup(pageIndex, entity.MarkupStrikeOut, rect, color, quadPoints)
}

// AddTextMarkup adds a text-markup annotation to a generated page.
func (m *Paper) AddTextMarkup(
	pageIndex int,
	markupType entity.MarkupType,
	rect [4]float64,
	color [3]float64,
	quadPoints [][8]float64,
) {
	types := [...]entity.AnnotationType{
		entity.AnnotationHighlight,
		entity.AnnotationUnderline,
		entity.AnnotationSquiggly,
		entity.AnnotationStrikeOut,
	}
	annotationType := entity.AnnotationHighlight
	if markupType >= 0 && int(markupType) < len(types) {
		annotationType = types[markupType]
	}
	m.AddAnnotation(entity.PageAnnotation{
		PageIndex:  pageIndex,
		Type:       annotationType,
		Rect:       rect,
		Color:      &color,
		QuadPoints: slices.Clone(quadPoints),
	})
}
