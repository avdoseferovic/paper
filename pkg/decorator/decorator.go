// Package decorator provides decorators over the core.Paper interface.
//
// It lives in its own package (rather than the root paper package) because it
// depends only on the core.Paper interface and pulls in pkg/metrics — keeping
// it here avoids coupling the root package to metrics and sidesteps the
// pkg/core -> pkg/metrics import direction.
package decorator

import (
	"context"

	"github.com/avdoseferovic/paper/internal/time"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/metrics"
	"github.com/avdoseferovic/paper/pkg/tree/node"
)

// Metrics is a core.Paper decorator that records timing metrics for each
// decorated operation and attaches a metrics.Report to the generated document.
type Metrics struct {
	addRowsTime    []*metrics.Time
	addRowTime     []*metrics.Time
	addAutoRowTime []*metrics.Time
	addPageTime    []*metrics.Time
	headerTime     *metrics.Time
	footerTime     *metrics.Time
	generateTime   *metrics.Time
	structureTime  *metrics.Time
	inner          core.Paper
}

// NewMetrics creates a metrics-recording decorator around the given paper
// instance.
func NewMetrics(inner core.Paper) core.Paper {
	return &Metrics{
		inner: inner,
	}
}

// FitInCurrentPage decorates the FitInCurrentPage method of paper instance.
func (m *Metrics) FitInCurrentPage(heightNewLine float64) bool {
	return m.inner.FitInCurrentPage(heightNewLine)
}

// GetCurrentConfig decorates the GetCurrentConfig method of paper instance.
func (m *Metrics) GetCurrentConfig() *entity.Config {
	return m.inner.GetCurrentConfig()
}

// Generate decorates the Generate method of paper instance.
func (m *Metrics) Generate() (*core.Pdf, error) {
	return m.generate(func() (*core.Pdf, error) {
		return m.inner.Generate()
	})
}

// GenerateCtx decorates the GenerateCtx method of paper instance.
func (m *Metrics) GenerateCtx(ctx context.Context) (*core.Pdf, error) {
	return m.generate(func() (*core.Pdf, error) {
		return m.inner.GenerateCtx(ctx)
	})
}

// AddPages decorates the AddPages method of paper instance.
func (m *Metrics) AddPages(pages ...core.Page) {
	timeSpent := time.GetTimeSpent(func() {
		m.inner.AddPages(pages...)
	})

	m.addPageTime = append(m.addPageTime, timeSpent)
}

// AddRows decorates the AddRows method of paper instance.
func (m *Metrics) AddRows(rows ...core.Row) {
	timeSpent := time.GetTimeSpent(func() {
		m.inner.AddRows(rows...)
	})

	m.addRowsTime = append(m.addRowsTime, timeSpent)
}

// AddHTML decorates the AddHTML method of paper instance.
func (m *Metrics) AddHTML(htmlStr string) error {
	return m.addHTML(func() error {
		return m.inner.AddHTML(htmlStr)
	})
}

// AddHTMLCtx decorates the AddHTMLCtx method of paper instance.
func (m *Metrics) AddHTMLCtx(ctx context.Context, htmlStr string) error {
	return m.addHTML(func() error {
		return m.inner.AddHTMLCtx(ctx, htmlStr)
	})
}

// AddRow decorates the AddRow method of paper instance.
func (m *Metrics) AddRow(rowHeight float64, cols ...core.Col) core.Row {
	var r core.Row
	timeSpent := time.GetTimeSpent(func() {
		r = m.inner.AddRow(rowHeight, cols...)
	})

	m.addRowTime = append(m.addRowTime, timeSpent)
	return r
}

// AddAutoRow decorates the AddAutoRow method of paper instance.
func (m *Metrics) AddAutoRow(cols ...core.Col) core.Row {
	var r core.Row
	timeSpent := time.GetTimeSpent(func() {
		r = m.inner.AddAutoRow(cols...)
	})

	m.addAutoRowTime = append(m.addAutoRowTime, timeSpent)
	return r
}

// RegisterHeader decorates the RegisterHeader method of paper instance.
func (m *Metrics) RegisterHeader(rows ...core.Row) error {
	var err error
	timeSpent := time.GetTimeSpent(func() {
		err = m.inner.RegisterHeader(rows...)
	})
	m.headerTime = timeSpent
	return err
}

// RegisterFooter decorates the RegisterFooter method of paper instance.
func (m *Metrics) RegisterFooter(rows ...core.Row) error {
	var err error
	timeSpent := time.GetTimeSpent(func() {
		err = m.inner.RegisterFooter(rows...)
	})
	m.footerTime = timeSpent
	return err
}

// GetStructure decorates the GetStructure method of paper instance.
func (m *Metrics) GetStructure() *node.Node[core.Structure] {
	var tree *node.Node[core.Structure]

	timeSpent := time.GetTimeSpent(func() {
		tree = m.inner.GetStructure()
	})
	m.structureTime = timeSpent

	return tree
}

func (m *Metrics) addHTML(innerAddHTML func() error) error {
	var err error
	timeSpent := time.GetTimeSpent(func() {
		err = innerAddHTML()
	})
	m.addRowsTime = append(m.addRowsTime, timeSpent)
	return err
}

func (m *Metrics) generate(innerGenerate func() (*core.Pdf, error)) (*core.Pdf, error) {
	var document *core.Pdf
	var err error

	timeSpent := time.GetTimeSpent(func() {
		document, err = innerGenerate()
	})
	m.generateTime = timeSpent

	if err != nil {
		return nil, err
	}

	bytes := document.GetBytes()

	report := m.buildMetrics(len(bytes)).Normalize()
	if innerReport := document.GetReport(); innerReport != nil {
		report.RenderIssues = append(report.RenderIssues, innerReport.RenderIssues...)
	}

	return core.NewPDF(bytes, report), nil
}

func (m *Metrics) buildMetrics(bytesSize int) *metrics.Report {
	var timeMetrics []metrics.TimeMetric

	if m.structureTime != nil {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "get_tree_structure",
			Times: []*metrics.Time{m.structureTime},
			Avg:   m.structureTime,
		})
	}

	if m.generateTime != nil {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "generate",
			Times: []*metrics.Time{m.generateTime},
			Avg:   m.generateTime,
		})
	}

	if m.headerTime != nil {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "header",
			Times: []*metrics.Time{m.headerTime},
			Avg:   m.headerTime,
		})
	}

	if m.footerTime != nil {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "footer",
			Times: []*metrics.Time{m.footerTime},
			Avg:   m.footerTime,
		})
	}

	if len(m.addPageTime) > 0 {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "add_page",
			Times: m.addPageTime,
			Avg:   m.getAVG(m.addPageTime),
		})
	}

	if len(m.addRowTime) > 0 {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "add_row",
			Times: m.addRowTime,
			Avg:   m.getAVG(m.addRowTime),
		})
	}

	if len(m.addRowsTime) > 0 {
		timeMetrics = append(timeMetrics, metrics.TimeMetric{
			Key:   "add_rows",
			Times: m.addRowsTime,
			Avg:   m.getAVG(m.addRowsTime),
		})
	}

	return &metrics.Report{
		TimeMetrics: timeMetrics,
		SizeMetric: metrics.SizeMetric{
			Key: "file_size",
			Size: metrics.Size{
				Value: float64(bytesSize),
				Scale: metrics.Byte,
			},
		},
	}
}

func (m *Metrics) getAVG(times []*metrics.Time) *metrics.Time {
	var sum float64
	for _, time := range times {
		sum += time.Value
	}

	return &metrics.Time{
		Value: sum / float64(len(times)),
		Scale: times[0].Scale,
	}
}
