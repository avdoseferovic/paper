//go:build !(js && wasm)

// This stub exists so the cmd/wasm package builds and lints on non-wasm
// platforms. A main package whose only source is excluded by build constraints
// fails to build, which would break `go build ./...` on the host. The real
// entry point lives in main.go behind the `js && wasm` build tag.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr,
		"paper wasm: this command must be built with GOOS=js GOARCH=wasm; see examples/cmd/wasm/README.md")
	os.Exit(1)
}
