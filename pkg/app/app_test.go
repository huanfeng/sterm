package app

import (
	"testing"
	"time"
	
	"serial-terminal/pkg/serial"
	"serial-terminal/pkg/terminal"
)

func TestSessionManagement(t *testing.T) {
	// Create a new session
	config := serial.DefaultConfig()
	config.Port = "COM1"
	
	session := NewSession("test_session", config)
	
	// Check initial state
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	
	if session.Name != "test_session" {
		t.Errorf("Session name = %s, want test_session", session.Name)
	}
	
	if !session.IsActive {
		t.Error("New session should be active")
	}
	
	if session.BytesSent != 0 || session.BytesRecv != 0 {
		t.Error("New session should have zero bytes")
	}
	
	// Update stats
	session.UpdateStats(100, 200)
	
	sent, recv := session.GetStats()
	if sent != 100 || recv != 200 {
		t.Errorf("Session stats = (%d, %d), want (100, 200)", sent, recv)
	}
	
	// End session
	session.End()
	
	if session.IsActive {
		t.Error("Ended session should not be active")
	}
	
	if session.EndTime == nil {
		t.Error("Ended session should have end time")
	}
}

func TestAppConfig(t *testing.T) {
	// Test default config
	config := DefaultAppConfig()
	
	if config.TerminalWidth != 80 {
		t.Errorf("Default terminal width = %d, want 80", config.TerminalWidth)
	}
	
	if config.TerminalHeight != 24 {
		t.Errorf("Default terminal height = %d, want 24", config.TerminalHeight)
	}
	
	if config.HistorySize != 10*1024*1024 {
		t.Errorf("Default history size = %d, want %d", config.HistorySize, 10*1024*1024)
	}
	
	if !config.EnableMouse {
		t.Error("Mouse should be enabled by default")
	}
	
	if !config.EnableShortcuts {
		t.Error("Shortcuts should be enabled by default")
	}
}

func TestApplicationCreation(t *testing.T) {
	// Create config with test serial port
	config := DefaultAppConfig()
	config.SerialConfig.Port = "COM1"
	config.SerialConfig.BaudRate = 9600
	
	// Create application
	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	
	// Check initial state
	if app.IsRunning() {
		t.Error("New application should not be running")
	}
	
	if app.IsPaused() {
		t.Error("New application should not be paused")
	}
	
	// Clean up
	app.Stop()
}

func TestApplicationLifecycle(t *testing.T) {
	// Create config with test serial port
	config := DefaultAppConfig()
	config.SerialConfig.Port = "COM1"
	config.SerialConfig.BaudRate = 9600
	
	// Create application
	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	
	// Test start/stop cycle
	// Note: This will fail if there's no actual serial port
	// In a real test, we'd mock the serial port
	
	// Stop should work even if not started
	err = app.Stop()
	if err != nil {
		t.Errorf("Stop on non-running app failed: %v", err)
	}
}

func TestApplicationPauseResume(t *testing.T) {
	// Create config
	config := DefaultAppConfig()
	config.SerialConfig.Port = "COM1"
	
	// Create application
	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Stop()
	
	// Should not be able to pause when not running
	err = app.Pause()
	if err == nil {
		t.Error("Pause should fail when app is not running")
	}
	
	// Should not be able to resume when not running
	err = app.Resume()
	if err == nil {
		t.Error("Resume should fail when app is not running")
	}
}

func TestColorConversion(t *testing.T) {
	// Test color conversion
	// This is a simple test to ensure the conversion function doesn't panic
	colors := []terminal.Color{
		terminal.ColorBlack,
		terminal.ColorRed,
		terminal.ColorGreen,
		terminal.ColorYellow,
		terminal.ColorBlue,
		terminal.ColorMagenta,
		terminal.ColorCyan,
		terminal.ColorWhite,
	}
	
	for _, color := range colors {
		tcellColor := convertColor(color)
		if tcellColor < 0 {
			t.Errorf("Invalid tcell color for terminal color %v", color)
		}
	}
}

func TestSessionIDGeneration(t *testing.T) {
	// Generate multiple IDs
	id1 := generateSessionID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := generateSessionID()
	
	// Check they are unique
	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}
	
	// Check they are not empty
	if id1 == "" || id2 == "" {
		t.Error("Session IDs should not be empty")
	}
}

func TestRunnerCreation(t *testing.T) {
	// Create serial config
	serialConfig := serial.DefaultConfig()
	serialConfig.Port = "COM1"
	
	// Create runner
	runner, err := NewRunner(serialConfig)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	
	// Check config
	if runner.config.SerialConfig.Port != "COM1" {
		t.Errorf("Runner serial port = %s, want COM1", runner.config.SerialConfig.Port)
	}
}

