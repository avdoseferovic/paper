# Contributing

For environment setup, the module/workspace layout, and the everyday
build/test/mocks workflow, see [DEVELOPMENT.md](DEVELOPMENT.md).

## Versioning

Paper follows Semantic Versioning. Before `v1.0.0`, minor releases may contain breaking API changes while the public surface is being finalized; those changes must be called out in `CHANGELOG.md`. After `v1.0.0`, breaking public API changes require a major version bump, and patch releases are limited to compatible bug fixes.

## Changelog

Pull requests that change the public API, package behavior, or root Go entry points must update `CHANGELOG.md`. Use the `override-changelog` label only for mechanical changes, tests, docs-only edits, or CI-only updates where a changelog entry would add noise.

## Release Protocol

Releases follow the step-by-step checklist in [RELEASING.md](RELEASING.md):
root module is tagged first, nested module pins (`examples`, `docs`) are bumped
and verified with `GOWORK=off` builds, then the nested modules are tagged.
