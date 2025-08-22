//go:build integration
// +build integration

package serial

import (
	"testing"
	"time"
)

// TestGetAvailablePorts tests the actual port enumeration
func TestGetAvailablePorts(t *testing.T) {
	port := NewCrossPlatformSerialPort()

	ports, err := port.GetAvailablePorts()
	if err != nil {
		t.Errorf("GetAvailablePorts() failed: %v", err)
	}

	// We can't guarantee any specific ports exist, but the function should not error
	t.Logf("Available ports: %v", ports)
}

// TestGetDetailedPortsList tests the detailed port information
func TestGetDetailedPortsList(t *testing.T) {
	portInfos, err := GetDetailedPortsList()
	if err != nil {
		t.Errorf("GetDetailedPortsList() failed: %v", err)
	}

	// Log the port information for manual verification
	for _, portInfo := range portInfos {
		t.Logf("Port: %s, Description: %s, VID: %s, PID: %s, Serial: %s",
			portInfo.Name, portInfo.Description, portInfo.VID, portInfo.PID, portInfo.SerialNumber)
	}
}

// TestIsPortAvailable tests port availability checking
func TestIsPortAvailable(t *testing.T) {
	// Test with a port that likely doesn't exist
	if IsPortAvailable("COM999") {
		t.Error("COM999 should not be available")
	}

	// Test with common Windows ports (may or may not exist)
	commonPorts := []string{"COM1", "COM2", "COM3", "COM4"}
	for _, port := range commonPorts {
		available := IsPortAvailable(port)
		t.Logf("Port %s available: %v", port, available)
	}
}

// TestSerialPortOpenNonExistent tests opening a non-existent port
func TestSerialPortOpenNonExistent(t *testing.T) {
	port := NewCrossPlatformSerialPort()

	config := SerialConfig{
		Port:     "COM999", // Likely non-existent port
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second,
	}

	err := port.Open(config)
	if err == nil {
		// If it somehow opened, close it
		port.Close()
		t.Log("Warning: COM999 actually exists and was opened")
	} else {
		t.Logf("Expected error opening non-existent port: %v", err)
	}
}
