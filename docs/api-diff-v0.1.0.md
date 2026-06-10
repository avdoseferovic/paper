# API Diff for v0.1.0 Prep

This report was generated against `HEAD` and the current working tree with:

```bash
go run golang.org/x/exp/cmd/apidiff@latest -m <head.export> <current.export>
```

`apidiff` ignored internal packages. The package removals below are intentional
pre-v1 breaks caused by splitting demos, docs examples, test helpers, and mocks
out of the root module.

## Incompatible Changes

- Root demo packages under `cmd/` were removed from the root module.
- Docs example packages under `docs/assets/examples/` were removed from the root module.
- `github.com/avdoseferovic/paper/pkg/test` was removed from the root module and moved to a nested module.
- `github.com/avdoseferovic/paper/mocks` was removed; generated mocks now live under `internal/mocks`.
- `MetricsDecorator` and `NewMetricsDecorator` were removed from the root package; use `github.com/avdoseferovic/paper/pkg/decorator.NewMetrics`.
- `New` changed from returning `core.Paper` to returning `*paper.Paper`.
- The previous misspelled current-page fit method was removed from `*Paper` and `core.Paper`.
- `(*Paper).FitInCurrentPage` and `core.Paper.FitInCurrentPage` were added.
- `(*Paper).Generate` and `core.Paper.Generate` changed from returning `core.Document` to returning `*core.Pdf`.
- `core.Paper.GenerateCtx` and `core.Paper.AddHTMLCtx` were added.
- `core.NewPDF` changed from returning `core.Document` to returning `*core.Pdf`.
- `paper.FromHTML` and `paper.FromHTMLReader` changed from returning `core.Document` to returning `*core.Pdf`.
- `pkg/config.documentBuilder.WithHTMLLimits` and `pkg/config.documentBuilder.WithUnsafeNoHTMLLimits` were added to the document builder interface.
- `pkg/config.documentBuilder.WithProtectionAlgorithm` was added to the document builder interface.

## Compatible Changes

- `core.Document.Write` and `(*core.Pdf).Write` were added.
- `core/entity.Config.HTMLLimits` and `core/entity.HTMLLimits` were added.
- `core/entity.Protection.Algorithm` and `pkg/consts/protection.Encryption` were added.
- `pkg/html.DefaultLimits`, `pkg/html.Limits`, `pkg/html.WithLimits`, and `pkg/html.WithUnsafeNoLimits` were added.
- `pkg/html` exported resource-limit sentinels: `ErrImageTooLarge`, `ErrDOMTooDeep`, `ErrDOMTooLarge`, `ErrSVGTooLarge`, and `ErrStyleRulesTooLarge`.
- `pkg/html.FromStringCtx`, `pkg/html.FromReaderCtx`, and `pkg/html/translate.TranslateCtx` were added.
- `pkg/html/dom.(*Document).ValidateLimits` and `WalkWithLimits` were added.
- `pkg/components/html.Limits`, `WithLimits`, and `WithUnsafeNoLimits` were added.
- `paper.FromHTMLCtx`, `paper.FromHTMLReaderCtx`, `(*Paper).GenerateCtx`, and `(*Paper).AddHTMLCtx` were added.
- `pkg/config.(*CfgBuilder).WithHTMLLimits` and `WithUnsafeNoHTMLLimits` were added.
- `pkg/config.(*CfgBuilder).WithProtectionAlgorithm` was added.
- `github.com/avdoseferovic/paper/pkg/decorator` was added.
