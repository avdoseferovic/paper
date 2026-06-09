package paper

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// barrier blocks each caller until n callers have arrived, then releases them
// together. It is a reusable (cyclic) barrier: a generation counter prevents
// the classic reuse race where a fast caller re-enters the next batch and
// bumps the count before a slow caller from the previous batch re-checks its
// wait predicate. It lets a test force a deterministic number of concurrent
// workers without relying on time.Sleep.
type barrier struct {
	mu    sync.Mutex
	cond  *sync.Cond
	n     int
	count int
	gen   int
}

func newBarrier(n int) *barrier {
	b := &barrier{n: n}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *barrier) wait() {
	b.mu.Lock()
	defer b.mu.Unlock()
	gen := b.gen
	b.count++
	if b.count == b.n {
		b.count = 0
		b.gen++
		b.cond.Broadcast()
		return
	}
	for gen == b.gen {
		b.cond.Wait()
	}
}

func TestProcessPageGroupsConcurrentlyPreservesInputOrder(t *testing.T) {
	t.Parallel()

	const n = 5
	pageGroups := make([][]core.Page, n)
	completed := make([]chan struct{}, n)
	for i := range pageGroups {
		pageGroups[i] = make([]core.Page, i+1)
		completed[i] = make(chan struct{})
	}

	// Force strict reverse completion order: each job waits for the next-higher
	// index to finish before completing, so jobs complete n-1, n-2, ..., 0.
	// This deterministically proves results map back to their input index
	// regardless of completion order (no time.Sleep, no scheduling guesswork).
	results, err := processPageGroupsConcurrently(n, pageGroups, func(group []core.Page) (pageProcessResult, error) {
		idx := len(group) - 1
		if idx < n-1 {
			<-completed[idx+1]
		}
		close(completed[idx])
		return pageProcessResult{bytes: []byte{byte(len(group))}}, nil
	})

	require.NoError(t, err)
	pdfs, _ := splitPageProcessResults(results)
	assert.Equal(t, [][]byte{{1}, {2}, {3}, {4}, {5}}, pdfs)
}

func TestProcessPageGroupsConcurrentlyRespectsWorkerLimit(t *testing.T) {
	t.Parallel()

	const workers = 2
	// An even job count so the workers rendezvous in full batches of `workers`.
	pageGroups := [][]core.Page{
		make([]core.Page, 1),
		make([]core.Page, 2),
		make([]core.Page, 3),
		make([]core.Page, 4),
	}

	var active int64
	var maxActive int64
	// The barrier only releases once `workers` jobs are in-flight together, so
	// the test deterministically observes concurrency at the limit (it would
	// deadlock if the pool ran fewer than `workers` at a time), while the
	// atomic maxActive check proves it never exceeds the limit.
	b := newBarrier(workers)

	results, err := processPageGroupsConcurrently(workers, pageGroups, func(group []core.Page) (pageProcessResult, error) {
		current := atomic.AddInt64(&active, 1)
		for {
			observed := atomic.LoadInt64(&maxActive)
			if current <= observed || atomic.CompareAndSwapInt64(&maxActive, observed, current) {
				break
			}
		}
		b.wait()
		atomic.AddInt64(&active, -1)
		return pageProcessResult{bytes: []byte{byte(len(group))}}, nil
	})

	require.NoError(t, err)
	pdfs, _ := splitPageProcessResults(results)
	assert.Equal(t, [][]byte{{1}, {2}, {3}, {4}}, pdfs)
	assert.Equal(t, int64(workers), maxActive)
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

func TestProcessPageGroupsConcurrentlyRecoversWorkerPanic(t *testing.T) {
	t.Parallel()

	results, err := processPageGroupsConcurrently(3, [][]core.Page{
		make([]core.Page, 1),
		make([]core.Page, 2),
	}, func(group []core.Page) (pageProcessResult, error) {
		if len(group) == 2 {
			panic("boom")
		}
		return pageProcessResult{bytes: []byte{byte(len(group))}}, nil
	})

	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "panic processing page group")
}
