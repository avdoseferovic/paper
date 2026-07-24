// Package dependency holds the guard test that keeps the root module free of
// third-party imports. It carries no implementation; this file exists so the
// directory is a buildable package rather than a test-only one, which `go build`
// rejects when packages are listed explicitly (as the Makefile does).
package dependency
