package core

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/avdoseferovic/paper/internal/time"
	"github.com/avdoseferovic/paper/pkg/merge"
	"github.com/avdoseferovic/paper/pkg/metrics"
)

var (
	ErrCannotMergeBytes = errors.New("cannot merge bytes")
	ErrCannotWriteFile  = errors.New("cannot write file")
)

type Pdf struct {
	bytes  []byte
	report *metrics.Report
}

// NewPDF creates a concrete PDF document.
func NewPDF(bytes []byte, report *metrics.Report) *Pdf {
	return &Pdf{
		bytes:  bytes,
		report: report,
	}
}

// GetBytes returns the PDF bytes.
func (p *Pdf) GetBytes() []byte {
	return p.bytes
}

// GetBase64 returns the PDF bytes in base64.
func (p *Pdf) GetBase64() string {
	return base64.StdEncoding.EncodeToString(p.bytes)
}

// Write streams the document to w without an intermediate copy beyond the already-generated buffer.
func (p *Pdf) Write(w io.Writer) (int64, error) {
	written, err := bytes.NewReader(p.bytes).WriteTo(w)
	if err != nil {
		return written, fmt.Errorf("cannot write document: %w", err)
	}

	return written, nil
}

// GetReport returns the metrics.Report.
func (p *Pdf) GetReport() *metrics.Report {
	return p.report
}

// Save saves the PDF in a file.
func (p *Pdf) Save(file string) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCannotWriteFile, err)
	}

	_, writeErr := p.Write(f)
	closeErr := f.Close()
	err = errors.Join(writeErr, closeErr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCannotWriteFile, err)
	}

	return nil
}

// Merge merges the PDF with another PDF.
func (p *Pdf) Merge(bytes []byte) error {
	var mergedBytes []byte
	var err error

	timeSpent := time.GetTimeSpent(func() {
		mergedBytes, err = merge.Bytes(p.bytes, bytes)
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCannotMergeBytes, err)
	}
	p.bytes = mergedBytes
	if p.report != nil {
		p.appendMetric(timeSpent)
	}

	return nil
}

func (p *Pdf) appendMetric(timeSpent *metrics.Time) {
	timeMetric := metrics.TimeMetric{
		Key:   "merge_pdf",
		Times: []*metrics.Time{timeSpent},
		Avg:   timeSpent,
	}
	timeMetric.Normalize()
	p.report.TimeMetrics = append(p.report.TimeMetrics, timeMetric)

	p.report.SizeMetric = metrics.SizeMetric{
		Key: "file_size",
		Size: metrics.Size{
			Value: float64(len(p.bytes)),
			Scale: metrics.Byte,
		},
	}
	p.report.Normalize()
}
