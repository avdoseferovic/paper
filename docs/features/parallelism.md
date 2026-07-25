# Parallelism

paper can render a document's pages on several goroutines instead of one. Two modes do this, and they differ in how the concurrently rendered pages are joined back together.

`WithParallelPagesMode` is the one to reach for. It renders page groups in parallel, then splices the rendered pages into a single document that is serialized once.

`WithConcurrentMode` is the older strategy: every chunk is serialized to a complete PDF and those byte streams are merged. Each chunk therefore pays for a full document — font programs, resource dictionary, cross-reference table — and the merge re-parses all of it, so its overhead grows with the worker count. It is kept for compatibility.

## Generation modes comparison

| Mode | Method | Memory | Speed |
|------|--------|--------|-------|
| Default (sequential) | `config.NewBuilder()` | Medium | Baseline |
| Low memory | `WithSequentialLowMemoryMode(n)` | Low | Slower |
| Parallel pages | `WithParallelPagesMode(workers)` | Higher | Fastest on large documents |
| Concurrent (legacy) | `WithConcurrentMode(workers)` | Highest | Slower than sequential on small documents |

## Measured speedup

Text-heavy documents, Apple M1 Pro (10 cores), compared against sequential generation on the same machine:

| Document | Sequential | Parallel pages (4 workers) | Speedup |
|----------|-----------|----------------------------|---------|
| 7 pages | 317µs | 422µs | 0.75x (slower) |
| 35 pages | 1.45ms | 1.30ms | 1.11x |
| 173 pages | 6.63ms | 3.92ms | 1.69x |

Parallel rendering only pays off once there is enough work to spread. Below roughly 30 pages the coordination costs more than it saves, so the default sequential mode is the better choice for small documents.

The remaining limit is the garbage collector rather than the core count: allocation is process-wide, so workers contend on it. Relaxing GC pressure in the host application widens the gap substantially. paper deliberately does not change the GC settings of the process it runs in.

## Usage notes

- Both parallel modes render page *groups*; `workers` sets how many groups are rendered concurrently.
- Memory scales with the worker count, since each worker holds its own pages while rendering.
- The generation modes are mutually exclusive; the last one called wins.
- Output is deterministic. With `WithDeterministic(true)`, parallel output is byte-for-byte identical to sequential output at any worker count, because fonts and images are named by a content hash of their definition rather than by registration order.

### When parallel pages falls back to sequential

Some features cannot be spliced, either because their PDF names are allocated sequentially (gradients, blend modes) or because they record absolute page indices (internal links, outlines, annotations, page geometries, form fields). When a document uses one of them, `WithParallelPagesMode` detects it after rendering and transparently re-renders the document sequentially, so the output stays correct. You get sequential performance in that case, not an error.

Documents using protection or document-catalog features (forms, PDF/A, tagged PDF, attachments) always use sequential generation, as before.

## GoDoc
* [builder : WithParallelPagesMode](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithParallelPagesMode)
* [builder : WithConcurrentMode](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithConcurrentMode)

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
