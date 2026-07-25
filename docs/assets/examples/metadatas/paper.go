// Package metadatas demonstrates setting document metadata such as title, author, and subject.
package metadatas

import (
	"time"

	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

// GetPaper builds the metadatas example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithAuthor("author", false).
		WithCreator("creator", false).
		WithSubject("subject", false).
		WithTitle("title", false).
		WithKeywords("keyword", false).
		WithCreationDate(time.Now()).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRows(
		text.NewRow(30, "metadatas"),
	)

	return m
}
