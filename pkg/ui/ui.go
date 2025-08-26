// Package ui provides user interface functionality
package ui

import (
	"fmt"
	"sterm/pkg/config"
	"sterm/pkg/history"
	"sterm/pkg/serial"
	"sterm/pkg/terminal"
	"time"
)

// Application represents the main application controller
type Application struct {
	terminal       terminal.Terminal
	configManager  config.ConfigManager
	historyManager history.HistoryManager
	serialPort     serial.SerialPort
	currentSession *Session
	isRunning      bool
	version        string
}

// NewApplication creates a new application instance
func NewApplication(version string) *Application {
	return &Application{
		version:   version,
		isRunning: false,
	}
}

// Session represents an active communication session
type Session struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Config      config.ConfigInfo `json:"config"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	DataSize    int64             `json:"data_size"`
	IsActive    bool              `json:"is_active"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// Validate checks if the session is valid
func (s Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if s.Name == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if err := s.Config.Validate(); err != nil {
		return fmt.Errorf("invalid session config: %w", err)
	}

	if s.StartTime.IsZero() {
		return fmt.Errorf("start time cannot be zero")
	}

	if s.EndTime != nil && s.EndTime.Before(s.StartTime) {
		return fmt.Errorf("end time cannot be before start time")
	}

	if s.DataSize < 0 {
		return fmt.Errorf("data size cannot be negative")
	}

	return nil
}

// Duration returns the duration of the session
func (s Session) Duration() time.Duration {
	if s.EndTime != nil {
		return s.EndTime.Sub(s.StartTime)
	}
	if s.IsActive {
		return time.Since(s.StartTime)
	}
	return 0
}

// NewSession creates a new session with the given configuration
func NewSession(name string, configInfo config.ConfigInfo) *Session {
	return &Session{
		ID:        fmt.Sprintf("session_%d", time.Now().Unix()),
		Name:      name,
		Config:    configInfo,
		StartTime: time.Now(),
		IsActive:  true,
		DataSize:  0,
		Tags:      make([]string, 0),
	}
}

// ErrorType represents different types of application errors
type ErrorType int

const (
	ErrorSerial ErrorType = iota
	ErrorConfig
	ErrorTerminal
	ErrorHistory
	ErrorFile
	ErrorValidation
	ErrorNetwork
	ErrorPermission
)

// String returns the string representation of ErrorType
func (e ErrorType) String() string {
	types := []string{
		"serial", "config", "terminal", "history", "file", "validation", "network", "permission",
	}

	if int(e) < len(types) {
		return types[e]
	}
	return "unknown"
}

// AppError represents an application-specific error
type AppError struct {
	Type      ErrorType `json:"type"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Cause     error     `json:"cause,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Context   string    `json:"context,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s", e.Type.String(), e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("[%s] %s", e.Type.String(), e.Message)
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, code, message string, cause error) *AppError {
	return &AppError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// ApplicationConfig represents the application configuration
type ApplicationConfig struct {
	DefaultSerialConfig serial.SerialConfig `json:"default_serial_config"`
	HistoryMaxSize      int                 `json:"history_max_size"`
	TerminalWidth       int                 `json:"terminal_width"`
	TerminalHeight      int                 `json:"terminal_height"`
	AutoSaveHistory     bool                `json:"auto_save_history"`
	LogLevel            string              `json:"log_level"`
	ConfigDir           string              `json:"config_dir"`
}

// DefaultApplicationConfig returns a default application configuration
func DefaultApplicationConfig() ApplicationConfig {
	return ApplicationConfig{
		DefaultSerialConfig: serial.DefaultConfig(),
		HistoryMaxSize:      10 * 1024 * 1024, // 10MB
		TerminalWidth:       80,
		TerminalHeight:      24,
		AutoSaveHistory:     true,
		LogLevel:            "info",
		ConfigDir:           ".sterm",
	}
}

// Validate checks if the application configuration is valid
func (c ApplicationConfig) Validate() error {
	if err := c.DefaultSerialConfig.Validate(); err != nil {
		return fmt.Errorf("invalid default serial config: %w", err)
	}

	if c.HistoryMaxSize <= 0 {
		return fmt.Errorf("history max size must be positive, got: %d", c.HistoryMaxSize)
	}

	if c.TerminalWidth <= 0 {
		return fmt.Errorf("terminal width must be positive, got: %d", c.TerminalWidth)
	}

	if c.TerminalHeight <= 0 {
		return fmt.Errorf("terminal height must be positive, got: %d", c.TerminalHeight)
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLogLevel := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			validLogLevel = true
			break
		}
	}
	if !validLogLevel {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	if c.ConfigDir == "" {
		return fmt.Errorf("config directory cannot be empty")
	}

	return nil
}

// TODO: Implement Application methods in later tasks
