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

.PHONY: help
help: Makefile ## Print help
	@grep -h "##" $(MAKEFILE_LIST) | grep -v grep | sed -e 's/:.*##/#/' | column -c 2 -t -s#

.PHONY: build
build:  ## Builds treeman binary.
	go build -o $(BIN) ./main.go

.PHONY: test
test:  ## Runs unit tests.
	@echo Running unit tests...
	@go test -timeout 30s -cover -short  -tags testtools ./...

.PHONY: coverage
coverage:  ## Generates a test coverage report.
	@echo Generating coverage report...
	@go test -timeout 30s -tags testtools ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out

lint: golint  ## Runs all lint checks.

golint: | vendor $(TOOLS_DIR)/golangci-lint  ## Runs Go lint checks.
	@echo Linting Go files...
	@$(TOOLS_DIR)/golangci-lint run

clean: dev-infra-down  ## Cleans generated files.
	@echo Cleaning...
	@rm -f coverage.out
	@go clean -testcache
	@rm -rf $(TOOLS_DIR)
	@rm -f nkey.key nkey.pub

vendor:  ## Downloads and tidies go modules.
	@go mod download
	@go mod tidy

.PHONY: gci-diff gci-write gci
gci-diff: $(GO_FILES) | gci-tool  ## Outputs improper go import ordering.
	@gci diff -s 'standard,default,prefix(github.com/infratographer)' $^

gci-write: $(GO_FILES) | gci-tool  ## Checks and updates all go files for proper import ordering.
	@gci write -s 'standard,default,prefix(github.com/infratographer)' $^

gci: | gci-diff gci-write  ## Outputs and corrects all improper go import ordering.

image: treeman-image  ## Builds all docker images.

treeman-image:  ## Builds the treeman docker image.
	$(CONTAINER_BUILD_CMD) -f images/treeman/Dockerfile . -t $(TREEMAN_CONTAINER_IMAGE_NAME):$(CONTAINER_TAG)

.PHONY: generate
generate: openapi  ## Generates OpenAPI types and specs.

.PHONY: openapi
openapi: openapi-types openapi-spec  ## Generates OpenAPI types and specs.

.PHONY: openapi-types
openapi-types:  ## Generates OpenAPI types.
	@echo Generating OpenAPI types...
	@$(OAPI_CODEGEN_CMD) -package v1 \
		-generate types \
		-o api/v1/types.gen.go treeman-openapi-v1.yaml

.PHONY: openapi-spec
openapi-spec:  ## Generates OpenAPI specs.
	@echo Generating OpenAPI spec...
	@$(OAPI_CODEGEN_CMD) -package v1 \
		-generate spec \
		-o api/v1/openapi.gen.go treeman-openapi-v1.yaml

nkey.key: | nk-tool  ## Generates a new NATS user key.
	@echo Generating nats $@
	@nk -gen user -pubout > $@

nkey.pub: nkey.key | nk-tool  ## Exports the NATS user public key.
	@echo Generating nats $@
	@nk -inkey $< -pubout > $@

.PHONY: nkey
nkey: nkey.key nkey.pub  ## Generates and exports a new NATS user public and private keys.

.PHONY: dev-infra-up dev-infra-down
dev-infra-up: compose.yaml nkey  ## Starts local services to simplify local development.
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

dev-infra-down: compose.yaml  ## Stops local services used for local development.
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
