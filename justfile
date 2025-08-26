# justfile for sterm
# Cross-platform build system using just (https://github.com/casey/just)

# Use PowerShell on Windows for better compatibility
set windows-shell := ["pwsh", "-Command"]

# Variables
app_name := "sterm"
version := "1.0.0"
build_dir := "build"

# Platform-specific executable extension
exe_ext := if os_family() == "windows" { ".exe" } else { "" }
exe := build_dir / app_name + exe_ext

# Default recipe - show help
default:
    @just --list --unsorted

# Build for current platform
build:
    go build -o {{exe}} .

# Run tests
test:
    go test -v -race ./...

# Run tests with coverage
coverage:
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
    go clean


# Download and tidy dependencies
deps:
    go mod download
    go mod tidy

# Build and run
run: build
    ./{{exe}}

# Run with arguments
run-args *args: build
    ./{{exe}} {{args}}

# List available ports
list: build
    ./{{exe}} list

# List ports with details
list-details: build
    ./{{exe}} list -d

# Connect to a port (usage: just connect COM3)
connect port: build
    ./{{exe}} connect {{port}}

# Build for all platforms
cross: windows linux

# Build for Windows (both amd64 and 386)
windows:
    $env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o {{build_dir}}/{{app_name}}-windows-amd64.exe .
    $env:GOOS="windows"; $env:GOARCH="386"; $env:CGO_ENABLED="0"; go build -o {{build_dir}}/{{app_name}}-windows-386.exe .

# Build for Linux (amd64, arm64, and arm)
linux:
    $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o {{build_dir}}/{{app_name}}-linux-amd64 .
    $env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="0"; go build -o {{build_dir}}/{{app_name}}-linux-arm64 .
    $env:GOOS="linux"; $env:GOARCH="arm"; $env:CGO_ENABLED="0"; go build -o {{build_dir}}/{{app_name}}-linux-arm .

# Format code
fmt:
    go fmt ./...
    gofmt -w .

# Run linter (requires golangci-lint)
lint:
    golangci-lint run ./...

# Quick development build without optimizations
dev:
    go build -gcflags="all=-N -l" -o {{exe}} .

# Run and watch for changes (requires watchexec)
watch *args:
    watchexec -e go -r -- just run-args {{args}}

# Test connection to a port with custom baud rate
test-connection port baud="115200":
    @just run-args connect {{port}} -b {{baud}}

# Show current version
version:
    @echo "{{app_name}} version {{version}}"

# Run all checks before commit
check: fmt test
    @echo "All checks passed!"

# Build optimized release binary for current platform
release-build:
    go build -ldflags="-s -w" -trimpath -o {{exe}} .

# Create release for current platform
release-current: release-build
    mkdir {{build_dir}}/release
    Copy-Item {{exe}} {{build_dir}}/release/
    Copy-Item README.md {{build_dir}}/release/
    Copy-Item LICENSE {{build_dir}}/release/
