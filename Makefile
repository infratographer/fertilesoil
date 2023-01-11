BIN?=treeman

# Utility settings
TOOLS_DIR := .tools
GOLANGCI_LINT_VERSION = v1.50.1

# Container build settings
CONTAINER_BUILD_CMD?=docker build

# Container settings
CONTAINER_REPO?=ghcr.io/infratographer/fertilesoil
TREEMAN_CONTAINER_IMAGE_NAME = $(CONTAINER_REPO)/treeman
CONTAINER_TAG?=latest

# OpenAPI settings
OAPI_CODEGEN_CMD?=go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen

# go files to be checked
GO_FILES=$(shell git ls-files '*.go')

## Targets

.PHONY: build
build:
	go build -o $(BIN) ./main.go

.PHONY: test
test:
	@echo Running unit tests...
	@go test -timeout 30s -cover -short  -tags testtools ./...

.PHONY: coverage
coverage:
	@echo Generating coverage report...
	@go test -timeout 30s -tags testtools ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint

golint: | vendor $(TOOLS_DIR)/golangci-lint
	@echo Linting Go files...
	@$(TOOLS_DIR)/golangci-lint run

clean:
	@echo Cleaning...
	@rm -rf coverage.out
	@go clean -testcache
	@rm -r $(TOOLS_DIR)

vendor:
	@go mod download
	@go mod tidy

.PHONY: gci-diff gci-write gci
gci-diff: $(GO_FILES) | gci-tool
	@gci diff -s 'standard,default,prefix(github.com/infratographer)' $^

gci-write: $(GO_FILES) | gci-tool
	@gci write -s 'standard,default,prefix(github.com/infratographer)' $^

gci: | gci-diff gci-write

image: treeman-image

treeman-image:
	$(CONTAINER_BUILD_CMD) -f images/treeman/Dockerfile . -t $(TREEMAN_CONTAINER_IMAGE_NAME):$(CONTAINER_TAG)

.PHONY: generate
generate: openapi

.PHONY: openapi
openapi: openapi-types openapi-spec

.PHONY: openapi-types
openapi-types:
	@echo Generating OpenAPI types...
	@$(OAPI_CODEGEN_CMD) -package v1 \
		-generate types \
		-o api/v1/types.gen.go treeman-openapi-v1.yaml

.PHONY: openapi-spec
openapi-spec:
	@echo Generating OpenAPI spec...
	@$(OAPI_CODEGEN_CMD) -package v1 \
		-generate spec \
		-o api/v1/openapi.gen.go treeman-openapi-v1.yaml

# Tools setup
$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

$(TOOLS_DIR)/golangci-lint: $(TOOLS_DIR)
	export \
		VERSION=$(GOLANGCI_LINT_VERSION) \
		URL=https://raw.githubusercontent.com/golangci/golangci-lint \
		BINDIR=$(TOOLS_DIR) && \
	curl -sfL $$URL/$$VERSION/install.sh | sh -s $$VERSION
	$(TOOLS_DIR)/golangci-lint version
	$(TOOLS_DIR)/golangci-lint linters

.PHONY: gci-tool
gci-tool:
	@which gci &>/dev/null \
		|| echo Installing gci tool \
		&& go install github.com/daixiang0/gci@latest
