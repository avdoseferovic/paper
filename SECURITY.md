# Security Policy

## Supported versions

Paper is pre-1.0. Security fixes are applied to the latest released minor
version only; there are no long-term support branches.

## Reporting a vulnerability

Please **do not** open a public issue for a security problem.

Report it through GitHub's private vulnerability reporting:
[**Report a vulnerability**](https://github.com/avdoseferovic/paper/security/advisories/new).

If that is unavailable to you, open a regular issue containing only the words
"security report — please contact me" and no technical detail, and a maintainer
will arrange a private channel.

Please include, as far as you can:

- the affected version or commit,
- a minimal input that reproduces the problem (PDF, HTML, SVG, or font file),
- what an attacker gains, and
- any suggested fix.

You can expect an acknowledgement within 7 days and a status update at least
every 14 days until the report is resolved.

## Scope

Paper parses untrusted input: HTML, CSS, SVG, images, fonts, and existing PDF
files. The following are in scope:

- memory-safety failures reachable from parsed input: panics, unbounded
  allocation, unbounded recursion, or non-termination,
- escaping the asset sandbox — reading a file outside the configured base
  directory via `WithImageBaseDir` / `WithStylesheetBaseDir`,
- outbound requests to hosts a caller did not authorise, bypassing the
  `WithRemoteAssets` opt-in and its `URLPolicy`,
- defects in PDF encryption or signing that weaken the protection a caller
  asked for.

The following are **not** vulnerabilities in Paper:

- MD5 and RC4 in the PDF standard security handler, and SHA-1 in PAdES VRI
  keys. These algorithms are named by ISO 32000-1 §7.6.3 and ETSI EN 319 142;
  they are what the formats specify, not a choice this library makes.
- Reading a path a caller passed in directly (`LoadPNG`, `Pdf.Save`,
  `AddUTF8Font`, `tmpl.Render`). Opening the caller's path is the documented
  purpose of those functions.
- Resource exhaustion from input a caller supplies to itself, where a documented
  limit exists and was not configured (see `WithHTMLLimits`).

## Hardening this project applies

- `govulncheck` runs in CI on every push and on a weekly schedule.
- Fuzz targets cover the untrusted-input parsers and run in CI.
- Untrusted asset paths are resolved through `os.Root`, which refuses absolute
  paths and any path escaping the configured base directory.
- Workflow actions are pinned to commit SHAs.
