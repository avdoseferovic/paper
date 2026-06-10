package time_test

import (
	"testing"
	buildtinTime "time"

	"github.com/avdoseferovic/paper/internal/time"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestGetTimeSpent(t *testing.T) {
	t.Parallel()
	// Act
	timeSpent := time.GetTimeSpent(func() {
		buildtinTime.Sleep(10 * buildtinTime.Millisecond)
	})

	// Assert
	assert.InDelta(t, float64(10*buildtinTime.Millisecond), timeSpent.Value, float64(10*buildtinTime.Millisecond))
}
