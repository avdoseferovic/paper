package paper

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

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

	// Protection and document-catalog features (forms, PDF/A, tagged PDF,
	// attachments, ...) are emitted once per document; chunked generation
	// would lose them, so both force sequential mode.
	if m.config.Protection != nil || m.config.RequiresSingleDocumentGeneration() {
		return m.generateSequentially(ctx)
	}

	// Parallel pages serializes a single spliced document, so it reproduces
	// deterministic output; the chunk-and-merge low memory path does not.
	if m.config.GenerationMode == consts.GenerationParallelPages {
		return m.generateParallelPages(ctx)
	}

	if m.config.GenerationMode == consts.GenerationSequentialLowMemory && !m.config.Deterministic {
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

	return core.NewPDF(documentBytes, newReport(consts.GenerationSequential, collectRenderIssues(provider))), nil
}

// chunkedPageGroups splits the built pages into ChunkWorkers groups of equal
// size (the last group takes the remainder), aborting on context cancellation.
func (m *Paper) chunkedPageGroups(ctx context.Context) ([][]core.Page, error) {
	chunks := cmp.Or(len(m.pageBuilder.pages)/m.config.ChunkWorkers, 1)
	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pageBuilder.pages); i += chunks {
		if err := generationCanceled(ctx); err != nil {
			return nil, err
		}
		end := min(i+chunks, len(m.pageBuilder.pages))
		pageGroups = append(pageGroups, m.pageBuilder.pages[i:end])
	}
	return pageGroups, nil
}

func (m *Paper) generateLowMemory(ctx context.Context) (*core.Pdf, error) {
	pageGroups, err := m.chunkedPageGroups(ctx)
	if err != nil {
		return nil, err
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
	if err := generationCanceled(ctx); err != nil {
		return nil, err
	}
	mergedBytes, err := merge.Bytes(ctx, pdfResults...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, newReport(consts.GenerationSequentialLowMemory, issues)), nil
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

// newReport builds the document's metrics report. The generation mode is recorded
// because it is not always the configured one: a document using a feature the
// configured mode cannot reproduce falls back to sequential generation, and
// callers (and tests) otherwise have no way to tell that happened.
func newReport(mode consts.GenerationMode, issues []metrics.RenderIssue) *metrics.Report {
	return &metrics.Report{
		GenerationMode: mode,
		RenderIssues:   slices.Clone(issues),
	}
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
