package merror_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/merror"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"
)

func TestDefaultErrorText(t *testing.T) {
	t.Parallel()
	// Assert
	assert.Equal(t, consts.FontFamilyArial, merror.DefaultErrorText.Family)
	assert.Equal(t, fontstyle.Bold, merror.DefaultErrorText.Style)
	assert.Equal(t, 10.0, merror.DefaultErrorText.Size)
	assert.Equal(t, 255, merror.DefaultErrorText.Color.Red)
	assert.Equal(t, 0, merror.DefaultErrorText.Color.Green)
	assert.Equal(t, 0, merror.DefaultErrorText.Color.Blue)
}
