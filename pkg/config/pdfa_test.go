package config_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestBuilder_WithPdfAClonesInput(t *testing.T) {
	t.Parallel()

	pdfA := entity.PdfAConfig{
		Level:           entity.PdfA3B,
		ICCProfile:      []byte("profile"),
		OutputCondition: "Custom",
		XMPSchemas: []entity.XMPSchema{{
			Schema:       "Factur-X Schema",
			NamespaceURI: "urn:factur-x",
			Prefix:       "fx",
			Properties: []entity.XMPSchemaProperty{{
				Name:        "DocumentFileName",
				ValueType:   "Text",
				Category:    "external",
				Description: "Invoice XML filename",
			}},
		}},
		XMPProperties: []entity.XMPPropertyBlock{{
			Namespace: "urn:factur-x",
			Prefix:    "fx",
			Properties: []entity.XMPProperty{{
				Name:  "DocumentFileName",
				Value: "factur-x.xml",
			}},
		}},
	}
	cfg := config.NewBuilder().WithPdfA(pdfA).Build()
	pdfA.ICCProfile[0] = 'X'
	pdfA.XMPSchemas[0].Properties[0].Name = "Mutated"
	pdfA.XMPProperties[0].Properties[0].Value = "mutated.xml"

	require.NotNil(t, cfg.PdfA)
	assert.Equal(t, entity.PdfA3B, cfg.PdfA.Level)
	assert.Equal(t, []byte("profile"), cfg.PdfA.ICCProfile)
	assert.Equal(t, "Custom", cfg.PdfA.OutputCondition)
	assert.Equal(t, "DocumentFileName", cfg.PdfA.XMPSchemas[0].Properties[0].Name)
	assert.Equal(t, "factur-x.xml", cfg.PdfA.XMPProperties[0].Properties[0].Value)
	assert.Equal(t, entity.PdfA3B, cfg.ToMap()["config_pdfa_level"])
}

func TestBuilder_WithPdfALevelAEnablesTaggedPDF(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithPdfA(entity.PdfAConfig{Level: entity.PdfA2A}).Build()

	assert.True(t, cfg.TaggedPDF)
}

func TestNormalizeConfig_ClonesPdfA(t *testing.T) {
	t.Parallel()

	input := &entity.Config{PdfA: &entity.PdfAConfig{
		ICCProfile: []byte("profile"),
		XMPSchemas: []entity.XMPSchema{{
			Schema: "Schema",
			Properties: []entity.XMPSchemaProperty{{
				Name: "Original",
			}},
		}},
		XMPProperties: []entity.XMPPropertyBlock{{
			Properties: []entity.XMPProperty{{Value: "original"}},
		}},
	}}
	normalized := config.NormalizeConfig(input)
	input.PdfA.ICCProfile[0] = 'X'
	input.PdfA.XMPSchemas[0].Properties[0].Name = "Mutated"
	input.PdfA.XMPProperties[0].Properties[0].Value = "mutated"

	require.NotNil(t, normalized.PdfA)
	assert.Equal(t, []byte("profile"), normalized.PdfA.ICCProfile)
	assert.Equal(t, "Original", normalized.PdfA.XMPSchemas[0].Properties[0].Name)
	assert.Equal(t, "original", normalized.PdfA.XMPProperties[0].Properties[0].Value)
}
