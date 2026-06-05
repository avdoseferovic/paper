package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestNewPaperReturnsConcretePaperWithoutChangingNew(t *testing.T) {
	t.Parallel()

	concrete := NewPaper()
	assert.IsType(t, &Paper{}, concrete)

	var asInterface core.Paper = concrete
	assert.NotNil(t, asInterface)

	fromExistingConstructor := New()
	assert.Implements(t, (*core.Paper)(nil), fromExistingConstructor)
}
