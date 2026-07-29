# Releasing

Paper has one root module and two nested modules: `examples` and `docs`.
(`pkg/test` is a regular package of the root module — it is not tagged
separately.)

Work through this checklist in order for every release. `vX.Y.Z` is the new
version.

## 1. Pre-tag verification

- [ ] `make dod` passes locally (build, test, fmt, lint)
- [ ] `make deadcode` reports nothing
- [ ] `make vuln` reports no vulnerabilities in any of the three modules
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
source, which hides stale pins. `GOWORK=off` alone does **not** expose them:
`examples/go.mod` and `docs/go.mod` also carry
`replace github.com/avdoseferovic/paper => ..`, and a `replace` survives
`GOWORK=off`. The replace has to be dropped for the duration of the check,
which is what the script below does.

Keep the `replace` the rest of the time. Without it the nested modules resolve
the *previous* tag from the proxy, and that tag's dependency set comes with
it — after v0.2.1 that meant `cascadia`, `boombuler/barcode`, `oksvg`,
`rasterx` and `tdewolff/parse` reappearing as indirect requirements of `docs`,
all of them libraries the root module has since replaced with internal
packages. The dependency guard in `internal/dependency` does not catch this,
because it runs under `go.work` and therefore sees local source.

- [ ] Update `require github.com/avdoseferovic/paper vX.Y.Z` in `examples/go.mod`
      and `docs/go.mod`
- [ ] Verify each nested module against the published tag, with the replace
      temporarily dropped so the pin is what actually resolves:
      ```bash
      set -e
      for m in examples docs; do
        pkgs=./...
        [ "$m" = docs ] && pkgs=./assets/examples/...
        (
          cd "$m"
          go mod edit -dropreplace=github.com/avdoseferovic/paper
          trap 'go mod edit -replace=github.com/avdoseferovic/paper=..' EXIT
          GOWORK=off go mod tidy
          GOWORK=off go build "$pkgs"
        )
      done
      ```
      A failure here means the pin does not match what was tagged — fix the tag
      or the pin, do not paper over it by restoring the replace early.
- [ ] Re-tidy against local source now that the replace is back, so the
      committed graph is the workspace one:
      ```bash
      cd examples && GOWORK=off go mod tidy && cd ..
      cd docs && GOWORK=off go mod tidy && cd ..
      ```
- [ ] Confirm `git diff examples/go.mod docs/go.mod` shows only the pin bump —
      any new third-party indirect requirement means the tag and the working
      tree disagree
- [ ] Re-check that the workspace build still passes (`make build`)
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
