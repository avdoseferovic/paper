# internal/paperpdf ownership

`internal/paperpdf` is Paper's internalized PDF runtime backend. It is derived
from `github.com/phpdave11/gofpdf v1.4.3` so Paper can build without depending on
the external GoFPDF module.

Treat this directory as third-party-derived backend code:

- Preserve the upstream license and notice files.
- Avoid style-only rewrites, broad refactors, or lint churn in this directory.
- Keep Paper-owned adapter logic in `internal/providers/paper`.
- Keep behavior changes small, reviewed, and backed by PDF rendering tests.
- When upstream-derived source is refreshed, document the source version and any
  local patches in `NOTICE`.

This boundary is intentional. Most Paper maintenance should happen in public
packages, `internal/providers/paper`, or smaller support packages rather than in
the copied backend runtime.
