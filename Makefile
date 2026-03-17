GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
BINARY := cinebuddy

.PHONY: build test run

build:
	$(GO) build -o $(BINARY) ./cmd/cinebuddy

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/cinebuddy
