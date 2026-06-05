package paper

import (
	"sync"

	"github.com/avdoseferovic/paper/v2/internal/cache"
	"github.com/avdoseferovic/paper/v2/pkg/consts/generation"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/merge"
)

// Generate is responsible to compute the component tree created by
// the usage of all other Paper methods, and generate the PDF document.
func (m *Paper) Generate() (core.Document, error) {
	m.pageBuilder.fillPageToAddNew()
	m.pageBuilder.setConfig()

	if m.config.Protection != nil {
		return m.generate()
	}

	if m.config.GenerationMode == generation.Concurrent {
		return m.generateConcurrently()
	}

	if m.config.GenerationMode == generation.SequentialLowMemory {
		return m.generateLowMemory()
	}

	return m.generate()
}

func (m *Paper) generate() (core.Document, error) {
	innerCtx := m.pageBuilder.cell.Copy()

	for i, page := range m.pageBuilder.pages {
		ensureProviderPage(m.provider, i+1)
		page.Render(m.provider, innerCtx)
	}

	documentBytes, err := m.provider.GenerateBytes()
	if err != nil {
		return nil, err
	}

	return core.NewPDF(documentBytes, nil), nil
}

func (m *Paper) generateConcurrently() (core.Document, error) {
	chunks := len(m.pageBuilder.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}
	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pageBuilder.pages); i += chunks {
		end := min(i+chunks, len(m.pageBuilder.pages))
		pageGroups = append(pageGroups, m.pageBuilder.pages[i:end])
	}

	pdfs, err := processPageGroupsConcurrently(m.config.ChunkWorkers, pageGroups, m.processPage)
	if err != nil {
		return nil, ErrCannotGenerateInParallelMode
	}

	mergedBytes, err := merge.Bytes(pdfs...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, nil), nil
}

func (m *Paper) generateLowMemory() (core.Document, error) {
	chunks := len(m.pageBuilder.pages) / m.config.ChunkWorkers
	if chunks == 0 {
		chunks = 1
	}

	pageGroups := make([][]core.Page, 0)
	for i := 0; i < len(m.pageBuilder.pages); i += chunks {
		end := min(i+chunks, len(m.pageBuilder.pages))
		pageGroups = append(pageGroups, m.pageBuilder.pages[i:end])
	}

	var pdfResults [][]byte
	for _, pageGroup := range pageGroups {
		bytes, err := m.processPage(pageGroup)
		if err != nil {
			return nil, ErrCannotGenerateInLowMemoryMode
		}

		pdfResults = append(pdfResults, bytes)
	}

	mergedBytes, err := merge.Bytes(pdfResults...)
	if err != nil {
		return nil, err
	}

	return core.NewPDF(mergedBytes, nil), nil
}

func (m *Paper) processPage(pages []core.Page) ([]byte, error) {
	innerCtx := m.pageBuilder.cell.Copy()

	innerProvider := getProvider(cache.NewMutexDecorator(cache.New()), m.config)
	for i, page := range pages {
		ensureProviderPage(innerProvider, i+1)
		page.Render(innerProvider, innerCtx)
	}

	return innerProvider.GenerateBytes()
}

func processPageGroupsConcurrently(
	workerCount int,
	pageGroups [][]core.Page,
	processor func([]core.Page) ([]byte, error),
) ([][]byte, error) {
	if len(pageGroups) == 0 {
		return nil, nil
	}
	if workerCount < 1 {
		workerCount = 1
	}
	workerCount = min(workerCount, len(pageGroups))

	results := make([][]byte, len(pageGroups))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for index := range jobs {
				result, err := processor(pageGroups[index])
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					continue
				}
				results[index] = result
			}
		}()
	}

	for index := range pageGroups {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func ensureProviderPage(provider core.Provider, pageNumber int) {
	if pp, ok := provider.(core.PageProvider); ok {
		pp.EnsurePage(pageNumber)
	}
}
