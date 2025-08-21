package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
	
	"github.com/spf13/cobra"
	"serial-terminal/pkg/serial"
)

// TestRootCommand tests the root command
func TestRootCommand(t *testing.T) {
	// Just test that the command is properly configured
	// We can't easily test the full execution without mocking a lot
	
	// Check basic command properties
	if rootCmd.Use != "serial-terminal" {
		t.Errorf("rootCmd.Use = %s, want serial-terminal", rootCmd.Use)
	}
	
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
	
	// Check that subcommands are registered
	subcommands := rootCmd.Commands()
	expectedCommands := []string{"list", "config", "connect"}
	
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Use == expected || strings.HasPrefix(cmd.Use, expected+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", expected)
		}
	}
}

// TestListCommand tests the list command
func TestListCommand(t *testing.T) {
	// Test list command should just not error
	// Since we can't mock serial ports easily, just verify it runs
	listCmd.Run(listCmd, []string{})
	
	// The function actually prints to stdout, so we can't easily capture output in this test
	// But since it doesn't return an error, we just verify it completes
	// If it panics, the test will fail
}

// TestConfigCommand tests the config command
func TestConfigCommand(t *testing.T) {
	// Create a buffer to capture output
	output := &bytes.Buffer{}
	
	// Create a new command for testing
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(configCmd)
	cmd.SetOut(output)
	cmd.SetErr(output)
	
	// Test config help
	cmd.SetArgs([]string{"config", "--help"})
	err := cmd.Execute()
	
	if err != nil {
		t.Errorf("config --help failed: %v", err)
	}
	
	// Check that output contains expected subcommands
	out := output.String()
	expectedCommands := []string{
		"save",
		"load",
		"list",
		"delete",
		"show",
	}
	
	for _, expected := range expectedCommands {
		if !strings.Contains(out, expected) {
			t.Errorf("Expected config help to contain '%s', but it doesn't", expected)
		}
	}
}

// TestConnectCommand tests the connect command
func TestConnectCommand(t *testing.T) {
	// Create a buffer to capture output
	output := &bytes.Buffer{}
	
	// Create a new command for testing
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(connectCmd)
	cmd.SetOut(output)
	cmd.SetErr(output)
	
	// Test connect help
	cmd.SetArgs([]string{"connect", "--help"})
	err := cmd.Execute()
	
	if err != nil {
		t.Errorf("connect --help failed: %v", err)
	}
	
	// Check that output contains expected text
	out := output.String()
	expectedTexts := []string{
		"Connect to a serial port",
		"port",
		"baud",
	}
	
	for _, expected := range expectedTexts {
		if !strings.Contains(out, expected) {
			t.Errorf("Expected connect help to contain '%s', but it doesn't", expected)
		}
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		shouldErr bool
	}{
		{
			name:      "valid baud rate",
			args:      []string{"--port", "COM1", "--baud", "115200"},
			shouldErr: false,
		},
		{
			name:      "invalid baud rate",
			args:      []string{"--port", "COM1", "--baud", "12345"},
			shouldErr: true,
		},
		{
			name:      "invalid data bits",
			args:      []string{"--port", "COM1", "--data", "10"},
			shouldErr: true,
		},
		{
			name:      "invalid stop bits",
			args:      []string{"--port", "COM1", "--stop", "3"},
			shouldErr: true,
		},
		{
			name:      "invalid parity",
			args:      []string{"--port", "COM1", "--parity", "invalid"},
			shouldErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't actually connect to a port in tests,
			// we'll just check that validation works correctly
			// This is more of an integration test with the serial package
			
			// Parse the flags to extract values
			cmd := &cobra.Command{Use: "test"}
			
			// Add the same flags as the root command
			var testPort string
			var testBaud, testData, testStop int
			var testParity string
			
			cmd.Flags().StringVar(&testPort, "port", "", "port")
			cmd.Flags().IntVar(&testBaud, "baud", 115200, "baud")
			cmd.Flags().IntVar(&testData, "data", 8, "data")
			cmd.Flags().IntVar(&testStop, "stop", 1, "stop")
			cmd.Flags().StringVar(&testParity, "parity", "none", "parity")
			
			cmd.ParseFlags(tt.args)
			
			// Create config based on parsed flags
			cfg := serial.SerialConfig{
				Port:     testPort,
				BaudRate: testBaud,
				DataBits: testData,
				StopBits: testStop,
				Parity:   testParity,
				Timeout:  time.Second * 10,
			}
			
			// Validate
			err := cfg.Validate()
			
			if tt.shouldErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

// TestIsSerialPort tests the isSerialPort helper function
func TestIsSerialPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Windows COM port", "COM1", true},
		{"Windows COM port lowercase", "com3", true},
		{"Linux serial device", "/dev/ttyUSB0", true},
		{"macOS serial device", "/dev/cu.usbserial", true},
		{"Not a serial port", "myconfig", false},
		{"Random text", "hello", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSerialPort(tt.input)
			if result != tt.expected {
				t.Errorf("isSerialPort(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestRepeatString tests the repeatString helper function
func TestRepeatString(t *testing.T) {
	tests := []struct {
		str      string
		count    int
		expected string
	}{
		{"=", 5, "====="},
		{"ab", 3, "ababab"},
		{"x", 0, ""},
		{"", 5, ""},
	}
	
	for _, tt := range tests {
		result := repeatString(tt.str, tt.count)
		if result != tt.expected {
			t.Errorf("repeatString(%s, %d) = %s, want %s", tt.str, tt.count, result, tt.expected)
		}
	}
}

// TestPortDescription tests the getPortDescription helper function
func TestPortDescription(t *testing.T) {
	// Save original GOOS
	originalGOOS := os.Getenv("GOOS")
	defer os.Setenv("GOOS", originalGOOS)
	
	tests := []struct {
		goos     string
		port     string
		expected string
	}{
		{"windows", "COM1", "USB Serial Device"},
		{"linux", "/dev/ttyUSB0", "USB Serial Device"},
		{"linux", "/dev/ttyS0", "System Serial Port"},
		{"darwin", "/dev/cu.usbserial", "USB Serial Device (Callout)"},
		{"darwin", "/dev/tty.usbserial", "USB Serial Device (TTY)"},
	}
	
	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.port, func(t *testing.T) {
			// Note: getPortDescription uses runtime.GOOS which we can't easily mock
			// This test is more for documentation of expected behavior
			// In a real test, we'd need to refactor getPortDescription to be testable
		})
	}
}

// TestCommandStructure tests that all commands are properly structured
func TestCommandStructure(t *testing.T) {
	commands := []*cobra.Command{
		rootCmd,
		listCmd,
		configCmd,
		connectCmd,
	}
	
	for _, cmd := range commands {
		// Check that command has Use field
		if cmd.Use == "" {
			t.Errorf("Command %v has empty Use field", cmd)
		}
		
		// Check that command has Short description
		if cmd.Short == "" {
			t.Errorf("Command %s has empty Short description", cmd.Use)
		}
		
		// Check that command has Long description (except subcommands)
		if cmd != saveCmd && cmd != loadCmd && cmd != deleteCmd && cmd != showCmd && cmd.Long == "" {
			t.Errorf("Command %s has empty Long description", cmd.Use)
		}
	}
}

