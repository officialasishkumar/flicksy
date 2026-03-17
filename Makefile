GO ?= $(shell command -v go || echo /opt/homebrew/bin/go)
GOLANGCI_LINT ?= $(shell command -v golangci-lint || echo $(CURDIR)/.bin/golangci-lint)
GOLANGCI_LINT_VERSION ?= v2.11.3
BINARY := flicksy
DIST_DIR := dist
VERSION ?= snapshot

.PHONY: build test run fmt vet lint ci release clean

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

lint:
	@test -x "$(GOLANGCI_LINT)" || (echo "install golangci-lint with: GOBIN=$(CURDIR)/.bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)" && exit 1)
	$(GOLANGCI_LINT) run ./...

ci: fmt vet test build

release:
	GO="$(GO)" ./scripts/release.sh "$(VERSION)"

clean:
	rm -rf $(BINARY) $(DIST_DIR)
