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

image: treeman-image

treeman-image:
	$(CONTAINER_BUILD_CMD) -f images/treeman/Dockerfile . -t $(TREEMAN_CONTAINER_IMAGE_NAME):$(CONTAINER_TAG)

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
