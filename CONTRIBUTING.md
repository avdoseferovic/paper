# Contributing

For environment setup, the module/workspace layout, and the everyday
build/test/mocks workflow, see [DEVELOPMENT.md](DEVELOPMENT.md).

## Versioning

Paper follows Semantic Versioning. Before `v1.0.0`, minor releases may contain breaking API changes while the public surface is being finalized. After `v1.0.0`, breaking public API changes require a major version bump, and patch releases are limited to compatible bug fixes.

## Release Protocol

Releases follow the step-by-step checklist in [RELEASING.md](RELEASING.md):
root module is tagged first, nested module pins (`examples`, `docs`) are bumped
and verified with `GOWORK=off` builds, then the nested modules are tagged.
