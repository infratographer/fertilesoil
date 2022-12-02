BIN?=treeman

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
	@go test -timeout 30s -tags testtools ./... -race -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out
	@go tool cover -html=coverage.out
