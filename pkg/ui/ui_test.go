package ui

import (
	"sterm/pkg/config"
	"sterm/pkg/serial"
	"testing"
	"time"
)

func TestSession_Validate(t *testing.T) {
	validConfig := config.ConfigInfo{
		Name: "test-config",
		Config: serial.SerialConfig{
			Port:     "COM1",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  time.Second * 5,
		},
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	tests := []struct {
		name    string
		session Session
		wantErr bool
	}{
		{
			name: "valid session",
			session: Session{
				ID:        "session_123",
				Name:      "test-session",
				Config:    validConfig,
				StartTime: time.Now(),
				IsActive:  true,
				DataSize:  1024,
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			session: Session{
				ID:        "",
				Name:      "test-session",
				Config:    validConfig,
				StartTime: time.Now(),
				IsActive:  true,
				DataSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			session: Session{
				ID:        "session_123",
				Name:      "",
				Config:    validConfig,
				StartTime: time.Now(),
				IsActive:  true,
				DataSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "invalid config",
			session: Session{
				ID:   "session_123",
				Name: "test-session",
				Config: config.ConfigInfo{
					Name: "",
					Config: serial.SerialConfig{
						Port:     "COM1",
						BaudRate: 115200,
						DataBits: 8,
						StopBits: 1,
						Parity:   "none",
						Timeout:  time.Second * 5,
					},
					CreatedAt:  time.Now(),
					LastUsedAt: time.Now(),
				},
				StartTime: time.Now(),
				IsActive:  true,
				DataSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "zero start time",
			session: Session{
				ID:        "session_123",
				Name:      "test-session",
				Config:    validConfig,
				StartTime: time.Time{},
				IsActive:  true,
				DataSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "end time before start time",
			session: Session{
				ID:        "session_123",
				Name:      "test-session",
				Config:    validConfig,
				StartTime: time.Now(),
				EndTime:   func() *time.Time { t := time.Now().Add(-time.Hour); return &t }(),
				IsActive:  false,
				DataSize:  1024,
			},
			wantErr: true,
		},
		{
			name: "negative data size",
			session: Session{
				ID:        "session_123",
				Name:      "test-session",
				Config:    validConfig,
				StartTime: time.Now(),
				IsActive:  true,
				DataSize:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSession_Duration(t *testing.T) {
	startTime := time.Now()

	tests := []struct {
		name     string
		session  Session
		expected time.Duration
	}{
		{
			name: "active session",
			session: Session{
				StartTime: startTime,
				IsActive:  true,
			},
			expected: time.Since(startTime), // Approximate
		},
		{
			name: "ended session",
			session: Session{
				StartTime: startTime,
				EndTime:   func() *time.Time { t := startTime.Add(time.Hour); return &t }(),
				IsActive:  false,
			},
			expected: time.Hour,
		},
		{
			name: "inactive session without end time",
			session: Session{
				StartTime: startTime,
				IsActive:  false,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.session.Duration()

			if tt.name == "active session" {
				// For active sessions, check if duration is reasonable (within 1 second)
				if duration < 0 || duration > time.Second {
					t.Errorf("Session.Duration() for active session = %v, should be small positive duration", duration)
				}
			} else {
				if duration != tt.expected {
					t.Errorf("Session.Duration() = %v, want %v", duration, tt.expected)
				}
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	name := "test-session"
	configInfo := config.ConfigInfo{
		Name: "test-config",
		Config: serial.SerialConfig{
			Port:     "COM1",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  time.Second * 5,
		},
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	session := NewSession(name, configInfo)

	if session == nil {
		t.Error("NewSession() returned nil")
	}

	if session.Name != name {
		t.Errorf("NewSession() Name = %s, want %s", session.Name, name)
	}

	if session.Config.Name != configInfo.Name {
		t.Errorf("NewSession() Config.Name = %s, want %s", session.Config.Name, configInfo.Name)
	}

	if !session.IsActive {
		t.Error("NewSession() should create active session")
	}

	if session.DataSize != 0 {
		t.Errorf("NewSession() DataSize = %d, want 0", session.DataSize)
	}

	if session.ID == "" {
		t.Error("NewSession() should generate non-empty ID")
	}

	if session.StartTime.IsZero() {
		t.Error("NewSession() should set start time")
	}

	if session.Tags == nil {
		t.Error("NewSession() should initialize Tags slice")
	}

	if err := session.Validate(); err != nil {
		t.Errorf("NewSession() should create valid session: %v", err)
	}
}

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorSerial, "serial"},
		{ErrorConfig, "config"},
		{ErrorTerminal, "terminal"},
		{ErrorHistory, "history"},
		{ErrorFile, "file"},
		{ErrorValidation, "validation"},
		{ErrorNetwork, "network"},
		{ErrorPermission, "permission"},
		{ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.errorType.String(); got != tt.expected {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "error without cause",
			appError: &AppError{
				Type:    ErrorSerial,
				Message: "serial port not found",
			},
			expected: "[serial] serial port not found",
		},
		{
			name: "error with cause",
			appError: &AppError{
				Type:    ErrorConfig,
				Message: "failed to load config",
				Cause:   &AppError{Type: ErrorFile, Message: "file not found"},
			},
			expected: "[config] failed to load config: [file] file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appError.Error(); got != tt.expected {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewAppError(t *testing.T) {
	errorType := ErrorSerial
	code := "SERIAL_001"
	message := "serial port error"
	cause := &AppError{Type: ErrorFile, Message: "underlying error"}

	appError := NewAppError(errorType, code, message, cause)

	if appError == nil {
		t.Error("NewAppError() returned nil")
	}

	if appError.Type != errorType {
		t.Errorf("NewAppError() Type = %v, want %v", appError.Type, errorType)
	}

	if appError.Code != code {
		t.Errorf("NewAppError() Code = %s, want %s", appError.Code, code)
	}

	if appError.Message != message {
		t.Errorf("NewAppError() Message = %s, want %s", appError.Message, message)
	}

	if appError.Cause != cause {
		t.Errorf("NewAppError() Cause = %v, want %v", appError.Cause, cause)
	}

	if appError.Timestamp.IsZero() {
		t.Error("NewAppError() should set timestamp")
	}
}

func TestApplicationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ApplicationConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultApplicationConfig(),
			wantErr: false,
		},
		{
			name: "invalid serial config",
			config: ApplicationConfig{
				DefaultSerialConfig: serial.SerialConfig{
					Port:     "",
					BaudRate: 115200,
					DataBits: 8,
					StopBits: 1,
					Parity:   "none",
					Timeout:  time.Second * 5,
				},
				HistoryMaxSize: 1024,
				TerminalWidth:  80,
				TerminalHeight: 24,
				LogLevel:       "info",
				ConfigDir:      ".sterm",
			},
			wantErr: true,
		},
		{
			name: "zero history max size",
			config: ApplicationConfig{
				DefaultSerialConfig: serial.DefaultConfig(),
				HistoryMaxSize:      0,
				TerminalWidth:       80,
				TerminalHeight:      24,
				LogLevel:            "info",
				ConfigDir:           ".sterm",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: ApplicationConfig{
				DefaultSerialConfig: serial.DefaultConfig(),
				HistoryMaxSize:      1024,
				TerminalWidth:       80,
				TerminalHeight:      24,
				LogLevel:            "invalid",
				ConfigDir:           ".sterm",
			},
			wantErr: true,
		},
		{
			name: "empty config dir",
			config: ApplicationConfig{
				DefaultSerialConfig: serial.DefaultConfig(),
				HistoryMaxSize:      1024,
				TerminalWidth:       80,
				TerminalHeight:      24,
				LogLevel:            "info",
				ConfigDir:           "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplicationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultApplicationConfig(t *testing.T) {
	config := DefaultApplicationConfig()

	if err := config.Validate(); err != nil {
		t.Errorf("DefaultApplicationConfig() should return valid config: %v", err)
	}

	if config.HistoryMaxSize != 10*1024*1024 {
		t.Errorf("DefaultApplicationConfig() HistoryMaxSize = %d, want %d", config.HistoryMaxSize, 10*1024*1024)
	}

	if config.TerminalWidth != 80 {
		t.Errorf("DefaultApplicationConfig() TerminalWidth = %d, want 80", config.TerminalWidth)
	}

	if config.TerminalHeight != 24 {
		t.Errorf("DefaultApplicationConfig() TerminalHeight = %d, want 24", config.TerminalHeight)
	}

	if !config.AutoSaveHistory {
		t.Error("DefaultApplicationConfig() AutoSaveHistory should be true")
	}

	if config.LogLevel != "info" {
		t.Errorf("DefaultApplicationConfig() LogLevel = %s, want info", config.LogLevel)
	}

	if config.ConfigDir != ".sterm" {
		t.Errorf("DefaultApplicationConfig() ConfigDir = %s, want .sterm", config.ConfigDir)
	}
}

func TestNewApplication(t *testing.T) {
	version := "1.0.0"
	app := NewApplication(version)

	if app == nil {
		t.Error("NewApplication() returned nil")
	}

	if app.version != version {
		t.Errorf("NewApplication() version = %s, want %s", app.version, version)
	}

	if app.isRunning {
		t.Error("NewApplication() should create non-running application")
	}

	if app.currentSession != nil {
		t.Error("NewApplication() should not have current session")
	}
}
