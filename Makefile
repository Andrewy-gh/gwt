# gwt - Git Worktree Manager
# Makefile for common development tasks

.PHONY: help build test clean install run fmt vet doctor all

# Default target
help:
	@echo "gwt - Git Worktree Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  make build    - Build the binary"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make install  - Install to GOPATH/bin"
	@echo "  make run      - Build and run gwt doctor"
	@echo "  make fmt      - Format code"
	@echo "  make vet      - Run go vet"
	@echo "  make doctor   - Run gwt doctor"
	@echo "  make all      - Format, vet, test, and build"

# Build the binary
build:
	@echo "Building gwt..."
	@go build -o gwt ./cmd/gwt

# Build with version information
build-release:
	@echo "Building gwt with version info..."
	@go build \
		-ldflags "-X github.com/Andrewy-gh/gwt/internal/version.Version=$$(git describe --tags --always) \
		          -X github.com/Andrewy-gh/gwt/internal/version.Commit=$$(git rev-parse --short HEAD) \
		          -X github.com/Andrewy-gh/gwt/internal/version.BuildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o gwt \
		./cmd/gwt

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f gwt gwt.exe
	@rm -f coverage.out coverage.html
	@rm -rf dist/

# Install to GOPATH/bin
install:
	@echo "Installing gwt..."
	@go install ./cmd/gwt

# Build and run gwt doctor
run: build
	@./gwt doctor

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run gwt doctor
doctor: build
	@./gwt doctor

# Do everything: format, vet, test, build
all: fmt vet test build
	@echo "All checks passed!"

# Cross-compile for all platforms
cross:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -o dist/gwt-linux-amd64 ./cmd/gwt
	@GOOS=darwin GOARCH=amd64 go build -o dist/gwt-darwin-amd64 ./cmd/gwt
	@GOOS=darwin GOARCH=arm64 go build -o dist/gwt-darwin-arm64 ./cmd/gwt
	@GOOS=windows GOARCH=amd64 go build -o dist/gwt-windows-amd64.exe ./cmd/gwt
	@echo "Binaries created in dist/"
