# Releasing

Paper has one root module and two nested modules: `examples` and `docs`.
(`pkg/test` is a regular package of the root module — it is not tagged
separately.)

Work through this checklist in order for every release. `vX.Y.Z` is the new
version.

## 1. Pre-tag verification

- [ ] `make dod` passes locally (build, test, fmt, lint)
- [ ] `go run golang.org/x/tools/cmd/deadcode@latest -test ./...` reports nothing
- [ ] Benchmarks compared against the previous release
      (`go test -run='^$' -bench=. -count=6 . | tee new.txt` then
      `benchstat old.txt new.txt`) — no unexplained regressions

## 2. Tag the root module

- [ ] ```bash
      git tag vX.Y.Z
      git push origin vX.Y.Z
      ```
- [ ] Verify the root module zip stays small and excludes nested module content:
      ```bash
      go mod download -x github.com/avdoseferovic/paper@vX.Y.Z
      ls -lh "$(go env GOMODCACHE)/cache/download/github.com/avdoseferovic/paper/@v/vX.Y.Z.zip"
      ```

## 3. Re-pin and verify nested modules

The committed `go.work` makes local and CI builds resolve the root module from
source, which hides stale pins — the `GOWORK=off` builds below are the real
standalone check.

- [ ] Update `require github.com/avdoseferovic/paper vX.Y.Z` in `examples/go.mod`
      and `docs/go.mod`, then run `go mod tidy` in each
- [ ] ```bash
      cd examples && GOWORK=off go build ./... && cd ..
      cd docs && GOWORK=off go build ./assets/examples/... && cd ..
      ```
- [ ] Commit and push the pin bumps

## 4. Tag nested modules

- [ ] ```bash
      git tag examples/vX.Y.Z docs/vX.Y.Z
      git push origin examples/vX.Y.Z docs/vX.Y.Z
      ```

## 5. Publish

- [ ] Create the GitHub release for the `vX.Y.Z` tag with auto-generated notes:
      ```bash
      gh release create vX.Y.Z --generate-notes
      ```
