# Serial Terminal (sterm)

A cross-platform serial port terminal emulator written in Go with full VT100/ANSI support.

## Features

- **Cross-platform support** - Works on Windows, Linux, and macOS
- **Full VT100/ANSI terminal emulation** - Complete escape sequence support
- **Interactive terminal interface** - Real-time serial communication with color support
- **Configuration management** - Save and manage multiple serial port configurations
- **Session history** - Record and replay communication sessions
- **Mouse support** - When enabled by terminal applications
- **Scrollback buffer** - Navigate through command history
- **Keyboard shortcuts** - Quick access to common functions
- **Multiple output formats** - Plain text, timestamped, and JSON history export

## Project Structure

```
sterm/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── justfile               # Build system configuration (see BUILD.md)
├── cmd/                   # CLI command definitions
│   ├── root.go           # Root command setup
│   ├── config.go         # Configuration commands
│   ├── connect.go        # Connection commands
│   └── list.go           # Port listing commands
├── pkg/                   # Package directory
│   ├── app/              # Main application controller
│   │   ├── app.go        # Application logic
│   │   └── runner.go     # Application runner
│   ├── serial/           # Serial port communication
│   │   └── serial.go     # Cross-platform serial implementation
│   ├── terminal/         # Terminal emulation engine
│   │   └── terminal.go   # VT100/ANSI parser and emulator
│   ├── config/           # Configuration management
│   │   └── config.go     # Configuration file handling
│   ├── history/          # Communication history
│   │   └── history.go    # History recording and replay
│   ├── menu/             # Interactive menu system
│   │   ├── menu.go       # Menu implementation
│   │   └── overlay.go    # Overlay management
│   └── ui/               # User interface components
│       └── ui.go         # UI data structures
└── README.md             # This file
```

## Quick Start

### Build and Install

Using just (recommended - see [BUILD.md](BUILD.md)):
```bash
# Install just (build system)
# Windows: scoop install just
# macOS: brew install just
# Linux: snap install --classic --edge just

# Build the application
just build

# Build and run immediately
just run
```

Using Go directly:
```bash
go build -o sterm .
```

### Basic Usage

```bash
# List available serial ports
sterm list
sterm list --details  # Show detailed port information

# Connect to a serial port
sterm connect COM3              # Windows
sterm connect /dev/ttyUSB0      # Linux
sterm connect /dev/cu.usbserial # macOS

# Connect with custom settings
sterm connect COM3 --baud 9600 --data 8 --parity none --stop 1
```

### Configuration Management

```bash
# Save current connection as a configuration
sterm config save my-arduino

# List saved configurations
sterm config list

# Connect using a saved configuration
sterm config load my-arduino

# Delete a configuration
sterm config delete my-arduino
```

## Interactive Terminal

Once connected, you have access to a full-featured terminal interface:

### Keyboard Shortcuts
- **F1**: Toggle main menu
- **F8**: Pause/resume data flow
- **Ctrl+Shift+Q**: Exit application
- **Ctrl+Shift+S**: Save session history
- **Ctrl+Shift+C**: Clear terminal screen
- **Alt+C**: Clear screen
- **Alt+H**: Clear scrollback history
- **Alt+R**: Reconnect
- **Alt+S**: Save session to file

### Navigation
- **Shift+PageUp/PageDown**: Scroll through history
- **Shift+Up/Down**: Line-by-line scrolling
- **Ctrl+Home/End**: Jump to top/bottom
- **ESC/Enter/Q**: Exit scroll mode

### Features
- **Local echo**: Optional local character echoing
- **Line wrap**: Configurable line wrapping
- **Mouse support**: Automatic when requested by terminal applications
- **Status bar**: Shows connection info, mode, and statistics

## Advanced Features

### Debug Mode
```bash
sterm --debug connect COM3
# Creates debug log in ~/.sterm/sterm-debug.log
```

### History Export
Sessions are automatically saved and can be exported in multiple formats:
- Plain text
- Timestamped entries
- JSON format with metadata

### Terminal Emulation
- Full VT100/ANSI escape sequence support
- 256-color support
- Mouse tracking (X10, VT200, Button Event, Any Event modes)
- Alternative screen buffer
- Scrollback regions
- Tab stops and character sets

## Requirements

- Go 1.21+ (for building)
- Windows 10+, Linux, or macOS 10.15+
- Serial port access permissions (may require admin/sudo on some systems)

## Development

See [BUILD.md](BUILD.md) for detailed build instructions and development setup.

### Running Tests
```bash
just test          # Run all tests
just coverage      # Run tests with coverage report
just check         # Run all pre-commit checks
```

## License

This project is provided as-is for educational and development purposes.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `just test`
5. Format code: `just fmt`
6. Submit a pull request