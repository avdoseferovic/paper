#!/usr/bin/env bash
#
# Installs the development toolchain into $(go env GOPATH)/bin.
# Versions are pinned to match CI (.github/workflows) and the Makefile.
# No sudo required — make sure $(go env GOPATH)/bin is on your PATH.

set -euo pipefail

GOLANGCI_LINT_VERSION="v2.11.3" # must match .github/workflows/golangci-lint.yml
MOCKERY_VERSION="v2.53.6"       # must match the Makefile mocks target

GOBIN="$(go env GOPATH)/bin"

go install golang.org/x/tools/cmd/goimports@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/godoc@latest
go install "github.com/vektra/mockery/v2@${MOCKERY_VERSION}"

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |
	sh -s -- -b "${GOBIN}" "${GOLANGCI_LINT_VERSION}"

echo ""
echo "Tools installed to ${GOBIN}."
case ":${PATH}:" in
*":${GOBIN}:"*) ;;
*)
	echo "WARNING: ${GOBIN} is not on your PATH. Add it, e.g.:"
	echo "  export PATH=\"\$PATH:${GOBIN}\""
	;;
esac

echo ""
echo "Optional (docs site): npm i -g docsify-cli"
