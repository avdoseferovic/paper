# Contributing

## Versioning

Paper follows Semantic Versioning. Before `v1.0.0`, minor releases may contain breaking API changes while the public surface is being finalized; those changes must be called out in `CHANGELOG.md`. After `v1.0.0`, breaking public API changes require a major version bump, and patch releases are limited to compatible bug fixes.

## Changelog

Pull requests that change the public API, package behavior, or root Go entry points must update `CHANGELOG.md`. Use the `override-changelog` label only for mechanical changes, tests, docs-only edits, or CI-only updates where a changelog entry would add noise.

## Release Protocol

Paper has one root module and three nested modules: `pkg/test`, `examples`, and `docs`. For the first release, tag the root module first while nested modules still use their local `replace` directives:

```bash
git tag v0.1.0
git push origin v0.1.0
```

After the root tag exists, update each nested `go.mod` to require `github.com/avdoseferovic/paper v0.1.0` and remove its local `replace`. Run `go mod tidy` in `pkg/test`, `examples`, and `docs`, then run `make dod`. Commit and push those nested-module pinning changes before creating nested module tags.

Tag nested modules only after that verification passes:

```bash
git tag pkg/test/v0.1.0 examples/v0.1.0 docs/v0.1.0
git push origin pkg/test/v0.1.0 examples/v0.1.0 docs/v0.1.0
```

Before pushing any release tag, verify the root module zip remains small and excludes nested module content:

```bash
go mod download -x github.com/avdoseferovic/paper@v0.1.0
ls -lh "$(go env GOMODCACHE)/cache/download/github.com/avdoseferovic/paper/@v/v0.1.0.zip"
```
