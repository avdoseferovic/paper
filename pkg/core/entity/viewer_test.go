package entity_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

func TestViewerPreferences_AppendMap(t *testing.T) {
	t.Parallel()

	m := (&entity.ViewerPreferences{
		PageLayout: entity.LayoutTwoPageRight,
		PageMode:   entity.ModeUseThumbs,
		OpenPage:   2,
		OpenZoom:   "FitH",
	}).AppendMap(map[string]any{})

	assert.Equal(t, entity.LayoutTwoPageRight, m["config_viewer_page_layout"])
	assert.Equal(t, entity.ModeUseThumbs, m["config_viewer_page_mode"])
	assert.Equal(t, 2, m["config_viewer_open_page"])
	assert.Equal(t, "FitH", m["config_viewer_open_zoom"])
}
