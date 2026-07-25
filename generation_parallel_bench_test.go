package paper_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
)

// BenchmarkGenerationModes compares the generation modes on the same document:
// sequential, low memory, and page-splicing parallel rendering at a couple of
// worker counts. Run with -benchtime 20x for stable numbers.
func BenchmarkGenerationModes(b *testing.B) {
	for _, rowCount := range []int{200, 1000, 5000} {
		rows := parallelTestRows(rowCount)

		b.Run(fmt.Sprintf("rows=%d/sequential", rowCount), func(b *testing.B) {
			benchmarkGeneration(b, config.NewBuilder().WithSequentialMode().Build(), rows)
		})

		b.Run(fmt.Sprintf("rows=%d/lowmemory-4", rowCount), func(b *testing.B) {
			benchmarkGeneration(b, config.NewBuilder().WithSequentialLowMemoryMode(4).Build(), rows)
		})

		for _, workers := range []int{4, 8} {
			b.Run(fmt.Sprintf("rows=%d/parallelpages-%d", rowCount, workers), func(b *testing.B) {
				benchmarkGeneration(b, config.NewBuilder().WithParallelPagesMode(workers).Build(), rows)
			})
		}
	}
}

func benchmarkGeneration(b *testing.B, cfg *entity.Config, rows []core.Row) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		m := paper.New(cfg)
		m.AddRows(rows...)
		doc, err := m.Generate(context.Background())
		if err != nil {
			b.Fatalf("generate: %v", err)
		}
		if len(doc.GetBytes()) == 0 {
			b.Fatal("generated empty PDF")
		}
	}
}
