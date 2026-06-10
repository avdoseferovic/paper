package paper

import (
	pdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func (g *provider) SetProtection(protection *entity.Protection) {
	if protection == nil {
		return
	}

	g.metadataPDF.SetProtectionAlgorithm(pdfProtectionAlgorithm(protection.Algorithm))
	g.metadataPDF.SetProtection(byte(protection.Type), protection.UserPassword, protection.OwnerPassword)
}

func pdfProtectionAlgorithm(algorithm protection.Encryption) pdf.ProtectionAlgorithm {
	if algorithm == protection.AES128 {
		return pdf.ProtectionAES128
	}

	return pdf.ProtectionRC4
}

func (g *provider) SetMetadata(metadata *entity.Metadata) {
	if metadata == nil {
		return
	}

	if metadata.Author != nil {
		g.metadataPDF.SetAuthor(metadata.Author.Text, metadata.Author.UTF8)
	}

	if metadata.Creator != nil {
		g.metadataPDF.SetCreator(metadata.Creator.Text, metadata.Creator.UTF8)
	}

	if metadata.Subject != nil {
		g.metadataPDF.SetSubject(metadata.Subject.Text, metadata.Subject.UTF8)
	}

	if metadata.Title != nil {
		g.metadataPDF.SetTitle(metadata.Title.Text, metadata.Title.UTF8)
	}

	if metadata.CreationDate != nil {
		g.metadataPDF.SetCreationDate(*metadata.CreationDate)
	}

	if metadata.KeywordsStr != nil {
		g.metadataPDF.SetKeywords(metadata.KeywordsStr.Text, metadata.KeywordsStr.UTF8)
	}
}
