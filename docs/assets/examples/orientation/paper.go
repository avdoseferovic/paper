// Package orientation demonstrates landscape orientation.
package orientation

import (
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
)

// GetPaper builds the orientation example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithOrientation(consts.OrientationHorizontal).
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRows(
		text.NewRow(30, "content"),
	)

	return m
}
