package svg_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/svg"
)

const tinySVG = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10" fill="red"/></svg>`

func TestRasterizeWithLimit(t *testing.T) {
	t.Parallel()

	png, width, height, err := svg.RasterizeWithLimit([]byte(tinySVG), 10, 10, 1_000_000)
	require.NoError(t, err)
	assert.NotEmpty(t, png)
	assert.True(t, width > 0)
	assert.True(t, height > 0)

	_, _, _, err = svg.RasterizeWithLimit([]byte(tinySVG), 1000, 1000, 4)
	assert.Error(t, err, "pixel limit must be enforced")
}
