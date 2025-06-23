# Get version from git tags, fallback to commit hash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")

# Build flags for version injection
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

# Binary output directory
BIN_DIR = bin

.PHONY: build install dev test clean help

# Default target
all: build

# Build the binary
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/gofeedfinder ./cmd/gofeedfinder

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/gofeedfinder

# Run directly without building binary
dev:
	go run $(LDFLAGS) ./cmd/gofeedfinder

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -v -cover ./...

# Run static analysis
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary to $(BIN_DIR)/gofeedfinder"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  dev        - Run directly with 'go run'"
	@echo "  test       - Run all tests"
	@echo "  test-cover - Run tests with coverage"
	@echo "  vet        - Run static analysis"
	@echo "  clean      - Remove build artifacts"
	@echo "  help       - Show this help"
	@echo ""
	@echo "Version: $(VERSION)"
