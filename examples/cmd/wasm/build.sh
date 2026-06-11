#!/usr/bin/env bash
# Builds the Paper browser demo: compiles the wasm binary into web/ and copies
# the matching wasm_exec.js from the active Go toolchain.
set -euo pipefail

cd "$(dirname "$0")"

GOOS=js GOARCH=wasm go build -o web/paper.wasm .

wasm_exec="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [[ ! -f "$wasm_exec" ]]; then
	echo "error: $wasm_exec not found (expected Go >= 1.21 with lib/wasm/wasm_exec.js)" >&2
	exit 1
fi
cp "$wasm_exec" web/wasm_exec.js

echo "Built web/paper.wasm and copied web/wasm_exec.js"
echo "Serve it with:  (cd $(pwd)/web && python3 -m http.server 8080)"
