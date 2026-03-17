GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
BINARY := filmpal

.PHONY: build test run

build:
	$(GO) build -o $(BINARY) ./cmd/filmpal

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/filmpal
