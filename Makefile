# asana-cli Makefile

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/vincentsch/asana-cli/internal/version.Version=$(VERSION) -X github.com/vincentsch/asana-cli/internal/version.Commit=$(COMMIT)

BUILD_DIR := dist
BINARY    := asana
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build build-all install clean test conformance lint docs docs-check

# Build for current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/asana

# Cross-compile for all platforms
build-all: $(BUILD_DIR)
	@set -e; \
	rm -f $(BUILD_DIR)/$(BINARY)-* $(BUILD_DIR)/checksums.txt; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(BUILD_DIR)/$(BINARY)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output=$$output.exe; fi; \
		echo "Building $$output..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output ./cmd/asana; \
	done

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Install to ~/.local/bin
install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed to $(HOME)/.local/bin/$(BINARY)"

# Remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test ./...

# Build and score asana against local, credential-free fixtures
conformance:
	go test ./internal/conformance -run TestAsanaConformance -count=1 -v

# Regenerate the command reference from the rungrad command tree
docs:
	go run ./cmd/generate-command-docs

# Verify committed command docs match the rungrad command tree
docs-check:
	go run ./cmd/generate-command-docs --check

# Run linter (if available)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi
