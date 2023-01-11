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

clean: dev-infra-down
	@echo Cleaning...
	@rm -f coverage.out
	@go clean -testcache
	@rm -rf $(TOOLS_DIR)
	@rm -f nkey.key nkey.pub

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

nkey.key: | nk-tool
	@echo Generating nats $@
	@nk -gen user -pubout > $@

nkey.pub: nkey.key | nk-tool
	@echo Generating nats $@
	@nk -inkey $< -pubout > $@

.PHONY: nkey
nkey: nkey.key nkey.pub

.PHONY: dev-infra-up dev-infra-down
dev-infra-up: compose.yaml nkey
	@echo Starting services
	@docker compose up -d

	@echo Running migrations
	@export FERTILESOIL_CRDB_HOST=localhost:26257 \
		FERTILESOIL_CRDB_USER=root \
		FERTILESOIL_CRDB_PARAMS=sslmode=disable \
		&& sleep 2 \
		&& while ! go run ./main.go migrate; do \
				echo attempting again in 2 seconds; \
				sleep 2; \
			done

dev-infra-down: compose.yaml
	@echo Stopping services
	@docker compose down

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

.PHONY: nk-tool
nk-tool:
	@which nk &>/dev/null || \
		echo Installing "nk" tool && \
		go install github.com/nats-io/nkeys/nk@latest && \
		export PATH=$$PATH:$(shell go env GOPATH)
