package paper

import (
	"context"
	"errors"
	"fmt"

	"github.com/avdoseferovic/paper/internal/cache"
	"github.com/avdoseferovic/paper/internal/pdf"
	paperprovider "github.com/avdoseferovic/paper/internal/providers/paper"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

// generateParallelPages renders page groups concurrently, each into its own
// document, then splices the rendered pages into a single document that is
// serialized once.
//
// Only page content streams and resource tables move between documents, so the
// per-chunk cost does not grow with the worker count. Serializing every chunk to
// PDF bytes and merging those byte streams instead would make each chunk pay for
// a full document (font programs, resource dictionary, xref) and re-parse all of
// it in the merge.
func (m *Paper) generateParallelPages(ctx context.Context) (*core.Pdf, error) {
	pageGroups, err := m.chunkedPageGroups(ctx)
	if err != nil {
		return nil, err
	}
	if len(pageGroups) == 0 {
		return m.generateSequentially(ctx)
	}

	sharedCache := cache.NewMutexDecorator(cache.New())

	rendered, err := processPageGroupsConcurrently(ctx, m.config.ChunkWorkers, pageGroups,
		func(ctx context.Context, pages []core.Page) (core.Provider, error) {
			return m.renderPagesToProvider(ctx, pages, sharedCache)
		})
	if err != nil {
		// A worker hit a feature that cannot be spliced. Every such feature is
		// configured rather than drawn, and is already routed to sequential
		// generation before a mode is chosen, so this is defence in depth
		// against one being added later. Rendering aborted as soon as the
		// feature appeared, so falling back does not pay for a full parallel
		// render first.
		if errors.Is(err, pdf.ErrAbsorbUnsupported) {
			return m.generateSequentially(ctx)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", ErrCannotGenerateInParallelMode, err)
	}

	if err := generationCanceled(ctx); err != nil {
		return nil, err
	}

	// Splicing re-checks absorbability, so this also catches anything the
	// per-page check could not see.
	doc, err := m.spliceAndSerialize(rendered)
	if errors.Is(err, pdf.ErrAbsorbUnsupported) {
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
	return core.NewPDF(documentBytes, newReport(consts.GenerationParallelPages, issues)), nil
}

// renderPagesToProvider renders pages into a fresh provider without
// serializing it, leaving the page content streams available for splicing.
//
// Spliceability is re-checked after every page so that a document using an
// unspliceable feature is detected as soon as that feature is drawn. The error
// aborts the whole worker pool, which keeps the sequential fallback from paying
// for a full parallel render first.
func (m *Paper) renderPagesToProvider(
	ctx context.Context,
	pages []core.Page,
	sharedCache cache.Cache,
) (core.Provider, error) {
	innerCtx := m.pageBuilder.cell.Copy()
	provider := getProvider(sharedCache, m.config)
	checker, canCheck := provider.(paperprovider.AbsorbabilityChecker)

	for i, page := range pages {
		if err := generationCanceled(ctx); err != nil {
			return nil, err
		}
		ensureProviderPage(provider, i+1)
		page.Render(provider, innerCtx)

		if canCheck {
			if err := checker.PagesAbsorbable(); err != nil {
				return nil, err
			}
		}
	}

	return provider, nil
}
