package paper

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/avdoseferovic/paper/internal/cache"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/merge"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

var errPanicProcessingPageGroup = errors.New("panic processing page group")

type pageProcessResult struct {
	bytes  []byte
	issues []metrics.RenderIssue
}

// Generate is responsible to compute the component tree created by the
// usage of all other Paper methods, and generate the PDF document.
// It observes ctx between pages and generation phases.
func (m *Paper) Generate(ctx context.Context) (*core.Pdf, error) {
	return m.generateDocument(ctx)
}

func (m *Paper) generateDocument(ctx context.Context) (*core.Pdf, error) {
	err := generationCanceled(ctx)
	if err != nil {
		return nil, err
	}

	m.pageBuilder.finalize()

	err = generationCanceled(ctx)
	if err != nil {
		return nil, err
	}

	if m.config.Protection != nil {
		return m.generateSequentially(ctx)
	}

	if m.config.GenerationMode == consts.GenerationConcurrent {
		return m.generateConcurrently(ctx)
	}

	if m.config.GenerationMode == consts.GenerationSequentialLowMemory {
		return m.generateLowMemory(ctx)
	}

	return m.generateSequentially(ctx)
}

func (m *Paper) generateSequentially(ctx context.Context) (*core.Pdf, error) {
	provider := getProvider(m.cache, m.config)
	innerCtx := m.pageBuilder.cell.Copy()

	for i, page := range m.pageBuilder.pages {
		err := generationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		ensureProviderPage(provider, i+1)
		page.Render(provider, innerCtx)
	}

	err := generationCanceled(ctx)
	if err != nil {
		return nil, err
	}

	documentBytes, err := provider.GenerateBytes()
	if err != nil {
		return nil, err
	}

	return core.NewPDF(documentBytes, reportFromIssues(collectRenderIssues(provider))), nil
}

func (m *Paper) generateConcurrently(ctx context.Context) (*core.Pdf, error) {
	chunks := len(m.pageBuilder.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}
	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pageBuilder.pages); i += chunks {
		err := generationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		end := min(i+chunks, len(m.pageBuilder.pages))
		pageGroups = append(pageGroups, m.pageBuilder.pages[i:end])
	}

	results, err := processPageGroupsConcurrently(ctx, m.config.ChunkWorkers, pageGroups, m.processPages)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrCannotGenerateInParallelMode
	}

	err = generationCanceled(ctx)
	if err != nil {
		return nil, err
	}

	pdfs, issues := splitPageProcessResults(results)
	mergedBytes, err := merge.Bytes(ctx, pdfs...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, reportFromIssues(issues)), nil
}

func (m *Paper) generateLowMemory(ctx context.Context) (*core.Pdf, error) {
	chunks := len(m.pageBuilder.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}

	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pageBuilder.pages); i += chunks {
		err := generationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		end := min(i+chunks, len(m.pageBuilder.pages))
		pageGroups = append(pageGroups, m.pageBuilder.pages[i:end])
	}

	var results []pageProcessResult
	for _, pageGroup := range pageGroups {
		err := generationCanceled(ctx)
		if err != nil {
			return nil, err
		}
		result, err := m.processPages(ctx, pageGroup)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			return nil, ErrCannotGenerateInLowMemoryMode
		}

		results = append(results, result)
	}

	pdfResults, issues := splitPageProcessResults(results)
	err := generationCanceled(ctx)
	if err != nil {
		return nil, err
	}
	mergedBytes, err := merge.Bytes(ctx, pdfResults...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, reportFromIssues(issues)), nil
}

func (m *Paper) processPages(ctx context.Context, pages []core.Page) (pageProcessResult, error) {
	innerCtx := m.pageBuilder.cell.Copy()

	innerProvider := getProvider(cache.NewMutexDecorator(cache.New()), m.config)
	for i, page := range pages {
		err := generationCanceled(ctx)
		if err != nil {
			return pageProcessResult{}, err
		}
		ensureProviderPage(innerProvider, i+1)
		page.Render(innerProvider, innerCtx)
	}

	err := generationCanceled(ctx)
	if err != nil {
		return pageProcessResult{}, err
	}

	bytes, err := innerProvider.GenerateBytes()
	if err != nil {
		return pageProcessResult{}, err
	}
	return pageProcessResult{
		bytes:  bytes,
		issues: collectRenderIssues(innerProvider),
	}, nil
}

func processPageGroupsConcurrently(
	ctx context.Context,
	workerCount int,
	pageGroups [][]core.Page,
	processor func(context.Context, []core.Page) (pageProcessResult, error),
) ([]pageProcessResult, error) {
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

	results := make([]pageProcessResult, len(pageGroups))
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
					runPageGroupJob(ctx, index, pageGroups, processor, results, recordErr)
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

// runPageGroupJob processes a single page group in an isolated scope so that a
// panic in the processor is recovered and reported via recordErr instead of
// crashing the worker goroutine (recover only works within the same goroutine).
func runPageGroupJob(
	ctx context.Context,
	index int,
	pageGroups [][]core.Page,
	processor func(context.Context, []core.Page) (pageProcessResult, error),
	results []pageProcessResult,
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

func generationCanceled(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("paper: generation canceled: %w", err)
	}
	return nil
}

func ensureProviderPage(provider core.Provider, pageNumber int) {
	if pp, ok := provider.(core.PageProvider); ok {
		pp.EnsurePage(pageNumber)
	}
}

func collectRenderIssues(provider core.Provider) []metrics.RenderIssue {
	issueProvider, ok := provider.(core.RenderIssueProvider)
	if !ok {
		return nil
	}
	return issueProvider.RenderIssues()
}

func reportFromIssues(issues []metrics.RenderIssue) *metrics.Report {
	if len(issues) == 0 {
		return nil
	}
	return &metrics.Report{RenderIssues: append([]metrics.RenderIssue(nil), issues...)}
}

func splitPageProcessResults(results []pageProcessResult) ([][]byte, []metrics.RenderIssue) {
	pdfs := make([][]byte, 0, len(results))
	issueCount := 0
	for _, result := range results {
		issueCount += len(result.issues)
	}
	issues := make([]metrics.RenderIssue, 0, issueCount)
	for _, result := range results {
		pdfs = append(pdfs, result.bytes)
		issues = append(issues, result.issues...)
	}
	return pdfs, issues
}
