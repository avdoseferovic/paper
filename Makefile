GO_FILES = $(shell find . '(' -path '*/.*' -o -path './vendor' ')' -prune -o -name '*.go' -print | cut -b3-)
GO_PATHS =  $(shell go list -f '{{ .Dir }}' ./... | grep -E -v 'docs|cmd|mocks')
EXAMPLES_PATHS = $(shell cd examples && go list -f '{{ .Dir }}' ./...)
DOCS_PATHS = $(shell cd docs && go list -f '{{ .Dir }}' ./...)
GOIMPORTS ?= $(shell if command -v goimports >/dev/null 2>&1; then command -v goimports; else echo "go run golang.org/x/tools/cmd/goimports@latest"; fi)

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
fmt:
	gofmt -s -w ${GO_FILES}
	gofumpt -l -w ${GO_FILES}
	$(GOIMPORTS) -w ${GO_PATHS} ${EXAMPLES_PATHS} ${DOCS_PATHS}

.PHONY: lint
lint:
	golangci-lint run --config=.golangci.yml ./...
	cd examples && golangci-lint run --config=../.golangci.yml --disable=gomoddirectives ./...
	cd docs && golangci-lint run --config=../.golangci.yml --disable=gomoddirectives ./assets/examples/...
	make mock-lint

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
