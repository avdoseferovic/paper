package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core"
)

func TestNewReturnsConcretePaper(t *testing.T) {
	t.Parallel()

	fromConstructor := New()
	assert.IsType(t, &Paper{}, fromConstructor)

	var fromConstructorAsInterface core.Paper = fromConstructor
	assert.NotNil(t, fromConstructorAsInterface)

	concrete := NewPaper()
	assert.IsType(t, &Paper{}, concrete)

	var asInterface core.Paper = concrete
	assert.NotNil(t, asInterface)
}
