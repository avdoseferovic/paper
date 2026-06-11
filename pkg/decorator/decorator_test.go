package decorator_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/pkg/components/text"

	"github.com/avdoseferovic/paper/pkg/components/col"
	"github.com/avdoseferovic/paper/pkg/components/row"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/tree/node"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/pkg/components/page"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

type contextKey struct{}

func TestNewMetrics(t *testing.T) {
	t.Parallel()
	// Act
	sut := decorator.NewMetrics(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*decorator.Metrics", fmt.Sprintf("%T", sut))
}

func TestMetrics_AddPages(t *testing.T) {
	t.Parallel()
	// Arrange
	pg := page.New()

	docToReturn := core.NewPDF([]byte{1, 2, 3}, nil)
	inner := mocks.NewPaper(t)
	inner.EXPECT().AddPages(pg).Times(2)
	inner.EXPECT().Generate(context.Background()).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)

	// Act
	sut.AddPages(pg)
	sut.AddPages(pg)

	// Assert
	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_page", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
}

func TestMetrics_AddRow(t *testing.T) {
	t.Parallel()
	// Arrange
	col := col.New(12)

	docToReturn := core.NewPDF([]byte{1, 2, 3}, nil)
	inner := mocks.NewPaper(t)
	inner.EXPECT().AddRow(10.0, col).Return(nil).Times(2)
	inner.EXPECT().Generate(context.Background()).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)

	// Act
	sut.AddRow(10, col)
	sut.AddRow(10, col)

	// Assert
	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_row", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
}

func TestMetrics_GeneratePreservesRenderIssues(t *testing.T) {
	t.Parallel()

	docToReturn := core.NewPDF([]byte{1, 2, 3}, &metrics.Report{
		RenderIssues: []metrics.RenderIssue{
			{Operation: "image.load", Message: "could not load image", Error: "missing"},
		},
	})
	inner := mocks.NewPaper(t)
	inner.EXPECT().Generate(context.Background()).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)

	doc, err := sut.Generate(context.Background())

	assert.NoError(t, err)
	if assert.NotNil(t, doc) && assert.NotNil(t, doc.GetReport()) {
		assert.Len(t, doc.GetReport().RenderIssues, 1)
		assert.Equal(t, "image.load", doc.GetReport().RenderIssues[0].Operation)
	}
}

func TestMetrics_GenerateDelegatesContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), contextKey{}, "request")
	docToReturn := core.NewPDF([]byte{1, 2, 3}, nil)
	inner := mocks.NewPaper(t)
	inner.EXPECT().Generate(ctx).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)

	doc, err := sut.Generate(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, doc)
}

func TestMetrics_AddHTMLDelegatesContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), contextKey{}, "request")
	inner := mocks.NewPaper(t)
	inner.EXPECT().AddHTML(ctx, "<p>hello</p>").Return(nil)

	sut := decorator.NewMetrics(inner)

	err := sut.AddHTML(ctx, "<p>hello</p>")

	assert.NoError(t, err)
}

func TestMetrics_AddRows(t *testing.T) {
	t.Parallel()
	// Arrange
	row := row.New(10).Add(col.New(12))

	docToReturn := core.NewPDF([]byte{1, 2, 3}, nil)
	inner := mocks.NewPaper(t)
	inner.EXPECT().AddRows(row).Times(2)
	inner.EXPECT().Generate(context.Background()).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)

	// Act
	sut.AddRows(row)
	sut.AddRows(row)

	// Assert
	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_rows", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
}

func TestMetrics_GetStructure(t *testing.T) {
	t.Parallel()
	// Arrange
	row := row.New(10).Add(col.New(12))

	docToReturn := core.NewPDF([]byte{1, 2, 3}, nil)
	inner := mocks.NewPaper(t)
	inner.EXPECT().AddRows(row).Once()
	inner.EXPECT().GetStructure().Return(&node.Node[core.Structure]{}).Once()
	inner.EXPECT().Generate(context.Background()).Return(docToReturn, nil)

	sut := decorator.NewMetrics(inner)
	sut.AddRows(row)

	// Act
	_ = sut.GetStructure()

	// Assert
	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 3)
	assert.Equal(t, "get_tree_structure", report.TimeMetrics[0].Key)
	assert.Equal(t, "generate", report.TimeMetrics[1].Key)
	assert.Equal(t, "add_rows", report.TimeMetrics[2].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 1)
}

func TestMetrics_FitInCurrentPage(t *testing.T) {
	t.Parallel()
	// Arrange
	inner := mocks.NewPaper(t)
	inner.EXPECT().FitInCurrentPage(10.0).Return(true)
	inner.EXPECT().FitInCurrentPage(20.0).Return(false)

	sut := decorator.NewMetrics(inner)

	// Act & Assert
	assert.True(t, sut.FitInCurrentPage(10))
	assert.False(t, sut.FitInCurrentPage(20))
}

func TestMetrics_GetCurrentConfig(t *testing.T) {
	t.Parallel()
	// Arrange
	cfgToReturn := &entity.Config{
		MaxGridSize: 15,
	}
	inner := mocks.NewPaper(t)
	inner.EXPECT().GetCurrentConfig().Return(cfgToReturn)

	sut := decorator.NewMetrics(inner)

	// Act
	cfg := sut.GetCurrentConfig()

	// Assert
	assert.Equal(t, cfgToReturn.MaxGridSize, cfg.MaxGridSize)
}

func TestMetrics_RegisterHeader(t *testing.T) {
	t.Parallel()
	// Arrange
	row := text.NewRow(10, "text")

	inner := mocks.NewPaper(t)
	inner.EXPECT().RegisterHeader(row).Return(nil)
	inner.EXPECT().Generate(context.Background()).Return(&core.Pdf{}, nil)

	sut := decorator.NewMetrics(inner)

	// Act
	err := sut.RegisterHeader(row)

	// Assert
	assert.Nil(t, err)

	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "header", report.TimeMetrics[1].Key)
}

func TestMetrics_RegisterFooter(t *testing.T) {
	t.Parallel()
	// Arrange
	row := text.NewRow(10, "text")

	inner := mocks.NewPaper(t)
	inner.EXPECT().RegisterFooter(row).Return(nil)
	inner.EXPECT().Generate(context.Background()).Return(&core.Pdf{}, nil)

	sut := decorator.NewMetrics(inner)

	// Act
	err := sut.RegisterFooter(row)

	// Assert
	assert.Nil(t, err)

	doc, err := sut.Generate(context.Background())
	assert.Nil(t, err)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "footer", report.TimeMetrics[1].Key)
}
