# Serial Terminal

A cross-platform serial port terminal application written in Go.

## Project Structure

```
sterm/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── pkg/                    # Package directory
│   ├── serial/             # Serial port communication
│   │   └── serial.go
│   ├── terminal/           # Terminal emulation
│   │   └── terminal.go
│   ├── config/             # Configuration management
│   │   └── config.go
│   ├── history/            # Communication history
│   │   └── history.go
│   └── ui/                 # User interface and application logic
│       └── ui.go
└── README.md               # This file
```

## Features

- Cross-platform serial port communication (Windows, Linux, macOS)
- VT100/ANSI terminal emulation
- Configuration management for different serial devices
- Communication history with large buffer support
- Command-line interface with keyboard shortcuts

## Build

```bash
go build -o sterm
```

## Usage

```bash
./sterm --help
```

## Development Status

This project is currently under development. The basic project structure has been established.