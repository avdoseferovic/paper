// Package protection demonstrates password protection and permission flags.
package protection

import (
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/consts/protection"

	"github.com/avdoseferovic/paper/pkg/config"
)

// GetPaper builds the protection example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithProtection(protection.None, "user", "owner").
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRows(
		text.NewRow(30, "supersecret content"),
	)

	return m
}
