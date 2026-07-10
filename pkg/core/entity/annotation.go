package entity

import "fmt"

// AnnotationType identifies a PDF annotation subtype supported by Paper.
type AnnotationType string

const (
	AnnotationLink      AnnotationType = "Link"
	AnnotationText      AnnotationType = "Text"
	AnnotationHighlight AnnotationType = "Highlight"
	AnnotationUnderline AnnotationType = "Underline"
	AnnotationSquiggly  AnnotationType = "Squiggly"
	AnnotationStrikeOut AnnotationType = "StrikeOut"
)

// MarkupType identifies the kind of text markup annotation.
type MarkupType int

const (
	MarkupHighlight MarkupType = iota
	MarkupUnderline
	MarkupSquiggly
	MarkupStrikeOut
)

// PageAnnotation describes an annotation emitted on a generated PDF page.
type PageAnnotation struct {
	PageIndex  int
	Type       AnnotationType
	Rect       [4]float64
	URI        string
	DestName   string
	DestPage   *int
	Contents   string
	Icon       string
	Open       bool
	Color      *[3]float64
	QuadPoints [][8]float64
}

func appendAnnotationsMap(annotations []PageAnnotation, m map[string]any) map[string]any {
	if len(annotations) == 0 {
		return m
	}
	m["config_annotations"] = len(annotations)
	for i, annotation := range annotations {
		prefix := fmt.Sprintf("config_annotation_%d", i)
		m[prefix+"_page_index"] = annotation.PageIndex
		if annotation.Type != "" {
			m[prefix+"_type"] = annotation.Type
		}
		if annotation.URI != "" {
			m[prefix+"_uri"] = annotation.URI
		}
		if annotation.DestName != "" {
			m[prefix+"_dest_name"] = annotation.DestName
		}
		if annotation.DestPage != nil {
			m[prefix+"_dest_page"] = *annotation.DestPage
		}
		if annotation.Contents != "" {
			m[prefix+"_contents"] = annotation.Contents
		}
		if annotation.Icon != "" {
			m[prefix+"_icon"] = annotation.Icon
		}
		if annotation.Open {
			m[prefix+"_open"] = annotation.Open
		}
	}
	return m
}

// CloneAnnotations returns an independent copy of annotations.
func CloneAnnotations(annotations []PageAnnotation) []PageAnnotation {
	if annotations == nil {
		return nil
	}
	clones := make([]PageAnnotation, len(annotations))
	for i, annotation := range annotations {
		clones[i] = annotation
		if annotation.Color != nil {
			color := *annotation.Color
			clones[i].Color = &color
		}
		if annotation.DestPage != nil {
			destPage := *annotation.DestPage
			clones[i].DestPage = &destPage
		}
		clones[i].QuadPoints = append([][8]float64(nil), annotation.QuadPoints...)
	}
	return clones
}
