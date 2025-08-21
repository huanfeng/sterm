// Package serial provides serial port communication functionality
package serial

import (
	"fmt"
	"time"
	
	"go.bug.st/serial"
)

// SerialConfig defines the configuration for serial port communication
type SerialConfig struct {
	Port     string        `json:"port"`
	BaudRate int           `json:"baud_rate"`
	DataBits int           `json:"data_bits"`
	StopBits int           `json:"stop_bits"`
	Parity   string        `json:"parity"`
	Timeout  time.Duration `json:"timeout"`
}

// Validate checks if the serial configuration is valid
func (c SerialConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}
	
	validBaudRates := []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600}
	validBaud := false
	for _, rate := range validBaudRates {
		if c.BaudRate == rate {
			validBaud = true
			break
		}
	}
	if !validBaud {
		return fmt.Errorf("invalid baud rate: %d", c.BaudRate)
	}
	
	if c.DataBits < 5 || c.DataBits > 8 {
		return fmt.Errorf("data bits must be between 5 and 8, got: %d", c.DataBits)
	}
	
	if c.StopBits < 1 || c.StopBits > 2 {
		return fmt.Errorf("stop bits must be 1 or 2, got: %d", c.StopBits)
	}
	
	validParity := []string{"none", "odd", "even", "mark", "space"}
	validParityFound := false
	for _, p := range validParity {
		if c.Parity == p {
			validParityFound = true
			break
		}
	}
	if !validParityFound {
		return fmt.Errorf("invalid parity: %s", c.Parity)
	}
	
	if c.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	
	return nil
}

// DefaultConfig returns a default serial configuration
func DefaultConfig() SerialConfig {
	return SerialConfig{
		Port:     "COM1", // Default port for Windows, will be platform-specific in implementation
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second * 5,
	}
}

// SerialPort interface defines the contract for serial port operations
type SerialPort interface {
	Open(config SerialConfig) error
	Close() error
	Read(buffer []byte) (int, error)
	Write(data []byte) (int, error)
	IsOpen() bool
	GetConfig() SerialConfig
	SetReadTimeout(timeout time.Duration) error
	GetAvailablePorts() ([]string, error)
}

// CrossPlatformSerialPort implements SerialPort interface using go.bug.st/serial
type CrossPlatformSerialPort struct {
	port   serial.Port
	config SerialConfig
	isOpen bool
}

// NewCrossPlatformSerialPort creates a new cross-platform serial port instance
func NewCrossPlatformSerialPort() *CrossPlatformSerialPort {
	return &CrossPlatformSerialPort{
		isOpen: false,
	}
}

// Open opens the serial port with the given configuration
func (sp *CrossPlatformSerialPort) Open(config SerialConfig) error {
	if sp.isOpen {
		return fmt.Errorf("serial port is already open")
	}
	
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Convert our config to go.bug.st/serial config
	mode := &serial.Mode{
		BaudRate: config.BaudRate,
		DataBits: config.DataBits,
		StopBits: convertStopBits(config.StopBits),
		Parity:   convertParity(config.Parity),
	}
	
	port, err := serial.Open(config.Port, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %w", config.Port, err)
	}
	
	// Set read timeout if specified
	if config.Timeout > 0 {
		if err := port.SetReadTimeout(config.Timeout); err != nil {
			port.Close()
			return fmt.Errorf("failed to set read timeout: %w", err)
		}
	}
	
	sp.port = port
	sp.config = config
	sp.isOpen = true
	
	return nil
}

// Close closes the serial port
func (sp *CrossPlatformSerialPort) Close() error {
	if !sp.isOpen {
		return fmt.Errorf("serial port is not open")
	}
	
	err := sp.port.Close()
	sp.port = nil
	sp.isOpen = false
	
	if err != nil {
		return fmt.Errorf("failed to close serial port: %w", err)
	}
	
	return nil
}

// Read reads data from the serial port
func (sp *CrossPlatformSerialPort) Read(buffer []byte) (int, error) {
	if !sp.isOpen {
		return 0, fmt.Errorf("serial port is not open")
	}
	
	n, err := sp.port.Read(buffer)
	if err != nil {
		return n, fmt.Errorf("failed to read from serial port: %w", err)
	}
	
	return n, nil
}

// Write writes data to the serial port
func (sp *CrossPlatformSerialPort) Write(data []byte) (int, error) {
	if !sp.isOpen {
		return 0, fmt.Errorf("serial port is not open")
	}
	
	n, err := sp.port.Write(data)
	if err != nil {
		return n, fmt.Errorf("failed to write to serial port: %w", err)
	}
	
	return n, nil
}

// IsOpen returns true if the serial port is open
func (sp *CrossPlatformSerialPort) IsOpen() bool {
	return sp.isOpen
}

// GetConfig returns the current serial port configuration
func (sp *CrossPlatformSerialPort) GetConfig() SerialConfig {
	return sp.config
}

// SetReadTimeout sets the read timeout for the serial port
func (sp *CrossPlatformSerialPort) SetReadTimeout(timeout time.Duration) error {
	if !sp.isOpen {
		return fmt.Errorf("serial port is not open")
	}
	
	if err := sp.port.SetReadTimeout(timeout); err != nil {
		return fmt.Errorf("failed to set read timeout: %w", err)
	}
	
	sp.config.Timeout = timeout
	return nil
}

// GetAvailablePorts returns a list of available serial ports
func (sp *CrossPlatformSerialPort) GetAvailablePorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to get available ports: %w", err)
	}
	
	return ports, nil
}

// convertStopBits converts our stop bits format to go.bug.st/serial format
func convertStopBits(stopBits int) serial.StopBits {
	switch stopBits {
	case 1:
		return serial.OneStopBit
	case 2:
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

// convertParity converts our parity format to go.bug.st/serial format
func convertParity(parity string) serial.Parity {
	switch parity {
	case "none":
		return serial.NoParity
	case "odd":
		return serial.OddParity
	case "even":
		return serial.EvenParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

// PortInfo contains information about a serial port
type PortInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	VID         string `json:"vid,omitempty"`
	PID         string `json:"pid,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
}

// GetDetailedPortsList returns detailed information about available serial ports
func GetDetailedPortsList() ([]PortInfo, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to get ports list: %w", err)
	}
	
	var portInfos []PortInfo
	for _, portName := range ports {
		portInfo := PortInfo{
			Name: portName,
		}
		
		// Note: Additional port details would require platform-specific implementation
		// For now, we just provide the port name
		
		portInfos = append(portInfos, portInfo)
	}
	
	return portInfos, nil
}

// IsPortAvailable checks if a specific port is available
func IsPortAvailable(portName string) bool {
	ports, err := serial.GetPortsList()
	if err != nil {
		return false
	}
	
	for _, port := range ports {
		if port == portName {
			return true
		}
	}
	
	return false
}

// SerialError represents a serial port specific error
type SerialError struct {
	Operation string
	Port      string
	Cause     error
}

// Error implements the error interface
func (e *SerialError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("serial %s operation failed on port %s: %v", e.Operation, e.Port, e.Cause)
	}
	return fmt.Sprintf("serial %s operation failed on port %s", e.Operation, e.Port)
}

// NewSerialError creates a new serial error
func NewSerialError(operation, port string, cause error) *SerialError {
	return &SerialError{
		Operation: operation,
		Port:      port,
		Cause:     cause,
	}
}

// ConnectionState represents the state of a serial connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateError
)

// String returns the string representation of ConnectionState
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// RetryConfig defines configuration for connection retry logic
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`
	BackoffFactor float64       `json:"backoff_factor"`
	MaxInterval   time.Duration `json:"max_interval"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		RetryInterval: time.Second,
		BackoffFactor: 2.0,
		MaxInterval:   time.Second * 10,
	}
}

// Validate checks if the retry configuration is valid
func (r RetryConfig) Validate() error {
	if r.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	
	if r.RetryInterval < 0 {
		return fmt.Errorf("retry interval cannot be negative")
	}
	
	if r.BackoffFactor < 1.0 {
		return fmt.Errorf("backoff factor must be >= 1.0")
	}
	
	if r.MaxInterval < r.RetryInterval {
		return fmt.Errorf("max interval cannot be less than retry interval")
	}
	
	return nil
}

// ResilientSerialPort extends CrossPlatformSerialPort with retry and recovery capabilities
type ResilientSerialPort struct {
	*CrossPlatformSerialPort
	retryConfig RetryConfig
	lastError   error
	state       ConnectionState
}

// NewResilientSerialPort creates a new resilient serial port with retry capabilities
func NewResilientSerialPort(retryConfig RetryConfig) *ResilientSerialPort {
	return &ResilientSerialPort{
		CrossPlatformSerialPort: NewCrossPlatformSerialPort(),
		retryConfig:             retryConfig,
		state:                   StateDisconnected,
	}
}

// OpenWithRetry opens the serial port with retry logic
func (rsp *ResilientSerialPort) OpenWithRetry(config SerialConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	if err := rsp.retryConfig.Validate(); err != nil {
		return fmt.Errorf("invalid retry configuration: %w", err)
	}
	
	rsp.state = StateConnecting
	
	var lastErr error
	interval := rsp.retryConfig.RetryInterval
	
	for attempt := 0; attempt <= rsp.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(interval)
			// Apply exponential backoff
			interval = time.Duration(float64(interval) * rsp.retryConfig.BackoffFactor)
			if interval > rsp.retryConfig.MaxInterval {
				interval = rsp.retryConfig.MaxInterval
			}
		}
		
		err := rsp.CrossPlatformSerialPort.Open(config)
		if err == nil {
			rsp.state = StateConnected
			rsp.lastError = nil
			return nil
		}
		
		lastErr = err
		
		// Check if this is a recoverable error
		if !isRecoverableError(err) {
			break
		}
	}
	
	rsp.state = StateError
	rsp.lastError = lastErr
	return fmt.Errorf("failed to open serial port after %d attempts: %w", rsp.retryConfig.MaxRetries+1, lastErr)
}

// Close closes the serial port and updates state
func (rsp *ResilientSerialPort) Close() error {
	err := rsp.CrossPlatformSerialPort.Close()
	if err != nil {
		rsp.state = StateError
		rsp.lastError = err
		return err
	}
	
	rsp.state = StateDisconnected
	rsp.lastError = nil
	return nil
}

// GetState returns the current connection state
func (rsp *ResilientSerialPort) GetState() ConnectionState {
	return rsp.state
}

// GetLastError returns the last error that occurred
func (rsp *ResilientSerialPort) GetLastError() error {
	return rsp.lastError
}

// Reconnect attempts to reconnect using the last known configuration
func (rsp *ResilientSerialPort) Reconnect() error {
	if rsp.config.Port == "" {
		return fmt.Errorf("no previous configuration available for reconnection")
	}
	
	// Close existing connection if open
	if rsp.IsOpen() {
		if err := rsp.Close(); err != nil {
			return fmt.Errorf("failed to close existing connection: %w", err)
		}
	}
	
	return rsp.OpenWithRetry(rsp.config)
}

// isRecoverableError determines if an error is recoverable and retry should be attempted
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}
	
	errorStr := err.Error()
	
	// Common recoverable error patterns
	recoverablePatterns := []string{
		"device busy",
		"resource temporarily unavailable",
		"timeout",
		"connection refused",
		"no such device", // Device might be reconnected
	}
	
	for _, pattern := range recoverablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring in a string
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// ConfigValidator provides advanced validation for serial configurations
type ConfigValidator struct {
	allowedPorts    []string
	allowedBauds    []int
	requireTimeout  bool
	maxTimeout      time.Duration
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		allowedBauds:   []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600},
		requireTimeout: true,
		maxTimeout:     time.Minute * 5,
	}
}

// SetAllowedPorts sets the list of allowed ports
func (cv *ConfigValidator) SetAllowedPorts(ports []string) {
	cv.allowedPorts = make([]string, len(ports))
	copy(cv.allowedPorts, ports)
}

// SetAllowedBaudRates sets the list of allowed baud rates
func (cv *ConfigValidator) SetAllowedBaudRates(bauds []int) {
	cv.allowedBauds = make([]int, len(bauds))
	copy(cv.allowedBauds, bauds)
}

// SetTimeoutRequirement sets whether timeout is required and the maximum allowed
func (cv *ConfigValidator) SetTimeoutRequirement(required bool, maxTimeout time.Duration) {
	cv.requireTimeout = required
	cv.maxTimeout = maxTimeout
}

// ValidateAdvanced performs advanced validation on a serial configuration
func (cv *ConfigValidator) ValidateAdvanced(config SerialConfig) error {
	// Basic validation first
	if err := config.Validate(); err != nil {
		return err
	}
	
	// Check allowed ports if specified
	if len(cv.allowedPorts) > 0 {
		allowed := false
		for _, port := range cv.allowedPorts {
			if config.Port == port {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("port %s is not in the allowed ports list", config.Port)
		}
	}
	
	// Check allowed baud rates
	if len(cv.allowedBauds) > 0 {
		allowed := false
		for _, baud := range cv.allowedBauds {
			if config.BaudRate == baud {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("baud rate %d is not in the allowed baud rates list", config.BaudRate)
		}
	}
	
	// Check timeout requirements
	if cv.requireTimeout && config.Timeout <= 0 {
		return fmt.Errorf("timeout is required but not set")
	}
	
	if config.Timeout > cv.maxTimeout {
		return fmt.Errorf("timeout %v exceeds maximum allowed timeout %v", config.Timeout, cv.maxTimeout)
	}
	
	return nil
}

// NewSerialPort creates a new serial port instance (convenience function)
func NewSerialPort() SerialPort {
	return NewCrossPlatformSerialPort()
}

// ListPorts returns a list of available serial ports on the system (global function)
func ListPorts() ([]string, error) {
	return serial.GetPortsList()
}

// HealthChecker provides health checking capabilities for serial connections
type HealthChecker struct {
	port         SerialPort
	checkData    []byte
	expectedResp []byte
	timeout      time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(port SerialPort, checkData, expectedResp []byte, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		port:         port,
		checkData:    checkData,
		expectedResp: expectedResp,
		timeout:      timeout,
	}
}

// CheckHealth performs a health check on the serial connection
func (hc *HealthChecker) CheckHealth() error {
	if !hc.port.IsOpen() {
		return fmt.Errorf("serial port is not open")
	}
	
	// Send check data
	_, err := hc.port.Write(hc.checkData)
	if err != nil {
		return fmt.Errorf("failed to send health check data: %w", err)
	}
	
	// Read response with timeout
	buffer := make([]byte, len(hc.expectedResp))
	
	// Set a temporary timeout for health check
	originalTimeout := hc.port.GetConfig().Timeout
	if err := hc.port.SetReadTimeout(hc.timeout); err != nil {
		return fmt.Errorf("failed to set health check timeout: %w", err)
	}
	
	// Restore original timeout after health check
	defer func() {
		hc.port.SetReadTimeout(originalTimeout)
	}()
	
	n, err := hc.port.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}
	
	if n != len(hc.expectedResp) {
		return fmt.Errorf("health check response length mismatch: expected %d, got %d", len(hc.expectedResp), n)
	}
	
	for i, b := range hc.expectedResp {
		if buffer[i] != b {
			return fmt.Errorf("health check response mismatch at byte %d: expected %02x, got %02x", i, b, buffer[i])
		}
	}
	
	return nil
}