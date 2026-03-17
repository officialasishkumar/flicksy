GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
BINARY := bloop

.PHONY: build test run

build:
	$(GO) build -o $(BINARY) ./cmd/bloop

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/bloop
