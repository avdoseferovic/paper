package paper_test

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/avdoseferovic/paper/pkg/components/code"
	componentimage "github.com/avdoseferovic/paper/pkg/components/image"
	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/consts/protection"
	"github.com/avdoseferovic/paper/pkg/core"
	coreentity "github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/test"
	"github.com/avdoseferovic/paper/pkg/tree/node"
	"go.uber.org/goleak"

	"github.com/avdoseferovic/paper"

	"github.com/stretchr/testify/assert"
)

// TestMain runs the package test binary under goleak so any goroutine leaked
// by the concurrent generation worker pool fails the suite deterministically.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type errorReader struct {
	err error
}

func (r errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("new default", func(t *testing.T) {
		t.Parallel()
		// Act
		sut := paper.New()

		// Assert
		assert.NotNil(t, sut)
		assert.Equal(t, "*paper.Paper", fmt.Sprintf("%T", sut))
	})
	t.Run("when config is sent, it should create Paper object", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := config.NewBuilder().
			Build()

		// Act
		sut := paper.New(cfg)

		// Assert
		assert.NotNil(t, sut)
		assert.Equal(t, "*paper.Paper", fmt.Sprintf("%T", sut))
	})
	t.Run("when config with an concurrent mode is sent, should create Paper object", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := config.NewBuilder().
			WithConcurrentMode(7).
			Build()

		// Act
		sut := paper.New(cfg)

		// Assert
		assert.NotNil(t, sut)
		assert.Equal(t, "*paper.Paper", fmt.Sprintf("%T", sut))
	})
	t.Run("when config with an low memory mode is sent, should create Paper object", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cfg := config.NewBuilder().
			WithSequentialLowMemoryMode(10).
			Build()

		// Act
		sut := paper.New(cfg)

		// Assert
		assert.NotNil(t, sut)
		assert.Equal(t, "*paper.Paper", fmt.Sprintf("%T", sut))
	})
}

func TestFromHTML(t *testing.T) {
	t.Parallel()

	t.Run("generates a PDF document from HTML", func(t *testing.T) {
		t.Parallel()

		doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)

		assert.NoError(t, err)
		if assert.NotNil(t, doc) {
			assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
			assert.Greater(t, len(doc.GetBytes()), 1000)
		}
	})

	t.Run("accepts the same config shape as New", func(t *testing.T) {
		t.Parallel()

		cfg := config.NewBuilder().
			WithMaxGridSize(20).
			Build()

		doc, err := paper.FromHTML(`<div style="display:flex"><p>A</p><p>B</p><p>C</p></div>`, cfg)

		assert.NoError(t, err)
		if assert.NotNil(t, doc) {
			assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
		}
	})
}

func TestFromHTMLReader(t *testing.T) {
	t.Parallel()

	t.Run("generates a PDF document from a reader", func(t *testing.T) {
		t.Parallel()

		doc, err := paper.FromHTMLReader(strings.NewReader(`<p>reader input</p>`))

		assert.NoError(t, err)
		if assert.NotNil(t, doc) {
			assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
		}
	})

	t.Run("returns read errors", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("read failed")

		doc, err := paper.FromHTMLReader(errorReader{err: wantErr})

		assert.Nil(t, doc)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMaroto_AddRow(t *testing.T) {
	t.Parallel()
	t.Run("When row height and available sapacing are equals, should add row in current page", func(t *testing.T) {
		t.Parallel()
		cfg := config.NewBuilder().
			WithDimensions(20, 20).
			WithBottomMargin(0).
			WithTopMargin(0).
			WithLeftMargin(0).
			WithRightMargin(0).
			Build()
		sut := paper.New(cfg)
		// Act
		sut.AddRow(19)
		sut.AddRow(1)

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_row.json")
	})
	t.Run("when col is not sent, should empty col is set", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()
		// Act
		sut.AddRow(10)

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_row_4.json")
	})
	t.Run("when one row is sent, should create one row", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRow(10, col.New(12))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_row_1.json")
	})
	t.Run("when two rows are sent, should create two rows", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRow(10, col.New(12))
		sut.AddRow(10, col.New(12))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_row_2.json")
	})
	t.Run("when rows do not fit on the current page, should create a new page", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_row_3.json")
	})
}

func TestMaroto_AddRows(t *testing.T) {
	t.Parallel()
	t.Run("when col is not sent, should empty col is set", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRows(row.New(15))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_rows_4.json")
	})
	t.Run("when one row is sent, should create one row", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRows(row.New(15).Add(col.New(12)))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_rows_1.json")
	})
	t.Run("when two rows are sent, should create two rows", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRows(row.New(15).Add(col.New(12)))
		sut.AddRows(row.New(15).Add(col.New(12)))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_rows_2.json")
	})
	t.Run("when rows do not fit on the current page, should create a new page", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		for range 20 {
			sut.AddRows(row.New(15).Add(col.New(12)))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_rows_3.json")
	})

	t.Run("when autoRow is sent, should set autoRow", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		for range 20 {
			sut.AddRows(row.New().Add(text.NewCol(12, "teste")))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_rows_5.json")
	})
}

func TestMaroto_AddAutoRow(t *testing.T) {
	t.Parallel()
	t.Run("When 100 automatic rows are sent, it should create 2 pages", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		for range 150 {
			sut.AddAutoRow(text.NewCol(12, "teste"))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_auto_row_1.json")
	})
}

func TestMaroto_AddPages(t *testing.T) {
	t.Parallel()
	t.Run("when a new page is created, should add a page", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddPages(
			page.New().Add(
				row.New(20).Add(col.New(12)),
			),
		)

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_pages_1.json")
	})
	t.Run("when two pages are created, should add two pages", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()

		// Act
		sut.AddPages(
			page.New().Add(
				row.New(20).Add(col.New(12)),
			),
			page.New().Add(
				row.New(20).Add(col.New(12)),
			),
		)

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_pages_2.json")
	})
	t.Run("when the sent page uses two pages, two pages are created", func(t *testing.T) {
		t.Parallel()
		// Arrange
		sut := paper.New()
		var rows []core.Row
		for range 15 {
			rows = append(rows, row.New(20).Add(col.New(12)))
		}

		// Act
		sut.AddPages(page.New().Add(rows...))

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_add_pages_3.json")
	})
}

// nolint:paralleltest // generate cannot be tested in parallel
func TestMaroto_Generate(t *testing.T) {
	t.Run("when one row is sent, should generate one row", func(t *testing.T) {
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRow(10, col.New(12))

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
	})
	t.Run("when two row are sent, should generate two row", func(t *testing.T) {
		// Arrange
		sut := paper.New()

		// Act
		sut.AddRow(10, col.New(12))
		sut.AddRow(10, col.New(12))

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
	})
	t.Run("when rows do not fit on the current page, should generate two pages", func(t *testing.T) {
		// Arrange
		sut := paper.New()

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
	})
	t.Run("when rows do not fit on the current page and concurrent mode is active, should executed in parallel", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithConcurrentMode(7).
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
	})
	t.Run("when protection and concurrent mode are active, should generate protected PDF bytes", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithConcurrentMode(7).
			WithProtection(protection.None, "user", "owner").
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRows(text.NewRow(10, "protected concurrent"))
		}

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		if assert.NotNil(t, doc) {
			assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
		}
	})
	t.Run("when two pages are sent and low memory mode is active, should executed in low memory mode", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithSequentialLowMemoryMode(10).
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		assert.NotNil(t, doc)
	})
	t.Run("when protection and sequential low memory mode are active, should generate protected PDF bytes", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithSequentialLowMemoryMode(10).
			WithProtection(protection.None, "user", "owner").
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRows(text.NewRow(10, "protected low memory"))
		}

		// Assert
		doc, err := sut.Generate()
		assert.Nil(t, err)
		if assert.NotNil(t, doc) {
			assert.True(t, bytes.HasPrefix(doc.GetBytes(), []byte("%PDF-")))
		}
	})
	t.Run("when two pages are sent and sequential generation is active, should executed in sequential generation mode", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithSequentialMode().
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_sequential.json")
	})
	t.Run("when two pages are sent and sequential low memory is active, should executed in sequential low memory mode", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithSequentialLowMemoryMode(10).
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_sequential_low_memory.json")
	})
	t.Run("when two pages are sent and concurrent mode is active, should executed in parallel", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithConcurrentMode(10).
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_concurrent.json")
	})
	t.Run("goroutines do not leak after multiple generate calls on concurrent mode", func(t *testing.T) {
		// goleak polls with backoff for goroutines to settle and filters
		// runtime/test-framework goroutines, replacing the previous flaky
		// time.Sleep + runtime.NumGoroutine() equality check (which was racy
		// because NumGoroutine is process-global).
		defer goleak.VerifyNone(t)

		// Arrange
		cfg := config.NewBuilder().
			WithConcurrentMode(10).
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}
		_, err1 := sut.Generate()
		_, err2 := sut.Generate()
		_, err3 := sut.Generate()

		// Assert
		assert.Nil(t, err1)
		assert.Nil(t, err2)
		assert.Nil(t, err3)
	})
	t.Run("when two pages are sent and page number is active, should add page number", func(t *testing.T) {
		// Arrange
		cfg := config.NewBuilder().
			WithPageNumber().
			Build()

		sut := paper.New(cfg)

		// Act
		for range 30 {
			sut.AddRow(10, col.New(12))
		}

		// Assert
		test.New(t).Assert(sut.GetStructure()).Equals("paper_page_number.json")
	})
}

func TestPaper_GetStructureIsIdempotent(t *testing.T) {
	t.Parallel()

	sut := paper.New()
	sut.AddRow(10, col.New(12))

	firstPageCount := structurePageCount(sut.GetStructure())
	secondPageCount := structurePageCount(sut.GetStructure())

	assert.Equal(t, 1, firstPageCount)
	assert.Equal(t, firstPageCount, secondPageCount)
}

func TestPaper_GetStructureBeforeGenerateDoesNotAddBlankPage(t *testing.T) {
	t.Parallel()

	sut := paper.New()
	sut.AddRow(10, col.New(12))

	structurePages := structurePageCount(sut.GetStructure())
	doc, err := sut.Generate()

	assert.NoError(t, err)
	if assert.NotNil(t, doc) {
		assert.Equal(t, structurePages, pdfPageCount(t, doc.GetBytes()))
	}
	assert.Equal(t, structurePages, structurePageCount(sut.GetStructure()))
}

func TestPaper_GenerateRepeatedCallsSequential(t *testing.T) {
	t.Parallel()

	assertRepeatedGenerateStable(t, config.NewBuilder().WithSequentialMode().Build())
}

func TestPaper_GenerateRepeatedCallsProtectedSequential(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().
		WithSequentialMode().
		WithProtection(protection.None, "user", "owner").
		Build()

	assertRepeatedGenerateStable(t, cfg)
}

func TestPaper_GenerateRepeatedCallsConcurrent(t *testing.T) {
	t.Parallel()

	assertRepeatedGenerateStable(t, config.NewBuilder().WithConcurrentMode(2).Build())
}

func TestPaper_GenerateRepeatedCallsLowMemory(t *testing.T) {
	t.Parallel()

	assertRepeatedGenerateStable(t, config.NewBuilder().WithSequentialLowMemoryMode(2).Build())
}

func assertRepeatedGenerateStable(t *testing.T, cfg *coreentity.Config) {
	t.Helper()

	sut := paper.New(cfg)
	for range 30 {
		sut.AddRow(10, col.New(12))
	}

	first, err := sut.Generate()
	assert.NoError(t, err)
	if !assert.NotNil(t, first) {
		return
	}

	second, err := sut.Generate()
	assert.NoError(t, err)
	if !assert.NotNil(t, second) {
		return
	}

	firstPageCount := pdfPageCount(t, first.GetBytes())
	secondPageCount := pdfPageCount(t, second.GetBytes())
	assert.Equal(t, firstPageCount, secondPageCount)
	assert.Equal(t, firstPageCount, structurePageCount(sut.GetStructure()))
}

func structurePageCount(tree *node.Node[core.Structure]) int {
	if tree == nil {
		return 0
	}
	count := 0
	for _, child := range tree.GetNexts() {
		if child.GetData().Type == "page" {
			count++
		}
	}
	return count
}

var pdfPageRe = regexp.MustCompile(`/Type\s*/Page(?:\s|/|>)`)

func pdfPageCount(t *testing.T, pdfBytes []byte) int {
	t.Helper()

	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")))
	count := len(pdfPageRe.FindAll(pdfBytes, -1))
	assert.Greater(t, count, 0)
	return count
}

func TestPaper_GenerateReportsProviderFallbackIssues(t *testing.T) {
	t.Run("sequential generation reports image fallback issue", func(t *testing.T) {
		sut := paper.New()
		sut.AddRow(20, componentimage.NewFromFileCol(12, "missing-image.png"))

		doc, err := sut.Generate()

		assert.NoError(t, err)
		if assert.NotNil(t, doc) && assert.NotNil(t, doc.GetReport()) {
			assert.NotEmpty(t, doc.GetBytes())
			assert.Len(t, doc.GetReport().RenderIssues, 1)
			assert.Equal(t, "image.load", doc.GetReport().RenderIssues[0].Operation)
			assert.Equal(t, "could not load image", doc.GetReport().RenderIssues[0].Message)
			assert.NotEmpty(t, doc.GetReport().RenderIssues[0].Error)
		}
	})

	t.Run("concurrent and low memory generation aggregate image fallback issues", func(t *testing.T) {
		for name, cfg := range map[string]*coreentity.Config{
			"concurrent": config.NewBuilder().WithConcurrentMode(2).Build(),
			"low-memory": config.NewBuilder().WithSequentialLowMemoryMode(2).Build(),
		} {
			t.Run(name, func(t *testing.T) {
				sut := paper.New(cfg)
				sut.AddRow(20, componentimage.NewFromFileCol(12, "missing-image.png"))

				doc, err := sut.Generate()

				assert.NoError(t, err)
				if assert.NotNil(t, doc) && assert.NotNil(t, doc.GetReport()) {
					assert.Len(t, doc.GetReport().RenderIssues, 1)
					assert.Equal(t, "image.load", doc.GetReport().RenderIssues[0].Operation)
				}
			})
		}
	})
}

func TestMaroto_FitlnCurrentPage(t *testing.T) {
	t.Parallel()
	t.Run("when component is smaller should available size, should return false", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(210.0, 297.0). // A4 we have 266.9975 of useful height
			Build())

		var rows []core.Row
		for range 26 {
			rows = append(rows, row.New(10).Add(col.New(12)))
		}

		sut.AddPages(page.New().Add(rows...))
		assert.False(t, sut.FitlnCurrentPage(10))
	})
	t.Run("when component is larger should the available size, should return true", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(210.0, 297.0).
			Build())

		var rows []core.Row
		for range 10 {
			rows = append(rows, row.New(10).Add(col.New(12)))
		}

		sut.AddPages(page.New().Add(rows...))
		assert.True(t, sut.FitlnCurrentPage(40))
	})
	t.Run("when it have content with an automatic height of 10 and the height sent fits the current page, it should return true",
		func(t *testing.T) {
			t.Parallel()
			sut := paper.New(config.NewBuilder().
				WithDimensions(210.0, 297.0).
				Build())

			var rows []core.Row
			for range 10 {
				rows = append(rows, row.New().Add(text.NewCol(12, "teste")))
			}

			sut.AddPages(page.New().Add(rows...))
			assert.True(t, sut.FitlnCurrentPage(40))
		})
}

func TestMaroto_GetCurrentConfig(t *testing.T) {
	t.Parallel()
	t.Run("When GetCurrentConfig is called, should return the current settings", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithMaxGridSize(20).
			Build())

		assert.Equal(t, 20, sut.GetCurrentConfig().MaxGridSize)
	})
}

func TestPaper_NormalizesCallerSuppliedInvalidMaxGridSize(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().WithMaxGridSize(8).Build()
	cfg.MaxGridSize = 0

	sut := paper.New(cfg)
	cfg.MaxGridSize = 99

	assert.Equal(t, 12, sut.GetCurrentConfig().MaxGridSize)
}

func TestPaper_InvalidCallerSuppliedMaxGridSizeDoesNotBreakHTMLOrFinalization(t *testing.T) {
	t.Parallel()

	cfg := config.NewBuilder().Build()
	cfg.MaxGridSize = 0
	sut := paper.New(cfg)

	err := sut.AddHTML(`<div style="display:flex"><p>A</p><p>B</p></div>`)
	assert.NoError(t, err)

	doc, err := sut.Generate()
	assert.NoError(t, err)
	if assert.NotNil(t, doc) {
		assert.Equal(t, 1, pdfPageCount(t, doc.GetBytes()))
		assert.Equal(t, 1, structurePageCount(sut.GetStructure()))
	}
}

// nolint:dupl // dupl is good here
func TestMaroto_RegisterHeader(t *testing.T) {
	t.Parallel()
	t.Run("when header size is greater than useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterHeader(row.New(1000))

		assert.NotNil(t, err)
		assert.Equal(t, "header height is greater than page useful area", err.Error())
	})
	t.Run("when margins make header larger than useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(100, 100).
			WithTopMargin(20).
			WithBottomMargin(20).
			WithLeftMargin(0).
			WithRightMargin(0).
			Build())

		err := sut.RegisterHeader(row.New(70))

		assert.ErrorIs(t, err, paper.ErrHeaderHeightIsGreaterThanUsefulArea)
	})
	t.Run("when header plus registered footer exceeds useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(100, 100).
			WithTopMargin(20).
			WithBottomMargin(20).
			WithLeftMargin(0).
			WithRightMargin(0).
			Build())

		assert.NoError(t, sut.RegisterFooter(row.New(40)))

		err := sut.RegisterHeader(row.New(30))

		assert.ErrorIs(t, err, paper.ErrHeaderHeightIsGreaterThanUsefulArea)
	})
	t.Run("when header size is correct, should not return error and apply header", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterHeader(code.NewBarRow(10, "header"))

		var rows []core.Row
		for range 5 {
			rows = append(rows, row.New(100).Add(col.New(12)))
		}

		sut.AddRows(rows...)

		// Assert
		assert.Nil(t, err)
		test.New(t).Assert(sut.GetStructure()).Equals("header.json")
	})
	t.Run("when autoRow is sent, should set autoRow", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterHeader(text.NewAutoRow("header"))

		var rows []core.Row
		for range 5 {
			rows = append(rows, row.New(100).Add(col.New(12)))
		}

		sut.AddRows(rows...)

		// Assert
		assert.Nil(t, err)
		test.New(t).Assert(sut.GetStructure()).Equals("header_auto_row.json")
	})
}

// nolint:dupl // dupl is good here
func TestMaroto_RegisterFooter(t *testing.T) {
	t.Parallel()
	t.Run("when footer size is greater than useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterFooter(row.New(1000))

		assert.NotNil(t, err)
		assert.Equal(t, "footer height is greater than page useful area", err.Error())
	})
	t.Run("when margins make footer larger than useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(100, 100).
			WithTopMargin(20).
			WithBottomMargin(20).
			WithLeftMargin(0).
			WithRightMargin(0).
			Build())

		err := sut.RegisterFooter(row.New(70))

		assert.ErrorIs(t, err, paper.ErrFooterHeightIsGreaterThanUsefulArea)
	})
	t.Run("when footer plus registered header exceeds useful area, should return error", func(t *testing.T) {
		t.Parallel()
		sut := paper.New(config.NewBuilder().
			WithDimensions(100, 100).
			WithTopMargin(20).
			WithBottomMargin(20).
			WithLeftMargin(0).
			WithRightMargin(0).
			Build())

		assert.NoError(t, sut.RegisterHeader(row.New(40)))

		err := sut.RegisterFooter(row.New(30))

		assert.ErrorIs(t, err, paper.ErrFooterHeightIsGreaterThanUsefulArea)
	})
	t.Run("when header size is correct, should not return error and apply header", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterFooter(code.NewBarRow(10, "footer"))

		var rows []core.Row
		for range 5 {
			rows = append(rows, row.New(100).Add(col.New(12)))
		}

		sut.AddRows(rows...)

		// Assert
		assert.Nil(t, err)
		test.New(t).Assert(sut.GetStructure()).Equals("footer.json")
	})
	t.Run("when autoRow is sent, should set autoRow", func(t *testing.T) {
		t.Parallel()
		sut := paper.New()

		err := sut.RegisterFooter(text.NewAutoRow("header"))

		var rows []core.Row
		for range 5 {
			rows = append(rows, row.New(100).Add(col.New(12)))
		}

		sut.AddRows(rows...)

		// Assert
		assert.Nil(t, err)
		test.New(t).Assert(sut.GetStructure()).Equals("footer_auto_row.json")
	})
}
