package core

import "github.com/avdoseferovic/paper/v2/pkg/metrics"

// RenderIssueProvider exposes best-effort render fallbacks recorded by a provider.
type RenderIssueProvider interface {
	RenderIssues() []metrics.RenderIssue
}
