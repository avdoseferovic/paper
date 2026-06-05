GO_FILES = $(shell find . '(' -path '*/.*' -o -path './vendor' ')' -prune -o -name '*.go' -print | cut -b3-)
GO_PATHS =  $(shell go list -f '{{ .Dir }}' ./... | grep -E -v 'docs|cmd|mocks')
GO_EXAMPLES =  $(shell go list -f '{{ .Dir }}' ./docs/assets/examples/...)

.PHONY: dod
dod: build test fmt lint

.PHONY: build
build:
	go build $(GO_PATHS)

.PHONY: test
test:
	go test $(GO_PATHS)
	go test $(GO_EXAMPLES)

.PHONY: fmt
fmt:
	gofmt -s -w ${GO_FILES}
	gofumpt -l -w ${GO_FILES}
	goimports -w ${GO_PATHS}

.PHONY: lint
lint:
	golangci-lint run --config=.golangci.yml ./...
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
	rm -R mocks || true
	mockery
	make fmt

.PHONY: examples
examples:
	go run docs/assets/examples/addpage/main.go
	go run docs/assets/examples/autorow/main.go
	go run docs/assets/examples/background/main.go
	go run docs/assets/examples/barcodegrid/main.go
	go run docs/assets/examples/billing/main.go
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
	go test docs/assets/examples/unittests/main_test.go
