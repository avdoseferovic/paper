// Package disablepagebreak demonstrates keeping a row on one page instead of letting it split.
package disablepagebreak

import (
	"log"
	"os"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/core"
)

// GetPaper builds the disablepagebreak example document.
func GetPaper(image string) core.Paper {
	bytes, err := os.ReadFile(image)
	if err != nil {
		log.Fatal(err)
	}
	b := config.NewBuilder().
		WithTopMargin(0).
		WithRightMargin(0).
		WithLeftMargin(0).
		WithDimensions(361.8, 203.2).
		WithDisableAutoPageBreak(true).
		WithOrientation(consts.OrientationHorizontal).
		WithMaxGridSize(20).
		WithBackgroundImage(bytes, extension.Png).
		Build()

	b.Margins.Bottom = 0

	mrt := paper.New(b)
	m := decorator.NewMetrics(mrt)

	m.AddPages(page.New())
	return m
}
