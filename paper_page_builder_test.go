package paper

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaperDelegatesPaginationStateToPageBuilder(t *testing.T) {
	t.Parallel()

	paperType := reflect.TypeFor[Paper]()
	_, ok := paperType.FieldByName("pageBuilder")
	require.True(t, ok)

	for _, field := range []string{"cell", "pages", "rows", "header", "footer", "currentHeight"} {
		_, ok := paperType.FieldByName(field)
		assert.False(t, ok, "Paper should delegate %s to pageBuilder", field)
	}
}
