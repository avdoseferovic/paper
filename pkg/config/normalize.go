package config

import (
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
		ProviderType:         cfg.ProviderType,
		Dimensions:           cloneDimensions(cfg.Dimensions),
		Margins:              cloneMargins(cfg.Margins),
		DefaultFont:          cloneFont(cfg.DefaultFont),
		CustomFonts:          append([]entity.CustomFont(nil), cfg.CustomFonts...),
		GenerationMode:       cfg.GenerationMode,
		ChunkWorkers:         cfg.ChunkWorkers,
		Debug:                cfg.Debug,
		MaxGridSize:          cfg.MaxGridSize,
		PageNumber:           clonePageNumber(cfg.PageNumber),
		Protection:           cloneProtection(cfg.Protection),
		Compression:          cfg.Compression,
		Metadata:             cloneMetadata(cfg.Metadata),
		BackgroundImage:      cloneImage(cfg.BackgroundImage),
		DisableAutoPageBreak: cfg.DisableAutoPageBreak,
		HTMLLimits:           cfg.HTMLLimits,
		OutlineFromHeadings:  cfg.OutlineFromHeadings,
	}

	if normalized.ProviderType == "" {
		normalized.ProviderType = defaults.ProviderType
	}
	if normalized.Dimensions == nil {
		normalized.Dimensions = cloneDimensions(defaults.Dimensions)
	}
	if normalized.Margins == nil {
		normalized.Margins = cloneMargins(defaults.Margins)
	}
	if normalized.DefaultFont == nil {
		normalized.DefaultFont = cloneFont(defaults.DefaultFont)
	}
	if normalized.GenerationMode == "" {
		normalized.GenerationMode = defaults.GenerationMode
	}
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
