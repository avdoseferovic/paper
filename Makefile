GO_FILES = $(shell find . '(' -path '*/.*' -o -path './vendor' ')' -prune -o -name '*.go' -print | cut -b3-)
GO_PATHS =  $(shell go list -f '{{ .Dir }}' ./... | grep -E -v 'docs|cmd|mocks')
EXAMPLES_PATHS = $(shell cd examples && go list -f '{{ .Dir }}' ./...)
DOCS_PATHS = $(shell cd docs && go list -f '{{ .Dir }}' ./...)
GOIMPORTS ?= $(shell if command -v goimports >/dev/null 2>&1; then command -v goimports; else echo "go run golang.org/x/tools/cmd/goimports@latest"; fi)
GOLANGCI_LINT ?= $(shell if command -v golangci-lint >/dev/null 2>&1; then command -v golangci-lint; else echo "go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1"; fi)

.PHONY: dod
dod: build test fmt lint

.PHONY: build
build:
	go build $(GO_PATHS)
	cd examples && go build ./...

.PHONY: test
test:
	go test $(GO_PATHS)
	cd examples && go test ./...
	cd docs && go test ./assets/examples/...
	# The browser filesystem shim implements a Go syscall contract by hand, and a
	# break in it renders an error box into an otherwise valid PDF rather than
	# failing, so it gets its own tests.
	node --test docs/assets/js/paper-fs.test.mjs

.PHONY: fmt
fmt:
	gofmt -s -w ${GO_FILES}
	gofumpt -l -w ${GO_FILES}
	$(GOIMPORTS) -w ${GO_PATHS} ${EXAMPLES_PATHS} ${DOCS_PATHS}

.PHONY: lint
lint: go-lint mock-lint

# golangci-lint runs against each module in the workspace. The config lives in
# .golangci.yml at the repo root and is shared by all three.
.PHONY: go-lint
go-lint:
	$(GOLANGCI_LINT) run --config .golangci.yml ./...
	cd examples && $(GOLANGCI_LINT) run --config ../.golangci.yml ./...
	cd docs && $(GOLANGCI_LINT) run --config ../.golangci.yml ./...

.PHONY: mock-lint
mock-lint:
	bash shell/mock-check.sh

.PHONY: install
install:
	bash shell/install.sh

# Serve the docs locally. Depends on wasm because the feature pages generate
# their PDF previews in the browser from docs/assets/wasm/paper.wasm.
.PHONY: docs
docs: wasm
	mkdir -p docs/assets/wasm
	# -f: the toolchain's wasm_exec.js is read-only (0444), so the copy is too.
	cp -f examples/cmd/wasm/web/paper.wasm examples/cmd/wasm/web/wasm_exec.js docs/assets/wasm/
	docsify serve docs/

.PHONY: godoc
godoc:
	godoc -http=127.0.0.1:6060


.PHONY: mocks
mocks:
	find internal/mocks -type f -name '*.go' -delete
	go run github.com/vektra/mockery/v2@v2.53.6
	go run ./internal/cmd/mockfix internal/mocks
	make fmt

.PHONY: wasm
wasm:
	cd examples/cmd/wasm && ./build.sh

# Assemble the full Pages site locally (docs homepage at the root + the built
# playground under /playground/) and serve it, mirroring the deploy workflow.
.PHONY: site
site: wasm
	rm -rf site
	cp -r docs site
	rm -f site/go.mod site/go.sum
	rm -rf site/plans
	# Only test files go: docsify ':include's the example sources, so deleting
	# every *.go (as this used to) left the code samples on the site empty.
	find site -name '*_test.go' -delete
	mkdir -p site/assets/wasm site/playground
	cp -f examples/cmd/wasm/web/paper.wasm examples/cmd/wasm/web/wasm_exec.js site/assets/wasm/
	cp -f examples/cmd/wasm/web/index.html examples/cmd/wasm/web/paper.wasm examples/cmd/wasm/web/wasm_exec.js site/playground/
	@echo "Serving site at http://localhost:8080/  (playground: http://localhost:8080/playground/)"
	cd site && python3 -m http.server 8080

.PHONY: examples
examples:
	go run ./docs/assets/examples/addpage/cmd
	go run ./docs/assets/examples/autorow/cmd
	go run ./docs/assets/examples/background/cmd
	go run ./docs/assets/examples/barcodegrid/cmd
	go run ./docs/assets/examples/billing/cmd
	go run ./docs/assets/examples/bookmark/cmd
	cd examples && go run ./cmd/paper-showcase ../docs/assets/pdf/showcase.pdf
	go run ./docs/assets/examples/cellstyle/cmd
	go run ./docs/assets/examples/checkbox/cmd
	go run ./docs/assets/examples/compression/cmd
	go run ./docs/assets/examples/customdimensions/cmd
	go run ./docs/assets/examples/customfont/cmd
	go run ./docs/assets/examples/custompage/cmd
	go run ./docs/assets/examples/datamatrixgrid/cmd
	go run ./docs/assets/examples/disablepagebreak/cmd
	go run ./docs/assets/examples/footer/cmd
	go run ./docs/assets/examples/header/cmd
	go run ./docs/assets/examples/imagegrid/cmd
	go run ./docs/assets/examples/line/cmd
	go run ./docs/assets/examples/list/cmd
	go run ./docs/assets/examples/lowmemory/cmd
	go run ./docs/assets/examples/margins/cmd
	go run ./docs/assets/examples/maxgridsum/cmd
	go run ./docs/assets/examples/mergepdf/cmd
	go run ./docs/assets/examples/metadatas/cmd
	go run ./docs/assets/examples/orientation/cmd
	go run ./docs/assets/examples/pagenumber/cmd
	go run ./docs/assets/examples/parallelism/cmd
	go run ./docs/assets/examples/protection/cmd
	go run ./docs/assets/examples/qrgrid/cmd
	go run ./docs/assets/examples/signaturegrid/cmd
	go run ./docs/assets/examples/simplest/cmd
	go run ./docs/assets/examples/textgrid/cmd
	go run ./docs/assets/examples/watermark/cmd
	go test docs/assets/examples/unittests/main_test.go
