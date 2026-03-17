GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
BINARY := flicksy

.PHONY: build test run

build:
	$(GO) build -o $(BINARY) ./cmd/flicksy

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/flicksy
