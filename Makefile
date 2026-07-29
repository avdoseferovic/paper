GO_FILES = $(shell find . '(' -path '*/.*' -o -path './vendor' ')' -prune -o -name '*.go' -print | cut -b3-)
GO_PATHS =  $(shell go list -f '{{ .Dir }}' ./... | grep -E -v 'docs|cmd|mocks')
EXAMPLES_PATHS = $(shell cd examples && go list -f '{{ .Dir }}' ./...)
DOCS_PATHS = $(shell cd docs && go list -f '{{ .Dir }}' ./...)
# Dev tools are pinned in tools/go.mod via Go 1.24 tool directives, so `make
# fmt` and CI cannot disagree because someone's goimports or gofumpt drifted.
#
# tools/ is deliberately NOT in go.work: workspace resolution applies MVS
# across every listed module, so a tool's dependency could quietly raise the
# version the library itself compiles against. GOWORK=off keeps the two apart.
#
# The tools are installed to ./.tools rather than invoked with `go -C tools
# tool`, because that form would run them with tools/ as the working directory
# and every path handed to them here is relative to the repo root.
TOOLBIN = $(CURDIR)/.tools

.PHONY: tools
tools:
	@GOWORK=off GOBIN=$(TOOLBIN) go -C $(CURDIR)/tools install tool

GOIMPORTS ?= $(TOOLBIN)/goimports
GOFUMPT ?= $(TOOLBIN)/gofumpt
GODOC ?= $(TOOLBIN)/godoc
DEADCODE ?= $(TOOLBIN)/deadcode
GOVULNCHECK ?= $(TOOLBIN)/govulncheck
MOCKERY ?= $(TOOLBIN)/mockery
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

.PHONY: fmt
fmt: tools
	gofmt -s -w ${GO_FILES}
	$(GOFUMPT) -l -w ${GO_FILES}
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

.PHONY: docs
docs:
	docsify serve docs/

.PHONY: godoc
godoc: tools
	$(GODOC) -http=127.0.0.1:6060

# govulncheck reports against the toolchain each module selects, so it runs per
# module rather than once across the workspace.
.PHONY: vuln
vuln: tools
	$(GOVULNCHECK) ./...
	cd examples && $(GOVULNCHECK) ./...
	cd docs && $(GOVULNCHECK) ./...

.PHONY: deadcode
deadcode: tools
	$(DEADCODE) -test ./...

.PHONY: mocks
mocks: tools
	find internal/mocks -type f -name '*.go' -delete
	$(MOCKERY)
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
	find site -name '*.go' -delete
	mkdir -p site/playground
	cp examples/cmd/wasm/web/index.html examples/cmd/wasm/web/paper.wasm examples/cmd/wasm/web/wasm_exec.js site/playground/
	@echo "Serving site at http://localhost:8080/  (playground: http://localhost:8080/playground/)"
	cd site && python3 -m http.server 8080

.PHONY: examples
examples:
	go run docs/assets/examples/addpage/main.go
	go run docs/assets/examples/autorow/main.go
	go run docs/assets/examples/background/main.go
	go run docs/assets/examples/barcodegrid/main.go
	go run docs/assets/examples/billing/main.go
	go run docs/assets/examples/bookmark/main.go
	cd examples && go run ./cmd/paper-showcase ../docs/assets/pdf/showcase.pdf
	go run docs/assets/examples/cellstyle/main.go
	go run docs/assets/examples/checkbox/main.go
	go run docs/assets/examples/compression/main.go
	go run docs/assets/examples/customdimensions/main.go
	go run docs/assets/examples/customfont/main.go
	go run docs/assets/examples/custompage/main.go
	go run docs/assets/examples/datamatrixgrid/main.go
	go run docs/assets/examples/disablepagebreak/main.go
	go run docs/assets/examples/footer/main.go
	go run docs/assets/examples/header/main.go
	go run docs/assets/examples/imagegrid/main.go
	go run docs/assets/examples/line/main.go
	go run docs/assets/examples/list/main.go
	go run docs/assets/examples/lowmemory/main.go
	go run docs/assets/examples/margins/main.go
	go run docs/assets/examples/maxgridsum/main.go
	go run docs/assets/examples/mergepdf/main.go
	go run docs/assets/examples/metadatas/main.go
	go run docs/assets/examples/orientation/main.go
	go run docs/assets/examples/pagenumber/main.go
	go run docs/assets/examples/parallelism/main.go
	go run docs/assets/examples/protection/main.go
	go run docs/assets/examples/qrgrid/main.go
	go run docs/assets/examples/signaturegrid/main.go
	go run docs/assets/examples/simplest/main.go
	go run docs/assets/examples/textgrid/main.go
	go run docs/assets/examples/watermark/main.go
	go test docs/assets/examples/unittests/main_test.go
