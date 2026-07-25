# Parallelism

`WithParallelPagesMode` renders page groups on several goroutines instead of one, then splices the rendered pages into a single document that is serialized once.

## Generation modes comparison

| Mode | Method | Memory | Speed |
|------|--------|--------|-------|
| Default (sequential) | `config.NewBuilder()` | Medium | Baseline |
| Low memory | `WithSequentialLowMemoryMode(n)` | Low | Slower |
| Parallel pages | `WithParallelPagesMode(workers)` | Higher | Fastest on large documents |

## Measured speedup

Text-heavy documents, Apple M1 Pro (10 cores), compared against sequential generation on the same machine:

| Document | Sequential | Parallel pages (4 workers) | Speedup |
|----------|-----------|----------------------------|---------|
| 7 pages | 317µs | 422µs | 0.75x (slower) |
| 35 pages | 1.45ms | 1.30ms | 1.11x |
| 173 pages | 6.63ms | 3.92ms | 1.69x |
| 140 pages, 150 bookmarks | 6.01ms | 3.87ms | 1.55x |

Parallel rendering only pays off once there is enough work to spread. Below roughly 30 pages the coordination costs more than it saves, so the default sequential mode is the better choice for small documents.

The bookmarked document is the last row: it used to abandon the parallel render and start over sequentially, costing about 15% over generating sequentially outright. It now splices like any other document.

The remaining limit is the garbage collector rather than the core count: allocation is process-wide, so workers contend on it. Relaxing GC pressure in the host application widens the gap substantially. paper deliberately does not change the GC settings of the process it runs in.

## Usage notes

- Rendering happens per page *group*; `workers` sets how many groups are rendered concurrently.
- Memory scales with the worker count, since each worker holds its own pages while rendering.
- The generation modes are mutually exclusive; the last one called wins.
- Output is deterministic. With `WithDeterministic(true)`, parallel output is byte-for-byte identical to sequential output at any worker count, because fonts, images, gradients and blend modes are named by a content hash of their definition rather than by registration order.
- `doc.GetReport().GenerationMode` reports the mode a document was actually generated with, which is not always the one you configured. Use it to check that a document you expected to render in parallel really did.

### What still falls back to sequential

Everything a page can draw is spliceable: text, images, external hyperlinks, internal links, bookmarks, gradients and blend modes (which is how watermarks draw). Page content is named by a content hash, and the state that records absolute page numbers — links and bookmarks — is rewritten as pages are spliced into place.

What is left is *document-catalog* features, which are written once for the whole document rather than per page: forms, PDF/A conformance, tagged PDF, attachments, named destinations, page labels, viewer preferences, an explicit document language or file ID, configured annotations and page geometries. A document using any of these, or document protection, is generated sequentially from the start — the configured mode is overridden before rendering begins, so nothing is rendered twice.

Splicing is also re-checked while rendering, as defence in depth. If a future feature turns out not to be spliceable, the worker that draws it aborts the pool immediately and `WithParallelPagesMode` transparently re-renders the document sequentially. You get sequential performance in that case, not an error and not a corrupt document.

## GoDoc
* [builder : WithParallelPagesMode](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithParallelPagesMode)

## Code Example
[filename](../assets/examples/parallelism/main.go  ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/parallelism.pdf
```

## Time Execution
[filename](../assets/text/parallelism.txt  ':include :type=code')

## Time Execution (100 pages)
[filename](../assets/text/parallel.txt ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/parallelism.json  ':include :type=code')
