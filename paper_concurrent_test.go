package paper

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessPageGroupsConcurrentlyPreservesOrderAndWorkerLimit(t *testing.T) {
	t.Parallel()

	pageGroups := [][]core.Page{
		make([]core.Page, 1),
		make([]core.Page, 2),
		make([]core.Page, 3),
		make([]core.Page, 4),
		make([]core.Page, 5),
	}

	var active int64
	var maxActive int64
	results, err := processPageGroupsConcurrently(2, pageGroups, func(group []core.Page) (pageProcessResult, error) {
		current := atomic.AddInt64(&active, 1)
		for {
			observed := atomic.LoadInt64(&maxActive)
			if current <= observed || atomic.CompareAndSwapInt64(&maxActive, observed, current) {
				break
			}
		}
		defer atomic.AddInt64(&active, -1)

		time.Sleep(time.Duration(6-len(group)) * time.Millisecond)
		return pageProcessResult{bytes: []byte{byte(len(group))}}, nil
	})

	require.NoError(t, err)
	pdfs, _ := splitPageProcessResults(results)
	assert.Equal(t, [][]byte{{1}, {2}, {3}, {4}, {5}}, pdfs)
	assert.LessOrEqual(t, maxActive, int64(2))
}

func TestProcessPageGroupsConcurrentlyReturnsProcessorError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("process page group")
	_, err := processPageGroupsConcurrently(3, [][]core.Page{
		make([]core.Page, 1),
		make([]core.Page, 2),
	}, func(group []core.Page) (pageProcessResult, error) {
		if len(group) == 2 {
			return pageProcessResult{}, expectedErr
		}
		return pageProcessResult{bytes: []byte{byte(len(group))}}, nil
	})

	assert.ErrorIs(t, err, expectedErr)
}
