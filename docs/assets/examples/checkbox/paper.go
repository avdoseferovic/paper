// Package checkbox demonstrates checkbox components with and without labels.
package checkbox

import (
	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/components/checkbox"
	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/decorator"
	"github.com/avdoseferovic/paper/pkg/props"
)

// GetPaper builds the checkbox example document.
func GetPaper() core.Paper {
	cfg := config.NewBuilder().
		WithDebug(true).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	m.AddRow(20,
		checkbox.NewCol(4, "Option A"),
		checkbox.NewCol(4, "Option B", props.Checkbox{Checked: true}),
		checkbox.NewCol(4, "Option C"),
	)
	m.AddRows(text.NewRow(8, "Default: unchecked / checked / unchecked"))

	m.AddRow(20,
		checkbox.NewCol(4, "Option A", props.Checkbox{Size: 8}),
		checkbox.NewCol(4, "Option B", props.Checkbox{Size: 8, Checked: true}),
		checkbox.NewCol(4, "Option C", props.Checkbox{Size: 8}),
	)
	m.AddRows(text.NewRow(8, "Custom size: 8mm"))

	m.AddRow(20,
		checkbox.NewCol(4, "Option A", props.Checkbox{Top: 5, Left: 5}),
		checkbox.NewCol(4, "Option B", props.Checkbox{Top: 5, Left: 5, Checked: true}),
		checkbox.NewCol(4, "Option C", props.Checkbox{Top: 5, Left: 5}),
	)
	m.AddRows(text.NewRow(8, "With top=5 left=5 offset"))

	m.AddAutoRow(
		checkbox.NewCol(3, "Item 1"),
		checkbox.NewCol(3, "Item 2", props.Checkbox{Checked: true}),
		checkbox.NewCol(3, "Item 3"),
		checkbox.NewCol(3, "Item 4", props.Checkbox{Checked: true}),
	)
	m.AddRows(text.NewRow(8, "Auto row"))

	return m
}
