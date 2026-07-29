#!/usr/bin/env bash
#
# Installs the development toolchain.
#
# Every tool is pinned by a Go 1.24 tool directive in tools/go.mod, so this
# script no longer names versions: `make tools` (which it delegates to) is the
# single source of truth, and Dependabot keeps tools/go.mod current.
#
# Tools land in ./.tools, which is gitignored and is where the Makefile looks
# for them. Nothing is written to $(go env GOPATH)/bin and no sudo is required.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

make -C "${REPO_ROOT}" tools

echo ""
echo "Tools installed to ${REPO_ROOT}/.tools:"
ls -1 "${REPO_ROOT}/.tools"

echo ""
echo "The Makefile invokes these by path, so they need not be on your PATH."
echo "To use them by hand: export PATH=\"${REPO_ROOT}/.tools:\$PATH\""

echo ""
echo "Optional (docs site): npm i -g docsify-cli"
