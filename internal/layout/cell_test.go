package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avdoseferovic/paper/internal/layout"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestApplyCellMargins(t *testing.T) {
	t.Parallel()

	cell := entity.Cell{X: 10, Y: 20, Width: 100, Height: 50}
	style := &props.Cell{
		MarginTop:    3,
		MarginRight:  7,
		MarginBottom: 5,
		MarginLeft:   2,
	}

	got := layout.ApplyCellMargins(cell, style)

	assert.Equal(t, entity.Cell{X: 12, Y: 23, Width: 91, Height: 42}, got)
}

func TestApplyCellMargins_ClampsNegativeAndOversizedMargins(t *testing.T) {
	t.Parallel()

	cell := entity.Cell{X: 10, Y: 20, Width: 5, Height: 4}
	style := &props.Cell{
		MarginTop:    3,
		MarginRight:  8,
		MarginBottom: 5,
		MarginLeft:   -2,
	}

	got := layout.ApplyCellMargins(cell, style)

	assert.Equal(t, entity.Cell{X: 10, Y: 23, Width: 0, Height: 0}, got)
}
