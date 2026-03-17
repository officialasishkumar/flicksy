GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
BINARY := flicksy
DIST_DIR := dist
VERSION ?= snapshot

.PHONY: build test run fmt vet ci release clean

build:
	$(GO) build -o $(BINARY) ./cmd/flicksy

test:
	$(GO) test ./...

run:
	$(GO) run ./cmd/flicksy

fmt:
	@test -z "$$(gofmt -l .)" || (echo "run gofmt on these files:" && gofmt -l . && exit 1)

vet:
	$(GO) vet ./...

ci: fmt vet test build

release:
	GO="$(GO)" ./scripts/release.sh "$(VERSION)"

clean:
	rm -rf $(BINARY) $(DIST_DIR)
