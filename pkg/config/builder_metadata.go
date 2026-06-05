package config

import (
	"time"

	"github.com/avdoseferovic/paper/v2/pkg/core/entity"
)

// WithAuthor defines the author name metadata.
func (b *CfgBuilder) WithAuthor(author string, isUTF8 bool) Builder {
	if author == "" {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.Author = &entity.Utf8Text{
		Text: author,
		UTF8: isUTF8,
	}

	return b
}

// WithCreator defines the creator name metadata.
func (b *CfgBuilder) WithCreator(creator string, isUTF8 bool) Builder {
	if creator == "" {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.Creator = &entity.Utf8Text{
		Text: creator,
		UTF8: isUTF8,
	}

	return b
}

// WithSubject defines the subject metadata.
func (b *CfgBuilder) WithSubject(subject string, isUTF8 bool) Builder {
	if subject == "" {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.Subject = &entity.Utf8Text{
		Text: subject,
		UTF8: isUTF8,
	}

	return b
}

// WithTitle defines the title metadata.
func (b *CfgBuilder) WithTitle(title string, isUTF8 bool) Builder {
	if title == "" {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.Title = &entity.Utf8Text{
		Text: title,
		UTF8: isUTF8,
	}

	return b
}

// WithCreationDate defines the creation date metadata.
func (b *CfgBuilder) WithCreationDate(creationDate time.Time) Builder {
	if creationDate.IsZero() {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.CreationDate = &creationDate

	return b
}

// WithKeywords defines the document's keyword metadata.
func (b *CfgBuilder) WithKeywords(keywordsStr string, isUTF8 bool) Builder {
	if keywordsStr == "" {
		return b
	}

	if b.metadata == nil {
		b.metadata = &entity.Metadata{}
	}

	b.metadata.KeywordsStr = &entity.Utf8Text{
		Text: keywordsStr,
		UTF8: isUTF8,
	}

	return b
}
