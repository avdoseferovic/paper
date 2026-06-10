# Releasing

Paper has one root module and two nested modules: `examples` and `docs`.
(`pkg/test` is a regular package of the root module — it is not tagged
separately.)

Work through this checklist in order for every release. `vX.Y.Z` is the new
version.

## 1. Pre-tag verification

- [ ] `CHANGELOG.md` updated: move `[Unreleased]` entries under `[vX.Y.Z] - YYYY-MM-DD`,
      with breaking changes called out explicitly
- [ ] `make dod` passes locally (build, test, fmt, lint)
- [ ] `go run golang.org/x/tools/cmd/deadcode@latest -test ./...` reports nothing
- [ ] Benchmarks compared against the previous release
      (`go test -run='^$' -bench=. -count=6 . | tee new.txt` then
      `benchstat old.txt new.txt`) — no unexplained regressions
- [ ] CI is green on the release commit

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

- [ ] GitHub release is created automatically by `.github/workflows/release.yml`
      on the `vX.Y.Z` tag — review the generated notes and paste in the
      CHANGELOG excerpt
