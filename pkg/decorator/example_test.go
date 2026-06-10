package decorator_test

import (
	"fmt"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/decorator"
)

// ExampleNewMetrics demonstrates how to wrap a paper instance with the metrics
// decorator.
func ExampleNewMetrics() {
	// optional
	b := config.NewBuilder()
	cfg := b.Build()

	mrt := paper.New(cfg)          // cfg is an optional
	m := decorator.NewMetrics(mrt) // decorator of paper

	// Do things and generate
	_, _ = m.Generate()
	fmt.Println("generated")

	// Output: generated
}
