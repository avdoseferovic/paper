package paper

import (
	"context"
	"fmt"
	"sync"

	"github.com/avdoseferovic/paper/pkg/core"
)

// processPageGroupsConcurrently runs processor over every page group on up to
// workerCount goroutines and returns the results in page-group order.
//
// The first error wins: on cancellation or a processor failure the remaining
// groups are abandoned and that error is returned. A processor panic is
// recovered and reported as an error rather than taking down the process.
func processPageGroupsConcurrently[T any](
	ctx context.Context,
	workerCount int,
	pageGroups [][]core.Page,
	processor func(context.Context, []core.Page) (T, error),
) ([]T, error) {
	if len(pageGroups) == 0 {
		return nil, nil
	}
	err := generationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	if workerCount < 1 {
		workerCount = 1
	}
	workerCount = min(workerCount, len(pageGroups))

	results := make([]T, len(pageGroups))
	jobs := make(chan int)
	done := ctx.Done()
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	recordErr := func(err error) {
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}
	failed := func() bool {
		errMu.Lock()
		defer errMu.Unlock()
		return firstErr != nil
	}

	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					recordErr(generationCanceled(ctx))
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					// Once any group has failed the result is discarded, so stop
					// spending work on the groups that follow.
					if failed() {
						return
					}
					runPageGroupJob(ctx, index, pageGroups, processor, results, recordErr)
				}
			}
		}()
	}

	for index := range pageGroups {
		if failed() {
			break
		}
		select {
		case <-done:
			recordErr(generationCanceled(ctx))
			close(jobs)
			wg.Wait()
			return nil, firstErr
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

// runPageGroupJob processes a single page group in an isolated scope so that a
// panic in the processor is recovered and reported via recordErr instead of
// crashing the worker goroutine (recover only works within the same goroutine).
func runPageGroupJob[T any](
	ctx context.Context,
	index int,
	pageGroups [][]core.Page,
	processor func(context.Context, []core.Page) (T, error),
	results []T,
	recordErr func(error),
) {
	defer func() {
		if r := recover(); r != nil {
			recordErr(fmt.Errorf("%w %d: %v", errPanicProcessingPageGroup, index, r))
		}
	}()

	result, err := processor(ctx, pageGroups[index])
	if err != nil {
		recordErr(err)
		return
	}
	results[index] = result
}
