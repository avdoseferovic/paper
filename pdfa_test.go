package paper_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestGenerate_WithPdfA2B_ShouldEmitMetadataAndOutputIntent(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithTitle("PDF/A Test", false).
		WithAuthor("Paper", false).
		WithPdfA(entity.PdfAConfig{Level: entity.PdfA2B}).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("PDF/A foundation")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-1.7")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Metadata")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Subtype /XML")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaid:part>2</pdfaid:part>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaid:conformance>B</pdfaid:conformance>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/OutputIntents [")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/S /GTS_PDFA1")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("sRGB IEC61966-2.1")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("acsp")))
}

func TestGenerate_WithPdfA1B_ShouldUsePDF14(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithPdfA(entity.PdfAConfig{Level: entity.PdfA1B}).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("PDF/A-1 foundation")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(pdf.GetBytes(), []byte("%PDF-1.4")))
}

func TestGenerate_WithPdfA2A_ShouldEnableTaggedPDF(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithPdfA(entity.PdfAConfig{Level: entity.PdfA2A}).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Tagged PDF/A foundation")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaid:conformance>A</pdfaid:conformance>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/StructTreeRoot")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("/Marked true")))
}

func TestGenerate_WithPdfAOutputCondition(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithPdfA(entity.PdfAConfig{
			Level:           entity.PdfA3B,
			OutputCondition: "Custom CMYK",
			ICCProfile:      []byte("profile-data"),
		}).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Custom output intent")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaid:part>3</pdfaid:part>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("Custom CMYK")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("profile-data")))
}

func TestGenerate_WithPdfACustomXMPExtensions(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithPdfA(entity.PdfAConfig{
			Level: entity.PdfA3B,
			XMPSchemas: []entity.XMPSchema{{
				Schema:       "Factur-X PDFA Extension Schema",
				NamespaceURI: "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#",
				Prefix:       "fx",
				Properties: []entity.XMPSchemaProperty{{
					Name:        "DocumentFileName",
					ValueType:   "Text",
					Category:    "external",
					Description: "Name of the embedded XML invoice file",
				}},
			}},
			XMPProperties: []entity.XMPPropertyBlock{{
				Namespace: "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#",
				Prefix:    "fx",
				Properties: []entity.XMPProperty{
					{Name: "DocumentFileName", Value: "factur-x.xml"},
					{Name: "DocumentType", Value: "INVOICE"},
					{Name: "ConformanceLevel", Value: "BASIC & WL"},
				},
			}},
		}).
		Build()
	doc := paper.New(cfg)
	doc.AddAutoRow(col.New(12).Add(text.New("Factur-X PDF/A metadata")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)
	pdfBytes := pdf.GetBytes()

	assert.True(t, bytes.Contains(pdfBytes, []byte(`xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"`)))
	assert.True(t, bytes.Contains(pdfBytes, []byte("PDF/A Associated File Attachment")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaSchema:schema>Factur-X PDFA Extension Schema</pdfaSchema:schema>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaSchema:prefix>fx</pdfaSchema:prefix>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<pdfaProperty:name>DocumentFileName</pdfaProperty:name>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte(`<rdf:Description rdf:about="" xmlns:fx="urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#">`)))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<fx:DocumentFileName>factur-x.xml</fx:DocumentFileName>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<fx:DocumentType>INVOICE</fx:DocumentType>")))
	assert.True(t, bytes.Contains(pdfBytes, []byte("<fx:ConformanceLevel>BASIC &amp; WL</fx:ConformanceLevel>")))
}

func TestGenerate_WithRuntimeSetPdfAAndConcurrentMode_ShouldKeepCatalogEntries(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithCompression(false).
		WithConcurrentMode(2).
		Build()
	doc := paper.New(cfg)
	doc.SetPdfA(entity.PdfAConfig{Level: entity.PdfA2B})
	doc.AddAutoRow(col.New(12).Add(text.New("Runtime PDF/A foundation")))

	pdf, err := doc.Generate(context.Background())
	require.NoError(t, err)

	assert.True(t, bytes.Contains(pdf.GetBytes(), []byte("/OutputIntents [")))
}
