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
OAPI_CODEGEN_CMD_VERSION?=v1.12.4

# Dev infra OAuth2 settings
DEV_OAUTH2_ADDR=localhost:8082
DEV_OAUTH2_ISSUER=fertilesoil
DEV_OAUTH2_SCOPE=test
DEV_OAUTH2_SUB=fertilesoil
DEV_OAUTH2_AUD=fertilesoil

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
	@rm -f .dc-data/audit/audit.log
	@rm -f .dc-data/nkey.key .dc-data/nkey.pub .dc-data/oauth2.json

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
openapi-types:  | oapi-codegen-cmd  ## Generates OpenAPI types.
	@echo Generating OpenAPI types...
	@oapi-codegen -package v1 \
		-generate types \
		-o api/v1/types.gen.go treeman-openapi-v1.yaml

.PHONY: openapi-spec
openapi-spec:  | oapi-codegen-cmd  ## Generates OpenAPI specs.
	@echo Generating OpenAPI spec...
	@oapi-codegen -package v1 \
		-generate spec \
		-o api/v1/openapi.gen.go treeman-openapi-v1.yaml

.dc-data/nkey.key: | nk-tool
	@echo Generating nats $@
	@mkdir -p $(shell dirname $@)
	@nk -gen user -pubout > $@

.dc-data/nkey.pub: .dc-data/nkey.key | nk-tool
	@echo Generating nats $@
	@mkdir -p $(shell dirname $@)
	@nk -inkey $< -pubout > $@

define OAUTH2_CONFIG
{
  "interactiveLogin": true,
  "httpServer": "NettyWrapper",
  "tokenCallbacks": [
    {
      "issuerId": "$(DEV_OAUTH2_ISSUER)",
      "tokenExpiry": 120,
      "requestMappings": [
        {
          "requestParam": "scope",
          "match": "$(DEV_OAUTH2_SCOPE)",
          "claims": {
            "sub": "$(DEV_OAUTH2_SUB)",
            "aud": [
              "$(DEV_OAUTH2_AUD)"
            ]
          }
        }
      ]
    }
  ]
}
endef

.dc-data/oauth2.json: export OAUTH2_CONFIG:=$(OAUTH2_CONFIG)
.dc-data/oauth2.json: .dc-data
	@echo Generating OAuth2 config $@
	@echo "$$OAUTH2_CONFIG" > $@

.PHONY: dev-oauth2-token
dev-oauth2-token:  ## Creates a new oauth2 authorization token for testing.
	@echo Generating OAuth2 token
	@echo Audience: $(DEV_OAUTH2_AUD)
	@echo Issuer: http://$(DEV_OAUTH2_ADDR)/$(DEV_OAUTH2_ISSUER)
	@echo JWKS URL: http://$(DEV_OAUTH2_ADDR)/$(DEV_OAUTH2_ISSUER)/jwks
	@curl -s --fail -X POST -H 'Content-Type: application/x-www-form-urlencoded' \
		-d "grant_type=client_credentials&client_id=random&client_secret=random&scope=$(DEV_OAUTH2_SCOPE)" \
		http://$(DEV_OAUTH2_ADDR)/$(DEV_OAUTH2_ISSUER)/token | jq -r 'to_entries[] | [.key, (.value | tostring)] | @tsv'

.dc-data/audit/audit.log:
	@mkdir -p $(shell dirname $@)
	@touch $@

.PHONY: dev-infra-up dev-infra-down
dev-infra-up: compose.yaml .dc-data/nkey.key .dc-data/nkey.pub .dc-data/oauth2.json .dc-data/audit/audit.log   ## Starts local services to simplify local development.
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

	@echo
	@echo 'Use "make dev-oauth2-token" to create a token'

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

.PHONY: oapi-codegen-cmd
oapi-codegen-cmd:
	@which oapi-codegen &>/dev/null || \
		echo Installing "oapi-codegen" command && \
		go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@$(OAPI_CODEGEN_CMD_VERSION) && \
		export PATH=$$PATH:$(shell go env GOPATH)
