BIN?=treeman

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build:
	go build -o $(BIN) ./main.go
