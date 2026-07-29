package config

import (
	"cmp"
	"slices"

	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

// NormalizeConfig returns an independent, valid config for Paper runtime use.
// It preserves caller-provided values where valid and falls back to builder
// defaults for required fields that are missing or invalid.
func NormalizeConfig(cfg *entity.Config) *entity.Config {
	defaults := NewBuilder().Build()
	if cfg == nil {
		return defaults
	}

	normalized := &entity.Config{
		ProviderType:                   cfg.ProviderType,
		Dimensions:                     cloneDimensions(cfg.Dimensions),
		Margins:                        cloneMargins(cfg.Margins),
		DefaultFont:                    cloneFont(cfg.DefaultFont),
		CustomFonts:                    slices.Clone(cfg.CustomFonts),
		GenerationMode:                 cfg.GenerationMode,
		ChunkWorkers:                   cfg.ChunkWorkers,
		Debug:                          cfg.Debug,
		MaxGridSize:                    cfg.MaxGridSize,
		PageNumber:                     clonePageNumber(cfg.PageNumber),
		Protection:                     cloneProtection(cfg.Protection),
		Compression:                    cfg.Compression,
		Metadata:                       cloneMetadata(cfg.Metadata),
		BackgroundImage:                cloneImage(cfg.BackgroundImage),
		FirstPageBackgroundImage:       cloneImage(cfg.FirstPageBackgroundImage),
		FirstPageForegroundImage:       cloneImage(cfg.FirstPageForegroundImage),
		FirstPageForegroundImages:      cloneImages(cfg.FirstPageForegroundImages),
		FirstPageFinalForegroundImages: cloneImages(cfg.FirstPageFinalForegroundImages),
		DisableAutoPageBreak:           cfg.DisableAutoPageBreak,
		HTMLLimits:                     cfg.HTMLLimits,
		OutlineFromHeadings:            cfg.OutlineFromHeadings,
		Watermark:                      props.CloneWatermark(cfg.Watermark),
		AcroForm:                       entity.CloneAcroForm(cfg.AcroForm),
		Annotations:                    entity.CloneAnnotations(cfg.Annotations),
		PageGeometries:                 entity.ClonePageGeometries(cfg.PageGeometries),
		PdfA:                           entity.ClonePdfAConfig(cfg.PdfA),
		TaggedPDF:                      cfg.TaggedPDF,
		Language:                       cfg.Language,
		ViewerPreferences:              cloneViewerPreferences(cfg.ViewerPreferences),
		PageLabels:                     slices.Clone(cfg.PageLabels),
		Attachments:                    entity.CloneAttachments(cfg.Attachments),
		NamedDestinations:              slices.Clone(cfg.NamedDestinations),
		FileID:                         slices.Clone(cfg.FileID),
		Deterministic:                  cfg.Deterministic,
	}

	normalized.ProviderType = cmp.Or(normalized.ProviderType, defaults.ProviderType)
	if normalized.Dimensions == nil {
		normalized.Dimensions = cloneDimensions(defaults.Dimensions)
	}
	if normalized.Margins == nil {
		normalized.Margins = cloneMargins(defaults.Margins)
	}
	if normalized.DefaultFont == nil {
		normalized.DefaultFont = cloneFont(defaults.DefaultFont)
	}
	normalized.GenerationMode = cmp.Or(normalized.GenerationMode, defaults.GenerationMode)
	if normalized.ChunkWorkers < 1 {
		normalized.ChunkWorkers = defaults.ChunkWorkers
	}
	if normalized.MaxGridSize <= 0 {
		normalized.MaxGridSize = defaults.MaxGridSize
	}
	if normalized.PageNumber != nil {
		normalized.PageNumber.WithFont(normalized.DefaultFont)
		normalized.PageNumber.Color = props.CloneColor(normalized.PageNumber.Color)
	}

	return normalized
}
