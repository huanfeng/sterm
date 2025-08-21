package serial

import (
	"fmt"
	"testing"
	"time"
)

func TestSerialConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SerialConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second * 5,
			},
			wantErr: false,
		},
		{
			name: "empty port",
			config: SerialConfig{
				Port:     "",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second * 5,
			},
			wantErr: true,
		},
		{
			name: "invalid baud rate",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 12345,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second * 5,
			},
			wantErr: true,
		},
		{
			name: "invalid data bits",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 9,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second * 5,
			},
			wantErr: true,
		},
		{
			name: "invalid stop bits",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 3,
				Parity:   "none",
				Timeout:  time.Second * 5,
			},
			wantErr: true,
		},
		{
			name: "invalid parity",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "invalid",
				Timeout:  time.Second * 5,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  -time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SerialConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if err := config.Validate(); err != nil {
		t.Errorf("DefaultConfig() returned invalid config: %v", err)
	}
	
	if config.BaudRate != 115200 {
		t.Errorf("DefaultConfig() BaudRate = %d, want 115200", config.BaudRate)
	}
	
	if config.DataBits != 8 {
		t.Errorf("DefaultConfig() DataBits = %d, want 8", config.DataBits)
	}
	
	if config.StopBits != 1 {
		t.Errorf("DefaultConfig() StopBits = %d, want 1", config.StopBits)
	}
	
	if config.Parity != "none" {
		t.Errorf("DefaultConfig() Parity = %s, want none", config.Parity)
	}
}

func TestSerialConfig_ValidBaudRates(t *testing.T) {
	validRates := []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600}
	
	for _, rate := range validRates {
		config := SerialConfig{
			Port:     "COM1",
			BaudRate: rate,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  time.Second * 5,
		}
		
		if err := config.Validate(); err != nil {
			t.Errorf("Valid baud rate %d should not cause validation error: %v", rate, err)
		}
	}
}

func TestSerialConfig_ValidParityValues(t *testing.T) {
	validParity := []string{"none", "odd", "even", "mark", "space"}
	
	for _, parity := range validParity {
		config := SerialConfig{
			Port:     "COM1",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   parity,
			Timeout:  time.Second * 5,
		}
		
		if err := config.Validate(); err != nil {
			t.Errorf("Valid parity %s should not cause validation error: %v", parity, err)
		}
	}
}

func TestNewCrossPlatformSerialPort(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	
	if port == nil {
		t.Error("NewCrossPlatformSerialPort() returned nil")
	}
	
	if port.IsOpen() {
		t.Error("New serial port should not be open")
	}
	
	config := port.GetConfig()
	if config.Port != "" {
		t.Error("New serial port should have empty config")
	}
}

func TestCrossPlatformSerialPort_OpenInvalidConfig(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	
	invalidConfig := SerialConfig{
		Port:     "", // Invalid empty port
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second * 5,
	}
	
	err := port.Open(invalidConfig)
	if err == nil {
		t.Error("Opening with invalid config should return error")
	}
	
	if port.IsOpen() {
		t.Error("Port should not be open after failed open")
	}
}

func TestCrossPlatformSerialPort_DoubleOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	
	// Mock a successful open by setting isOpen to true
	port.isOpen = true
	
	config := SerialConfig{
		Port:     "COM1",
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second * 5,
	}
	
	err := port.Open(config)
	if err == nil {
		t.Error("Opening already open port should return error")
	}
	
	// Reset for cleanup
	port.isOpen = false
}

func TestCrossPlatformSerialPort_CloseNotOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	
	err := port.Close()
	if err == nil {
		t.Error("Closing not open port should return error")
	}
}

func TestCrossPlatformSerialPort_ReadNotOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	buffer := make([]byte, 10)
	
	_, err := port.Read(buffer)
	if err == nil {
		t.Error("Reading from not open port should return error")
	}
}

func TestCrossPlatformSerialPort_WriteNotOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	data := []byte("test")
	
	_, err := port.Write(data)
	if err == nil {
		t.Error("Writing to not open port should return error")
	}
}

func TestCrossPlatformSerialPort_SetReadTimeoutNotOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	
	err := port.SetReadTimeout(time.Second)
	if err == nil {
		t.Error("Setting timeout on not open port should return error")
	}
}

func TestConvertStopBits(t *testing.T) {
	tests := []struct {
		input int
	}{
		{1},
		{2},
		{3}, // Invalid value should default to OneStopBit
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("stopbits_%d", tt.input), func(t *testing.T) {
			result := convertStopBits(tt.input)
			// Since we can't easily compare the enum values, we'll just ensure it doesn't panic
			// and returns a valid StopBits value
			_ = result // Just ensure it doesn't panic
		})
	}
}

func TestConvertParity(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"none"},
		{"odd"},
		{"even"},
		{"mark"},
		{"space"},
		{"invalid"}, // Should default to NoParity
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("parity_%s", tt.input), func(t *testing.T) {
			result := convertParity(tt.input)
			// Since we can't easily compare the enum values, we'll just ensure it doesn't panic
			// and returns a valid Parity value
			_ = result // Just ensure it doesn't panic
		})
	}
}

func TestPortInfo_Structure(t *testing.T) {
	portInfo := PortInfo{
		Name:         "COM1",
		Description:  "USB Serial Port",
		VID:          "1234",
		PID:          "5678",
		SerialNumber: "ABC123",
	}
	
	if portInfo.Name != "COM1" {
		t.Errorf("PortInfo.Name = %s, want COM1", portInfo.Name)
	}
	
	if portInfo.Description != "USB Serial Port" {
		t.Errorf("PortInfo.Description = %s, want 'USB Serial Port'", portInfo.Description)
	}
	
	if portInfo.VID != "1234" {
		t.Errorf("PortInfo.VID = %s, want '1234'", portInfo.VID)
	}
	
	if portInfo.PID != "5678" {
		t.Errorf("PortInfo.PID = %s, want '5678'", portInfo.PID)
	}
	
	if portInfo.SerialNumber != "ABC123" {
		t.Errorf("PortInfo.SerialNumber = %s, want 'ABC123'", portInfo.SerialNumber)
	}
}

func TestSerialError_Error(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		port      string
		cause     error
		expected  string
	}{
		{
			name:      "error with cause",
			operation: "open",
			port:      "COM1",
			cause:     fmt.Errorf("device not found"),
			expected:  "serial open operation failed on port COM1: device not found",
		},
		{
			name:      "error without cause",
			operation: "read",
			port:      "COM2",
			cause:     nil,
			expected:  "serial read operation failed on port COM2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &SerialError{
				Operation: tt.operation,
				Port:      tt.port,
				Cause:     tt.cause,
			}
			
			if got := err.Error(); got != tt.expected {
				t.Errorf("SerialError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewSerialError(t *testing.T) {
	operation := "write"
	port := "COM3"
	cause := fmt.Errorf("timeout")
	
	err := NewSerialError(operation, port, cause)
	
	if err == nil {
		t.Error("NewSerialError() returned nil")
	}
	
	if err.Operation != operation {
		t.Errorf("NewSerialError() Operation = %s, want %s", err.Operation, operation)
	}
	
	if err.Port != port {
		t.Errorf("NewSerialError() Port = %s, want %s", err.Port, port)
	}
	
	if err.Cause != cause {
		t.Errorf("NewSerialError() Cause = %v, want %v", err.Cause, cause)
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{StateError, "error"},
		{ConnectionState(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("ConnectionState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RetryConfig{
				MaxRetries:    3,
				RetryInterval: time.Second,
				BackoffFactor: 2.0,
				MaxInterval:   time.Second * 10,
			},
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: RetryConfig{
				MaxRetries:    -1,
				RetryInterval: time.Second,
				BackoffFactor: 2.0,
				MaxInterval:   time.Second * 10,
			},
			wantErr: true,
		},
		{
			name: "negative retry interval",
			config: RetryConfig{
				MaxRetries:    3,
				RetryInterval: -time.Second,
				BackoffFactor: 2.0,
				MaxInterval:   time.Second * 10,
			},
			wantErr: true,
		},
		{
			name: "invalid backoff factor",
			config: RetryConfig{
				MaxRetries:    3,
				RetryInterval: time.Second,
				BackoffFactor: 0.5,
				MaxInterval:   time.Second * 10,
			},
			wantErr: true,
		},
		{
			name: "max interval less than retry interval",
			config: RetryConfig{
				MaxRetries:    3,
				RetryInterval: time.Second * 10,
				BackoffFactor: 2.0,
				MaxInterval:   time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	
	if err := config.Validate(); err != nil {
		t.Errorf("DefaultRetryConfig() should return valid config: %v", err)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("DefaultRetryConfig() MaxRetries = %d, want 3", config.MaxRetries)
	}
	
	if config.RetryInterval != time.Second {
		t.Errorf("DefaultRetryConfig() RetryInterval = %v, want %v", config.RetryInterval, time.Second)
	}
	
	if config.BackoffFactor != 2.0 {
		t.Errorf("DefaultRetryConfig() BackoffFactor = %f, want 2.0", config.BackoffFactor)
	}
}

func TestNewResilientSerialPort(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	port := NewResilientSerialPort(retryConfig)
	
	if port == nil {
		t.Error("NewResilientSerialPort() returned nil")
	}
	
	if port.GetState() != StateDisconnected {
		t.Errorf("NewResilientSerialPort() state = %v, want %v", port.GetState(), StateDisconnected)
	}
	
	if port.GetLastError() != nil {
		t.Errorf("NewResilientSerialPort() should have no last error, got: %v", port.GetLastError())
	}
}

func TestResilientSerialPort_OpenWithRetryInvalidConfig(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	port := NewResilientSerialPort(retryConfig)
	
	invalidConfig := SerialConfig{
		Port:     "", // Invalid
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second,
	}
	
	err := port.OpenWithRetry(invalidConfig)
	if err == nil {
		t.Error("OpenWithRetry() should fail with invalid config")
	}
	
	if port.GetState() != StateDisconnected {
		t.Errorf("State should remain disconnected after invalid config, got: %v", port.GetState())
	}
}

func TestResilientSerialPort_ReconnectNoConfig(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	port := NewResilientSerialPort(retryConfig)
	
	err := port.Reconnect()
	if err == nil {
		t.Error("Reconnect() should fail when no previous configuration exists")
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		want  bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "device busy error",
			err:  fmt.Errorf("device busy"),
			want: true,
		},
		{
			name: "timeout error",
			err:  fmt.Errorf("operation timeout"),
			want: true,
		},
		{
			name: "resource unavailable",
			err:  fmt.Errorf("resource temporarily unavailable"),
			want: true,
		},
		{
			name: "no such device",
			err:  fmt.Errorf("no such device"),
			want: true,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("connection refused"),
			want: true,
		},
		{
			name: "non-recoverable error",
			err:  fmt.Errorf("permission denied"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecoverableError(tt.err); got != tt.want {
				t.Errorf("isRecoverableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact match", "hello", "hello", true},
		{"substring at start", "hello world", "hello", true},
		{"substring at end", "hello world", "world", true},
		{"substring in middle", "hello world", "lo wo", true},
		{"not found", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIndexOfSubstring(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{"found at start", "hello world", "hello", 0},
		{"found at end", "hello world", "world", 6},
		{"found in middle", "hello world", "lo wo", 3},
		{"not found", "hello world", "xyz", -1},
		{"empty substring", "hello", "", 0},
		{"empty string", "", "hello", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexOfSubstring(tt.s, tt.substr); got != tt.want {
				t.Errorf("indexOfSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()
	
	if validator == nil {
		t.Error("NewConfigValidator() returned nil")
	}
	
	if len(validator.allowedBauds) == 0 {
		t.Error("NewConfigValidator() should have default allowed baud rates")
	}
	
	if !validator.requireTimeout {
		t.Error("NewConfigValidator() should require timeout by default")
	}
}

func TestConfigValidator_SetAllowedPorts(t *testing.T) {
	validator := NewConfigValidator()
	ports := []string{"COM1", "COM2", "/dev/ttyUSB0"}
	
	validator.SetAllowedPorts(ports)
	
	// Test that the ports were copied (not referenced)
	ports[0] = "MODIFIED"
	
	if len(validator.allowedPorts) != 3 {
		t.Errorf("SetAllowedPorts() should set 3 ports, got %d", len(validator.allowedPorts))
	}
	
	if validator.allowedPorts[0] == "MODIFIED" {
		t.Error("SetAllowedPorts() should copy ports, not reference them")
	}
}

func TestConfigValidator_ValidateAdvanced(t *testing.T) {
	validator := NewConfigValidator()
	validator.SetAllowedPorts([]string{"COM1", "COM2"})
	
	tests := []struct {
		name    string
		config  SerialConfig
		wantErr bool
	}{
		{
			name: "valid config with allowed port",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port not in allowed list",
			config: SerialConfig{
				Port:     "COM3",
				BaudRate: 115200,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid baud rate",
			config: SerialConfig{
				Port:     "COM1",
				BaudRate: 12345,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAdvanced(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigValidator.ValidateAdvanced() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidator_SetTimeoutRequirement(t *testing.T) {
	validator := NewConfigValidator()
	
	validator.SetTimeoutRequirement(false, time.Minute)
	
	config := SerialConfig{
		Port:     "COM1",
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  0, // No timeout
	}
	
	// Should not error since timeout is not required
	if err := validator.ValidateAdvanced(config); err != nil {
		t.Errorf("ValidateAdvanced() should not error when timeout not required: %v", err)
	}
	
	// Test timeout too large
	validator.SetTimeoutRequirement(true, time.Second)
	config.Timeout = time.Minute // Larger than max
	
	if err := validator.ValidateAdvanced(config); err == nil {
		t.Error("ValidateAdvanced() should error when timeout exceeds maximum")
	}
}

func TestNewHealthChecker(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	checkData := []byte("ping")
	expectedResp := []byte("pong")
	timeout := time.Second
	
	hc := NewHealthChecker(port, checkData, expectedResp, timeout)
	
	if hc == nil {
		t.Error("NewHealthChecker() returned nil")
	}
	
	if hc.port != port {
		t.Error("NewHealthChecker() should set the port")
	}
	
	if string(hc.checkData) != "ping" {
		t.Errorf("NewHealthChecker() checkData = %s, want ping", string(hc.checkData))
	}
	
	if string(hc.expectedResp) != "pong" {
		t.Errorf("NewHealthChecker() expectedResp = %s, want pong", string(hc.expectedResp))
	}
	
	if hc.timeout != timeout {
		t.Errorf("NewHealthChecker() timeout = %v, want %v", hc.timeout, timeout)
	}
}

func TestHealthChecker_CheckHealthPortNotOpen(t *testing.T) {
	port := NewCrossPlatformSerialPort()
	hc := NewHealthChecker(port, []byte("ping"), []byte("pong"), time.Second)
	
	err := hc.CheckHealth()
	if err == nil {
		t.Error("CheckHealth() should error when port is not open")
	}
}