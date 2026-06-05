package paper_test

import (
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/v2"

	"github.com/avdoseferovic/paper/v2/pkg/core/entity"

	"github.com/avdoseferovic/paper/v2/pkg/components/text"

	"github.com/avdoseferovic/paper/v2/pkg/components/col"
	"github.com/avdoseferovic/paper/v2/pkg/components/row"
	"github.com/avdoseferovic/paper/v2/pkg/core"
	"github.com/avdoseferovic/paper/v2/pkg/tree/node"

	"github.com/avdoseferovic/paper/v2/mocks"
	"github.com/avdoseferovic/paper/v2/pkg/components/page"
	"github.com/avdoseferovic/paper/v2/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricsDecorator(t *testing.T) {
	t.Parallel()
	// Act
	sut := paper.NewMetricsDecorator(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*paper.MetricsDecorator", fmt.Sprintf("%T", sut))
}

func TestMetricsDecorator_AddPages(t *testing.T) {
	t.Parallel()
	// Arrange
	pg := page.New()

	docToReturn := mocks.NewDocument(t)
	docToReturn.EXPECT().GetBytes().Return([]byte{1, 2, 3})
	docToReturn.EXPECT().GetReport().Return(nil)
	inner := mocks.NewMaroto(t)
	inner.EXPECT().AddPages(pg)
	inner.EXPECT().Generate().Return(docToReturn, nil)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	sut.AddPages(pg)
	sut.AddPages(pg)

	// Assert
	doc, err := sut.Generate()
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_page", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
	inner.AssertNumberOfCalls(t, "AddPages", 2)
}

func TestMetricsDecorator_AddRow(t *testing.T) {
	t.Parallel()
	// Arrange
	col := col.New(12)

	docToReturn := mocks.NewDocument(t)
	docToReturn.EXPECT().GetBytes().Return([]byte{1, 2, 3})
	docToReturn.EXPECT().GetReport().Return(nil)
	inner := mocks.NewMaroto(t)
	inner.EXPECT().AddRow(10.0, col).Return(nil)
	inner.EXPECT().Generate().Return(docToReturn, nil)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	sut.AddRow(10, col)
	sut.AddRow(10, col)

	// Assert
	doc, err := sut.Generate()
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_row", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
	inner.AssertNumberOfCalls(t, "AddRow", 2)
}

func TestMetricsDecorator_GeneratePreservesRenderIssues(t *testing.T) {
	t.Parallel()

	docToReturn := core.NewPDF([]byte{1, 2, 3}, &metrics.Report{
		RenderIssues: []metrics.RenderIssue{
			{Operation: "image.load", Message: "could not load image", Error: "missing"},
		},
	})
	inner := mocks.NewMaroto(t)
	inner.EXPECT().Generate().Return(docToReturn, nil)

	sut := paper.NewMetricsDecorator(inner)

	doc, err := sut.Generate()

	assert.NoError(t, err)
	if assert.NotNil(t, doc) && assert.NotNil(t, doc.GetReport()) {
		assert.Len(t, doc.GetReport().RenderIssues, 1)
		assert.Equal(t, "image.load", doc.GetReport().RenderIssues[0].Operation)
	}
}

func TestMetricsDecorator_AddRows(t *testing.T) {
	t.Parallel()
	// Arrange
	row := row.New(10).Add(col.New(12))

	docToReturn := mocks.NewDocument(t)
	docToReturn.EXPECT().GetBytes().Return([]byte{1, 2, 3})
	docToReturn.EXPECT().GetReport().Return(nil)
	inner := mocks.NewMaroto(t)
	inner.EXPECT().AddRows(row)
	inner.EXPECT().Generate().Return(docToReturn, nil)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	sut.AddRows(row)
	sut.AddRows(row)

	// Assert
	doc, err := sut.Generate()
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "add_rows", report.TimeMetrics[1].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 2)
	inner.AssertNumberOfCalls(t, "AddRows", 2)
}

func TestMetricsDecorator_GetStructure(t *testing.T) {
	t.Parallel()
	// Arrange
	row := row.New(10).Add(col.New(12))

	docToReturn := mocks.NewDocument(t)
	docToReturn.EXPECT().GetBytes().Return([]byte{1, 2, 3})
	docToReturn.EXPECT().GetReport().Return(nil)
	inner := mocks.NewMaroto(t)
	inner.EXPECT().AddRows(row)
	inner.EXPECT().GetStructure().Return(&node.Node[core.Structure]{})
	inner.EXPECT().Generate().Return(docToReturn, nil)

	sut := paper.NewMetricsDecorator(inner)
	sut.AddRows(row)

	// Act
	_ = sut.GetStructure()

	// Assert
	doc, err := sut.Generate()
	assert.Nil(t, err)
	assert.NotNil(t, doc)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 3)
	assert.Equal(t, "get_tree_structure", report.TimeMetrics[0].Key)
	assert.Equal(t, "generate", report.TimeMetrics[1].Key)
	assert.Equal(t, "add_rows", report.TimeMetrics[2].Key)
	assert.Len(t, report.TimeMetrics[1].Times, 1)
	inner.AssertNumberOfCalls(t, "AddRows", 1)
	inner.AssertNumberOfCalls(t, "GetStructure", 1)
}

func TestMetricsDecorator_FitlnCurrentPage(t *testing.T) {
	t.Parallel()
	// Arrange
	inner := mocks.NewMaroto(t)
	inner.EXPECT().FitlnCurrentPage(10.0).Return(true)
	inner.EXPECT().FitlnCurrentPage(20.0).Return(false)

	sut := paper.NewMetricsDecorator(inner)

	// Act & Assert
	assert.True(t, sut.FitlnCurrentPage(10))
	assert.False(t, sut.FitlnCurrentPage(20))
}

func TestMetricsDecorator_GetCurrentConfig(t *testing.T) {
	t.Parallel()
	// Arrange
	cfgToReturn := &entity.Config{
		MaxGridSize: 15,
	}
	inner := mocks.NewMaroto(t)
	inner.EXPECT().GetCurrentConfig().Return(cfgToReturn)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	cfg := sut.GetCurrentConfig()

	// Assert
	assert.Equal(t, cfgToReturn.MaxGridSize, cfg.MaxGridSize)
}

func TestMetricsDecorator_RegisterHeader(t *testing.T) {
	t.Parallel()
	// Arrange
	row := text.NewRow(10, "text")

	inner := mocks.NewMaroto(t)
	inner.EXPECT().RegisterHeader(row).Return(nil)
	inner.EXPECT().Generate().Return(&core.Pdf{}, nil)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	err := sut.RegisterHeader(row)

	// Assert
	assert.Nil(t, err)

	doc, err := sut.Generate()
	assert.Nil(t, err)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "header", report.TimeMetrics[1].Key)
}

func TestMetricsDecorator_RegisterFooter(t *testing.T) {
	t.Parallel()
	// Arrange
	row := text.NewRow(10, "text")

	inner := mocks.NewMaroto(t)
	inner.EXPECT().RegisterFooter(row).Return(nil)
	inner.EXPECT().Generate().Return(&core.Pdf{}, nil)

	sut := paper.NewMetricsDecorator(inner)

	// Act
	err := sut.RegisterFooter(row)

	// Assert
	assert.Nil(t, err)

	doc, err := sut.Generate()
	assert.Nil(t, err)

	report := doc.GetReport()
	assert.NotNil(t, report)
	assert.Len(t, report.TimeMetrics, 2)
	assert.Equal(t, "generate", report.TimeMetrics[0].Key)
	assert.Equal(t, "footer", report.TimeMetrics[1].Key)
}
