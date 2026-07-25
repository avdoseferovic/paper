package paper

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/avdoseferovic/paper/internal/cache"
	"github.com/avdoseferovic/paper/internal/pdf"
	paperprovider "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

// generateParallelPages renders page groups concurrently, each into its own
// document, then splices the rendered pages into a single document that is
// serialized once.
//
// This differs from generateConcurrently in what happens after rendering:
// concurrent mode serializes every chunk to PDF bytes and merges those byte
// streams, so each chunk pays for a full document (font programs, resource
// dictionary, xref) and the merge re-parses all of it. Here only page content
// streams and resource tables move between documents, so the per-chunk cost
// does not grow with the worker count.
func (m *Paper) generateParallelPages(ctx context.Context) (*core.Pdf, error) {
	pageGroups, err := m.chunkedPageGroups(ctx)
	if err != nil {
		return nil, err
	}
	if len(pageGroups) == 0 {
		return m.generateSequentially(ctx)
	}

	sharedCache := cache.NewMutexDecorator(cache.New())

	rendered, err := m.renderPageGroupsConcurrently(ctx, pageGroups, sharedCache)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", ErrCannotGenerateInParallelMode, err)
	}

	if err := generationCanceled(ctx); err != nil {
		return nil, err
	}

	doc, err := m.spliceAndSerialize(rendered)
	if errors.Is(err, pdf.ErrAbsorbUnsupported) {
		// The document turned out to use a feature whose PDF names or page
		// references are position-dependent, which is only detectable once the
		// pages have been rendered. Correctness wins: re-render sequentially.
		return m.generateSequentially(ctx)
	}
	return doc, err
}

// spliceAndSerialize joins the per-worker documents in page order and writes
// the final PDF in a single serialization pass.
//
// The first worker's document is the splice target rather than a fresh one:
// every provider is built with one page already started, so absorbing into a
// new document would prepend a blank page.
func (m *Paper) spliceAndSerialize(rendered []core.Provider) (*core.Pdf, error) {
	master := rendered[0]
	if master == nil {
		return nil, fmt.Errorf("%w: first page group did not render", ErrCannotGenerateInParallelMode)
	}
	absorber, ok := master.(paperprovider.PageAbsorber)
	if !ok {
		return nil, fmt.Errorf("%w: provider cannot absorb pages", ErrCannotGenerateInParallelMode)
	}

	for _, worker := range rendered[1:] {
		if worker == nil {
			continue
		}
		if err := absorber.AbsorbPagesFrom(worker); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrCannotGenerateInParallelMode, err)
		}
	}

	documentBytes, err := master.GenerateBytes()
	if err != nil {
		return nil, err
	}

	var issues []metrics.RenderIssue
	issues = append(issues, collectRenderIssues(master)...)
	return core.NewPDF(documentBytes, reportFromIssues(issues)), nil
}

// renderPageGroupsConcurrently renders each page group into its own provider,
// returning the providers in page order.
func (m *Paper) renderPageGroupsConcurrently(
	ctx context.Context,
	pageGroups [][]core.Page,
	sharedCache cache.Cache,
) ([]core.Provider, error) {
	workerCount := m.config.ChunkWorkers
	if workerCount < 1 {
		workerCount = 1
	}
	workerCount = min(workerCount, len(pageGroups))

	results := make([]core.Provider, len(pageGroups))
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
					m.runRenderJob(ctx, index, pageGroups, sharedCache, results, recordErr)
				}
			}
		}()
	}

	for index := range pageGroups {
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

// runRenderJob renders one page group in an isolated scope so a panic is
// reported through recordErr instead of killing the worker goroutine.
func (m *Paper) runRenderJob(
	ctx context.Context,
	index int,
	pageGroups [][]core.Page,
	sharedCache cache.Cache,
	results []core.Provider,
	recordErr func(error),
) {
	defer func() {
		if r := recover(); r != nil {
			recordErr(fmt.Errorf("%w %d: %v", errPanicProcessingPageGroup, index, r))
		}
	}()

	provider, err := m.renderPagesToProvider(ctx, pageGroups[index], sharedCache)
	if err != nil {
		recordErr(err)
		return
	}
	results[index] = provider
}

// renderPagesToProvider renders pages into a fresh provider without
// serializing it, leaving the page content streams available for splicing.
func (m *Paper) renderPagesToProvider(
	ctx context.Context,
	pages []core.Page,
	sharedCache cache.Cache,
) (core.Provider, error) {
	innerCtx := m.pageBuilder.cell.Copy()
	provider := getProvider(sharedCache, m.config)

	for i, page := range pages {
		if err := generationCanceled(ctx); err != nil {
			return nil, err
		}
		ensureProviderPage(provider, i+1)
		page.Render(provider, innerCtx)
	}

	return provider, nil
}
