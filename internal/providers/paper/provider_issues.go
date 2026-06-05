package paper

import "github.com/avdoseferovic/paper/pkg/metrics"

// RenderIssues returns the render fallback issues recorded by the provider.
func (g *provider) RenderIssues() []metrics.RenderIssue {
	return append([]metrics.RenderIssue(nil), g.issues...)
}

func (g *provider) recordRenderIssue(operation, message string, err error) {
	issue := metrics.RenderIssue{
		Operation: operation,
		Message:   message,
	}
	if err != nil {
		issue.Error = err.Error()
	}
	g.issues = append(g.issues, issue)
}
