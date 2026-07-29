package config

import (
	"slices"

	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// Build finalizes the customization returning the entity.Config.
func (b *CfgBuilder) Build() *entity.Config {
	pageNumber := clonePageNumber(b.pageNumber)
	if pageNumber != nil {
		pageNumber.WithFont(b.defaultFont)
		pageNumber.Color = props.CloneColor(pageNumber.Color)
	}

	return &entity.Config{
		ProviderType:                   b.providerType,
		Dimensions:                     cloneDimensions(b.getDimensions()),
		Margins:                        cloneMargins(b.margins),
		GenerationMode:                 b.generationMode,
		ChunkWorkers:                   b.chunkWorkers,
		Debug:                          b.debug,
		MaxGridSize:                    b.maxGridSize,
		DefaultFont:                    cloneFont(b.defaultFont),
		PageNumber:                     pageNumber,
		Protection:                     cloneProtection(b.protection),
		Compression:                    b.compression,
		Metadata:                       cloneMetadata(b.metadata),
		CustomFonts:                    slices.Clone(b.customFonts),
		BackgroundImage:                cloneImage(b.backgroundImage),
		FirstPageBackgroundImage:       cloneImage(b.firstPageBackgroundImage),
		FirstPageForegroundImage:       cloneImage(b.firstPageForegroundImage),
		FirstPageForegroundImages:      cloneImages(b.firstPageForegroundImages),
		FirstPageFinalForegroundImages: cloneImages(b.firstPageFinalForegroundImages),
		DisableAutoPageBreak:           b.disableAutoPageBreak,
		HTMLLimits:                     b.htmlLimits,
		OutlineFromHeadings:            b.outlineFromHeadings,
		Watermark:                      props.CloneWatermark(b.watermark),
		AcroForm:                       entity.CloneAcroForm(b.acroForm),
		Annotations:                    entity.CloneAnnotations(b.annotations),
		PageGeometries:                 entity.ClonePageGeometries(b.pageGeometries),
		PdfA:                           entity.ClonePdfAConfig(b.pdfA),
		TaggedPDF:                      b.taggedPDF,
		Language:                       b.language,
		ViewerPreferences:              cloneViewerPreferences(b.viewerPreferences),
		PageLabels:                     slices.Clone(b.pageLabels),
		Attachments:                    entity.CloneAttachments(b.attachments),
		NamedDestinations:              slices.Clone(b.namedDestinations),
		FileID:                         slices.Clone(b.fileID),
		Deterministic:                  b.deterministic,
	}
}

func cloneViewerPreferences(prefs *entity.ViewerPreferences) *entity.ViewerPreferences {
	if prefs == nil {
		return nil
	}
	return new(*prefs)
}

func cloneDimensions(dimensions *entity.Dimensions) *entity.Dimensions {
	if dimensions == nil {
		return nil
	}
	return new(*dimensions)
}

func cloneMargins(margins *entity.Margins) *entity.Margins {
	if margins == nil {
		return nil
	}
	return new(*margins)
}

func cloneFont(font *props.Font) *props.Font {
	if font == nil {
		return nil
	}
	clone := props.NormalizeFont(*font, "")
	return &clone
}

func clonePageNumber(pageNumber *props.PageNumber) *props.PageNumber {
	if pageNumber == nil {
		return nil
	}
	clone := props.ClonePageNumber(*pageNumber)
	return &clone
}

func cloneProtection(protection *entity.Protection) *entity.Protection {
	if protection == nil {
		return nil
	}
	return new(*protection)
}

func cloneMetadata(metadata *entity.Metadata) *entity.Metadata {
	if metadata == nil {
		return nil
	}
	clone := &entity.Metadata{
		Author:      cloneUTF8Text(metadata.Author),
		Creator:     cloneUTF8Text(metadata.Creator),
		Subject:     cloneUTF8Text(metadata.Subject),
		Title:       cloneUTF8Text(metadata.Title),
		KeywordsStr: cloneUTF8Text(metadata.KeywordsStr),
	}
	if metadata.CreationDate != nil {
		creationDate := *metadata.CreationDate
		clone.CreationDate = &creationDate
	}
	return clone
}

func cloneUTF8Text(text *entity.Utf8Text) *entity.Utf8Text {
	if text == nil {
		return nil
	}
	return new(*text)
}

func cloneImage(image *entity.Image) *entity.Image {
	if image == nil {
		return nil
	}
	clone := *image
	clone.Bytes = slices.Clone(image.Bytes)
	clone.Dimensions = cloneDimensions(image.Dimensions)
	clone.PageCell = cloneCell(image.PageCell)
	return &clone
}

func cloneImages(images []*entity.Image) []*entity.Image {
	if len(images) == 0 {
		return nil
	}
	clones := make([]*entity.Image, 0, len(images))
	for _, image := range images {
		clones = append(clones, cloneImage(image))
	}
	return clones
}

func cloneCell(cell *entity.Cell) *entity.Cell {
	if cell == nil {
		return nil
	}
	return new(*cell)
}
