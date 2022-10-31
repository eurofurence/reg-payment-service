# On Windows OS is set. This makefile requires Linux or MacOS
ifeq ($(OS),)
SHELL := /bin/bash
MAKE ?= make

checktool = $(shell command -v $1 2>/dev/null)
tool = $(if $(call checktool, $(firstword $1)), $1, @echo "$(firstword $1) was not found on the system. Please install it")

GO ?= $(call checktool, go)
GOTEST ?= $(GO) test
GOTEST_ARGS ?= -timeout 2m -count 1 -cover

GOBUILD ?= $(GO) build
GOBUILD_ARGS ?= -ldflags "-s -w" -trimpath

GOLANGCI_LINT ?= $(call tool,golangci-lint)
DOCKER_COMPOSE ?= $(call tool, docker-compose)

.PHONY: test
test: lint
	@$(GO) clean -testcache
	$(GOTEST) $(GOTEST_ARGS) ./... -v

.PHONY: lint
lint:
	@$(GOLANGCI_LINT) run ./...

.PHONY: build
build: lint
	@$(GOBUILD) $(GOBUILD_ARGS) -o build/service cmd/main.go

.PHONY: up
up:
	@$(DOCKER_COMPOSE) up -d --build

.PHONY: down
down:
	@$(DOCKER_COMPOSE) down
endif
