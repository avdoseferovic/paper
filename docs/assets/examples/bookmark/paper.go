// Package bookmark demonstrates PDF outline bookmarks: text components with an
// Outline prop appear in the PDF viewer's bookmark sidebar.
package bookmark

import (
	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the bookmark example document.
func GetPaper() core.Paper {
	m := paper.New()

	m.AddAutoRow(col.New(12).Add(text.New("Introduction", props.Text{
		Size:    16,
		Outline: &props.Outline{Level: 0},
	})))
	m.AddAutoRow(col.New(12).Add(text.New("Some introductory prose.", props.Text{})))

	m.AddAutoRow(col.New(12).Add(text.New("Getting Started", props.Text{
		Size:    16,
		Outline: &props.Outline{Level: 0},
	})))
	m.AddAutoRow(col.New(12).Add(text.New("Installation", props.Text{
		Size:    12,
		Outline: &props.Outline{Level: 1},
	})))
	m.AddAutoRow(col.New(12).Add(text.New("First Document", props.Text{
		Size:    12,
		Outline: &props.Outline{Level: 1, Title: "Your first document"},
	})))

	return m
}
