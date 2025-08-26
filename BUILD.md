# Build Instructions

This project uses [just](https://github.com/casey/just) as its command runner and build system, replacing traditional Makefiles with better cross-platform support.

## Prerequisites

1. **Go 1.21+** - Required for building the application
2. **just** - Command runner (see installation below)

### Installing just

#### Windows
```bash
# Using scoop
scoop install just

# Using cargo
cargo install just

# Download binary from GitHub releases
# https://github.com/casey/just/releases
```

#### macOS
```bash
# Using Homebrew
brew install just

# Using MacPorts
port install just
```

#### Linux
```bash
# Using snap
snap install --edge --classic just

# Using cargo
cargo install just

# Download pre-built binary
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
```

## Quick Start

```bash
# Show all available commands
just

# Build the application
just build

# Build and run
just run

# Run tests
just test

# List available serial ports
just list

# Connect to a serial port
just connect COM3
```

## Common Commands

### Development
```bash
# Format code
just fmt

# Run tests with coverage
just coverage

# Build development version (with debug symbols)
just dev

# Auto-rebuild on file changes (requires watchexec)
just watch
```

### Building
```bash
# Build for current platform
just build

# Build optimized release
just release-build

# Build for all platforms
just cross

# Build for specific platform
just windows  # Windows (amd64, 386)
just linux    # Linux (amd64, arm64, arm)
just darwin   # macOS (amd64, arm64)
```

### Serial Terminal Specific
```bash
# List serial ports
just list

# List ports with detailed information
just list-details

# Connect to a port
just connect COM3

# Test connection with custom baud rate
just test-connection COM3 9600
```

### Maintenance
```bash
# Clean build artifacts
just clean

# Update dependencies
just deps

# Run all pre-commit checks
just check
```

## justfile vs Makefile

The justfile provides several advantages over traditional Makefiles:

1. **Better cross-platform support** - Works consistently on Windows, macOS, and Linux
2. **Cleaner syntax** - No need for special variables like `$@` or `$<`
3. **Built-in help** - `just --list` shows all available commands
4. **Parameter support** - Easy to pass arguments to recipes
5. **No implicit rules** - More predictable behavior

## Customization

Edit the `justfile` to:
- Change the application name or version
- Add new build targets
- Modify build flags
- Add custom commands

## Troubleshooting

### Command not found: just
Make sure just is installed and in your PATH.

### Permission denied
On Unix-like systems, you may need to use `sudo` for installation commands.

### Build fails on macOS
The macOS build requires CGO for USB device enumeration. Make sure you have Xcode Command Line Tools installed:
```bash
xcode-select --install
```

## Related Files

- `justfile` - The build configuration file
- `Makefile` - Legacy Makefile (can be removed if using just)
- `go.mod` - Go module dependencies
- `go.sum` - Go module checksums