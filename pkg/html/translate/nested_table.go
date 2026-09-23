package translate

import (
	"github.com/avdoseferovic/paper/pkg/components/table"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// nestedTableComponent keeps a one-row table measurable and splittable when
// it is the sole content of an outer table cell.
type nestedTableComponent struct {
	*table.Table
	row *splittableTableRow
}

func (n *nestedTableComponent) SetConfig(config *entity.Config) {
	n.row.SetConfig(config)
}

func (n *nestedTableComponent) splitAt(provider core.Provider, remainingHeight, width float64) (core.Component, core.Component, bool) {
	first, rest, split := n.row.SplitAt(provider, remainingHeight, width)
	if !split || first == nil || rest == nil {
		return nil, nil, false
	}
	firstRow, firstOK := first.(*splittableTableRow)
	restRow, restOK := rest.(*splittableTableRow)
	if !firstOK || !restOK {
		return nil, nil, false
	}
	return &nestedTableComponent{Table: firstRow.table, row: firstRow}, &nestedTableComponent{Table: restRow.table, row: restRow}, true
}
