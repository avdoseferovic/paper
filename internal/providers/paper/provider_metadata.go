package paper

import "github.com/avdoseferovic/paper/pkg/core/entity"

func (g *provider) SetProtection(protection *entity.Protection) {
	if protection == nil {
		return
	}

	g.fpdf.SetProtection(byte(protection.Type), protection.UserPassword, protection.OwnerPassword)
}

func (g *provider) SetMetadata(metadata *entity.Metadata) {
	if metadata == nil {
		return
	}

	if metadata.Author != nil {
		g.fpdf.SetAuthor(metadata.Author.Text, metadata.Author.UTF8)
	}

	if metadata.Creator != nil {
		g.fpdf.SetCreator(metadata.Creator.Text, metadata.Creator.UTF8)
	}

	if metadata.Subject != nil {
		g.fpdf.SetSubject(metadata.Subject.Text, metadata.Subject.UTF8)
	}

	if metadata.Title != nil {
		g.fpdf.SetTitle(metadata.Title.Text, metadata.Title.UTF8)
	}

	if metadata.CreationDate != nil {
		g.fpdf.SetCreationDate(*metadata.CreationDate)
	}

	if metadata.KeywordsStr != nil {
		g.fpdf.SetKeywords(metadata.KeywordsStr.Text, metadata.KeywordsStr.UTF8)
	}
}
