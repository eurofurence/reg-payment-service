checktool = $(shell command -v $1 2>/dev/null)
tool = $(if $(call checktool, $(firstword $1)), $1, @echo "$(firstword $1) was not found on the system. Please install it")

GO ?= $(call tool, go)
GOTEST ?= $(GO) test -v ./...

GOBUILD ?= $(GO) build
GOBUILD_ARGS ?= -ldflags "-s -w" -trimpath

GOLANGCI_LINT ?= $(call tool,golangci-lint)

.PHONY: test
test:
	@$(GO) clean -testcache
	@$(GOTEST)

.PHONY: lint
lint:
	@$(GOLANGCI_LINT) run ./...

.PHONY: build
build: lint
	@$(GOBUILD) $(GOBUILD_ARGS) -o build/service cmd/main.go
	