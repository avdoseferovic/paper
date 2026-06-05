package config

import (
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
		ProviderType:         b.providerType,
		Dimensions:           cloneDimensions(b.getDimensions()),
		Margins:              cloneMargins(b.margins),
		GenerationMode:       b.generationMode,
		ChunkWorkers:         b.chunkWorkers,
		Debug:                b.debug,
		MaxGridSize:          b.maxGridSize,
		DefaultFont:          cloneFont(b.defaultFont),
		PageNumber:           pageNumber,
		Protection:           cloneProtection(b.protection),
		Compression:          b.compression,
		Metadata:             cloneMetadata(b.metadata),
		CustomFonts:          append([]entity.CustomFont(nil), b.customFonts...),
		BackgroundImage:      cloneImage(b.backgroundImage),
		DisableAutoPageBreak: b.disableAutoPageBreak,
	}
}

func cloneDimensions(dimensions *entity.Dimensions) *entity.Dimensions {
	if dimensions == nil {
		return nil
	}
	clone := *dimensions
	return &clone
}

func cloneMargins(margins *entity.Margins) *entity.Margins {
	if margins == nil {
		return nil
	}
	clone := *margins
	return &clone
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
	clone := *protection
	return &clone
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
	clone := *text
	return &clone
}

func cloneImage(image *entity.Image) *entity.Image {
	if image == nil {
		return nil
	}
	clone := *image
	clone.Bytes = append([]byte(nil), image.Bytes...)
	clone.Dimensions = cloneDimensions(image.Dimensions)
	return &clone
}
