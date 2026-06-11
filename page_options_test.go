package paper

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/require"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/pagesize"
	"github.com/avdoseferovic/paper/pkg/html"
)

func TestConfigFromPageOptions_WhenSizeAndMargins_ShouldApplyAllFourMargins(t *testing.T) {
	t.Parallel()

	cfg := configFromPageOptions(&html.PageOptions{
		PageSize:     "a5",
		MarginLeft:   10,
		MarginTop:    10,
		MarginRight:  10,
		MarginBottom: 10,
	})

	w, h := pagesize.GetDimensions(pagesize.A5)
	require.NotNil(t, cfg.Dimensions)
	assert.Equal(t, w, cfg.Dimensions.Width)
	assert.Equal(t, h, cfg.Dimensions.Height)
	require.NotNil(t, cfg.Margins)
	assert.Equal(t, 10.0, cfg.Margins.Left)
	assert.Equal(t, 10.0, cfg.Margins.Top)
	assert.Equal(t, 10.0, cfg.Margins.Right)
	assert.Equal(t, 10.0, cfg.Margins.Bottom)
}

func TestConfigFromPageOptions_WhenSizeOnly_ShouldKeepDefaultMargins(t *testing.T) {
	t.Parallel()

	cfg := configFromPageOptions(&html.PageOptions{
		PageSize:     "a5",
		MarginLeft:   -1,
		MarginTop:    -1,
		MarginRight:  -1,
		MarginBottom: -1,
	})

	defaults := config.NewBuilder().Build()
	require.NotNil(t, cfg.Margins)
	assert.Equal(t, defaults.Margins.Left, cfg.Margins.Left)
	assert.Equal(t, defaults.Margins.Top, cfg.Margins.Top)
	assert.Equal(t, defaults.Margins.Right, cfg.Margins.Right)
	assert.Equal(t, defaults.Margins.Bottom, cfg.Margins.Bottom)
}
