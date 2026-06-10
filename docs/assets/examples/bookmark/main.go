// Package main demonstrates PDF outline bookmarks: text components with an
// Outline prop appear in the PDF viewer's bookmark sidebar.
package main

import (
	"log"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
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

	document, err := m.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/bookmark.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}
}
