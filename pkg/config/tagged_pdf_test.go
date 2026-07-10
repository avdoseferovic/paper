package config_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestBuilder_WithTaggedPDF(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithTaggedPDF(true).Build()

	assert.True(t, cfg.TaggedPDF)
	assert.True(t, cfg.ToMap()["config_tagged_pdf"].(bool))
}

func TestNormalizeConfig_PreservesTaggedPDF(t *testing.T) {
	t.Parallel()

	normalized := config.NormalizeConfig(&entity.Config{TaggedPDF: true})

	assert.True(t, normalized.TaggedPDF)
}
