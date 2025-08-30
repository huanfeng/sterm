// Package app provides the main application controller
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sterm/pkg/config"
	"sterm/pkg/history"
	"sterm/pkg/menu"
	"sterm/pkg/serial"
	"sterm/pkg/terminal"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Application represents the main application controller
type Application struct {
	// Core components
	serialPort     serial.SerialPort
	terminal       *terminal.TerminalEmulator
	configMgr      config.ConfigManager
	historyMgr     history.HistoryManager
	inputProcessor *terminal.InputProcessor // Keep single instance for state

	// UI components
	screen     tcell.Screen
	shortcuts  *terminal.ShortcutManager
	mainMenu   *menu.Menu
	overlayMgr *menu.OverlayManager

	// Session management
	session *Session

	// Control
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	updateNotify chan struct{} // Channel to notify UI updates
	pauseChan    chan bool     // Channel to control pause state

	// State
	isRunning     bool
	isPaused      bool
	localEcho     bool      // Whether to echo typed characters locally
	lineWrap      bool      // Whether to wrap long lines
	statusMessage string    // Temporary status message
	statusTime    time.Time // When status message was set

	// Cached status bar strings
	cachedStatusLeft  string
	cachedStatusRight string
	cachedBytesRecv   int64
	cachedBytesSent   int64

	// Configuration
	config AppConfig

	// Debug
	debugLog  *os.File
	debugMode bool
}

// AppConfig contains application configuration
type AppConfig struct {
	SerialConfig            serial.SerialConfig
	TerminalWidth           int
	TerminalHeight          int
	HistorySize             int
	EnableMouse             bool
	EnableShortcuts         bool
	SaveHistory             bool
	HistoryFormat           history.FileFormat
	SendWindowSizeOnConnect bool   // Send window size when connecting
	SendWindowSizeOnResize  bool   // Send window size when resizing
	TerminalType            string // Terminal type to report (vt100, xterm, etc.)
	Version                 string // Application version
	DebugMode               bool   // Enable debug logging
}

// DefaultAppConfig returns default application configuration
func DefaultAppConfig() AppConfig {
	return AppConfig{
		Version:                 "1.0.0",
		SerialConfig:            serial.DefaultConfig(),
		TerminalWidth:           80,
		TerminalHeight:          24,
		HistorySize:             10 * 1024 * 1024, // 10MB
		EnableMouse:             true,
		EnableShortcuts:         true,
		SaveHistory:             true,
		HistoryFormat:           history.FormatTimestamped,
		SendWindowSizeOnConnect: false,   // Disabled by default - can cause issues with some devices
		SendWindowSizeOnResize:  false,   // Disabled by default
		TerminalType:            "xterm", // Default to xterm for better compatibility
	}
}

// Session represents an active serial terminal session
type Session struct {
	ID        string
	Name      string
	Config    serial.SerialConfig
	StartTime time.Time
	EndTime   *time.Time
	BytesSent int64
	BytesRecv int64
	IsActive  bool
	mu        sync.RWMutex
}

// NewSession creates a new session
func NewSession(name string, config serial.SerialConfig) *Session {
	return &Session{
		ID:        generateSessionID(),
		Name:      name,
		Config:    config,
		StartTime: time.Now(),
		IsActive:  true,
	}
}

// End marks the session as ended
func (s *Session) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.EndTime = &now
	s.IsActive = false
}

// UpdateStats updates session statistics
func (s *Session) UpdateStats(bytesSent, bytesRecv int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.BytesSent += bytesSent
	s.BytesRecv += bytesRecv
}

// GetStats returns session statistics
func (s *Session) GetStats() (bytesSent, bytesRecv int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.BytesSent, s.BytesRecv
}

// logDebug writes debug message to log file
func (app *Application) logDebug(format string, args ...interface{}) {
	if app.debugLog != nil {
		msg := fmt.Sprintf(format, args...)
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Fprintf(app.debugLog, "[%s] %s\n", timestamp, msg)
		_ = app.debugLog.Sync() // Ensure it's written immediately
	}
}

// Debugf implements the terminal.Logger interface
func (app *Application) Debugf(format string, args ...interface{}) {
	app.logDebug(format, args...)
}

// createDebugLog creates debug log file in user's .sterm directory
func createDebugLog() *os.File {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		debugLog, _ := os.Create("sterm-debug.log")
		return debugLog
	}

	// Create .sterm directory if it doesn't exist
	serialTerminalDir := filepath.Join(homeDir, ".sterm")
	if err := os.MkdirAll(serialTerminalDir, 0755); err != nil {
		// Fallback to current directory
		debugLog, _ := os.Create("sterm-debug.log")
		return debugLog
	}

	// Create debug log file in the directory
	debugLogPath := filepath.Join(serialTerminalDir, "sterm-debug.log")
	debugLog, err := os.Create(debugLogPath)
	if err != nil {
		// Fallback to current directory
		debugLog, _ = os.Create("sterm-debug.log")
		return debugLog
	}

	return debugLog
}

// NewApplication creates a new application instance
func NewApplication(config AppConfig) (*Application, error) {
	// Validate configuration
	if err := config.SerialConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid serial config: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create debug log file only if debug mode is enabled
	var debugLog *os.File
	if config.DebugMode {
		debugLog = createDebugLog()
	}

	// Create components
	app := &Application{
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		updateNotify: make(chan struct{}, 100), // Buffered channel for updates
		pauseChan:    make(chan bool, 1),       // Channel for pause control
		isRunning:    false,
		isPaused:     false,
		localEcho:    false, // Local echo off by default
		lineWrap:     true,  // Line wrap on by default
		debugLog:     debugLog,
		debugMode:    config.DebugMode,
	}

	// Initialize components
	if err := app.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return app, nil
}

// initializeComponents initializes all application components
func (app *Application) initializeComponents() error {
	// Create serial port
	app.serialPort = serial.NewSerialPort()

	// Create config manager
	app.configMgr = config.NewFileConfigManager("")

	// Create history manager
	var err error
	app.historyMgr = history.NewMemoryHistoryManager(app.config.HistorySize)

	// Create screen
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}

	// Use default terminal colors instead of forcing black background
	defaultStyle := tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset)
	screen.SetStyle(defaultStyle)
	screen.Clear()

	// Don't enable mouse by default to preserve text selection
	// Mouse will only be enabled when terminal explicitly requests it
	// Users can use Ctrl+PageUp/Down for scrolling instead

	app.screen = screen

	// Get actual terminal dimensions from tcell screen
	width, height := screen.Size()
	// Only override if config explicitly sets non-zero values
	// Otherwise use the actual terminal size
	if app.config.TerminalWidth <= 0 || app.config.TerminalHeight <= 0 {
		app.config.TerminalWidth = width
		app.config.TerminalHeight = height - 1 // Reserve 1 line for status bar
	} else {
		// Use configured size if explicitly set
		width = app.config.TerminalWidth
		height = app.config.TerminalHeight
	}

	// Create terminal emulator (with reduced height for status bar)
	app.terminal = terminal.NewTerminalEmulator(
		nil, // Will set input/output later
		nil,
		width,
		height,
	)

	// Set initial line wrap state
	app.terminal.SetLineWrap(app.lineWrap)

	// Set logger for terminal debugging
	app.terminal.SetLogger(app)

	// Set mouse mode change callback to dynamically enable/disable mouse
	app.terminal.SetMouseModeChangeCallback(func(mode terminal.MouseMode) {
		if mode == terminal.MouseModeOff {
			// Disable tcell mouse to allow native text selection
			if app.screen != nil {
				app.screen.DisableMouse()
				app.logDebug("Mouse disabled in tcell for native text selection")
			}
		} else {
			// Enable tcell mouse for terminal mouse events
			if app.screen != nil && app.config.EnableMouse {
				app.screen.EnableMouse()
				app.logDebug("Mouse enabled in tcell for mode: %v", mode)
			}
		}
	})

	// Create input processor (single instance to maintain state)
	app.inputProcessor = terminal.NewInputProcessor(app.terminal)

	// Create shortcut manager
	app.shortcuts = terminal.NewShortcutManager()
	app.setupShortcuts()

	// Create menu system
	app.overlayMgr = menu.NewOverlayManager(app.screen)
	app.mainMenu = menu.NewMenu("Serial Terminal", app.screen)
	app.setupMenu()

	return nil
}

// setupShortcuts sets up application shortcuts
func (app *Application) setupShortcuts() {
	// Exit shortcut - use Ctrl+Shift+Q to avoid conflict with terminal
	_ = app.shortcuts.SetShortcutHandler("exit", func() error {
		return app.Stop()
	})

	// Save history shortcut
	_ = app.shortcuts.SetShortcutHandler("save", func() error {
		return app.SaveHistory("")
	})

	// Clear screen shortcut
	_ = app.shortcuts.SetShortcutHandler("clear", func() error {
		return app.ClearScreen()
	})

	// Pause/Resume shortcut
	app.shortcuts.CustomShortcut(
		"pause",
		"Pause/Resume data flow",
		tcell.KeyF8,
		0,
		0,
		func() error {
			if app.isPaused {
				return app.Resume()
			}
			return app.Pause()
		},
	)

	// Disconnect shortcut
	_ = app.shortcuts.SetShortcutHandler("disconnect", func() error {
		return app.Disconnect()
	})

	// Help shortcut - show main menu which contains help and options
	_ = app.shortcuts.SetShortcutHandler("help", func() error {
		if app.mainMenu != nil && app.mainMenu.IsVisible() {
			app.hideMainMenu()
		} else {
			app.showMainMenu()
		}
		return nil
	})
}

// Start starts the application
func (app *Application) Start() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.isRunning {
		return fmt.Errorf("application is already running")
	}

	// Open serial port
	if err := app.serialPort.Open(app.config.SerialConfig); err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	// Create session
	app.session = NewSession(
		fmt.Sprintf("%s_%d", app.config.SerialConfig.Port, app.config.SerialConfig.BaudRate),
		app.config.SerialConfig,
	)

	// Start terminal
	if err := app.terminal.Start(); err != nil {
		app.serialPort.Close()
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	// Set running state
	app.isRunning = true

	// Send initial terminal size to remote device if configured
	if app.config.SendWindowSizeOnConnect {
		width, height := app.screen.Size()
		// Reserve 1 line for status bar
		terminalHeight := height - 1
		if app.serialPort != nil && app.serialPort.IsOpen() {
			// Send terminal type response based on configuration
			if app.config.TerminalType == "vt100" {
				_, _ = app.serialPort.Write([]byte("\x1b[?1;2c")) // VT100 with AVO
			} else if app.config.TerminalType == "xterm" {
				_, _ = app.serialPort.Write([]byte("\x1b[?62;c")) // xterm
			}

			// Send window size using stty-compatible format
			// Some systems expect: ESC[8;<height>;<width>t
			// Others use environment variables or stty
			sizeSeq := fmt.Sprintf("\x1b[8;%d;%dt", terminalHeight, width)
			_, _ = app.serialPort.Write([]byte(sizeSeq))

			// Also try sending as environment variable format
			// This helps with programs that use LINES/COLUMNS
			envSeq := fmt.Sprintf("\x1b]0;LINES=%d;COLUMNS=%d\x07", terminalHeight, width)
			_, _ = app.serialPort.Write([]byte(envSeq))

			app.logDebug("Sent initial terminal size %dx%d to remote", width, terminalHeight)
		}
	}

	// Start data flow goroutines
	app.wg.Add(2)
	go app.handleSerialInput()
	go app.handleUserInput()

	// Start UI update loop
	app.wg.Add(1)
	go app.updateUI()

	return nil
}

// Stop stops the application
func (app *Application) Stop() error {
	app.logDebug("Stop() called")

	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.isRunning {
		app.logDebug("App already stopped")
		return nil
	}

	app.logDebug("Setting isRunning to false")
	// Set running state immediately to stop loops
	app.isRunning = false

	app.logDebug("Canceling context")
	// Cancel context to stop goroutines
	app.cancel()

	// Post a special event to break out of PollEvent
	if app.screen != nil {
		app.logDebug("Posting interrupt event")
		// Post a resize event to wake up PollEvent
		_ = app.screen.PostEvent(tcell.NewEventResize(0, 0))
	}

	// Close serial port first to stop I/O
	if app.serialPort != nil && app.serialPort.IsOpen() {
		app.logDebug("Closing serial port")
		app.serialPort.Close()
	}

	// Stop terminal
	if app.terminal != nil {
		_ = app.terminal.Stop()
	}

	app.logDebug("Waiting for goroutines to finish...")
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.logDebug("All goroutines finished")
		// Goroutines finished normally
	case <-time.After(2 * time.Second):
		app.logDebug("Timeout waiting for goroutines")
		// Force continue after timeout
		fmt.Println("Warning: Some goroutines didn't stop cleanly")
	}

	// Now safe to finalize screen
	if app.screen != nil {
		app.screen.Fini()
		app.screen = nil
	}

	// End session
	if app.session != nil {
		app.session.End()
	}

	// Save history if configured and debug mode is enabled
	if app.config.SaveHistory && app.debugMode && app.historyMgr != nil && app.session != nil {
		filename := fmt.Sprintf("session_%s.log", app.session.ID)
		_ = app.historyMgr.SaveToFile(filename, app.config.HistoryFormat)
	}

	// Close debug log
	if app.debugLog != nil {
		app.debugLog.Close()
		app.debugLog = nil
	}

	return nil
}

// handleSerialInput reads data from serial port and sends to terminal
func (app *Application) handleSerialInput() {
	defer app.wg.Done()

	// Use larger buffer for better performance with high-speed data
	buffer := make([]byte, 65536) // 64KB buffer

	// Track last data receive time for flush detection
	var lastDataTime time.Time
	flushTimer := time.NewTimer(100 * time.Millisecond) // Increased to 100ms for better reliability
	flushTimer.Stop()
	needsFlush := false

	for {
		select {
		case <-app.ctx.Done():
			return
		case isPaused := <-app.pauseChan:
			// Handle pause state change
			if isPaused {
				// Wait for resume signal
				for {
					select {
					case <-app.ctx.Done():
						return
					case resumed := <-app.pauseChan:
						if !resumed {
							break
						}
					}
					// Break inner loop when resumed
					if !app.isPaused {
						break
					}
				}
			}
		case <-flushTimer.C:
			// Force UI update after a period of no data
			if needsFlush {
				app.forceImmediateUIUpdate()
				needsFlush = false
			}
		default:
			// Check if paused without blocking
			if app.isPaused {
				// Wait a bit before checking again
				select {
				case <-app.ctx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}

			// Read from serial port with timeout
			app.serialPort.SetReadTimeout(100 * time.Millisecond)
			n, err := app.serialPort.Read(buffer)
			if err != nil {
				// Timeout or error - check if we need to flush
				if needsFlush && !lastDataTime.IsZero() && time.Since(lastDataTime) > 100*time.Millisecond {
					// Force a final UI update if we haven't received data for 100ms
					app.logDebug("Read timeout - forcing immediate UI update")
					app.forceImmediateUIUpdate()
					lastDataTime = time.Time{}
					needsFlush = false
				}
				continue
			}

			if n > 0 {
				data := buffer[:n]

				// Process in terminal
				err := app.terminal.ProcessOutput(data)
				if err != nil {
					app.logDebug("ProcessOutput error: %v", err)
				}

				// Save to history
				if app.historyMgr != nil {
					_ = app.historyMgr.Write(data, history.DirectionOutput)
				}

				// Update session stats
				if app.session != nil {
					app.session.UpdateStats(0, int64(n))
				}

				// Request UI update
				app.requestUIUpdate()

				// Track when we last received data
				lastDataTime = time.Now()
				needsFlush = true

				// Reset flush timer - will fire if no more data arrives
				if !flushTimer.Stop() {
					// Drain the channel if timer already fired
					select {
					case <-flushTimer.C:
					default:
					}
				}
				flushTimer.Reset(100 * time.Millisecond)
			}
		}
	}
}

// handleUserInput handles keyboard and mouse input
func (app *Application) handleUserInput() {
	defer app.wg.Done()

	eventChan := make(chan tcell.Event)
	exitChan := make(chan struct{})

	go func() {
		for {
			// Check if we should exit before polling
			select {
			case <-app.ctx.Done():
				close(exitChan)
				return
			case <-exitChan:
				return
			default:
			}

			if !app.isRunning {
				close(exitChan)
				return
			}

			// PollEvent will block, but we need to be able to interrupt it
			// We use PostEventInterrupt to break out of PollEvent when stopping
			event := app.screen.PollEvent()
			if event != nil {
				select {
				case eventChan <- event:
				case <-app.ctx.Done():
					close(exitChan)
					return
				}
			}
		}
	}()

	for {
		select {
		case <-app.ctx.Done():
			app.logDebug("handleUserInput: context done")
			// Post an event to break PollEvent
			if app.screen != nil {
				_ = app.screen.PostEvent(tcell.NewEventResize(0, 0))
			}
			return
		case <-exitChan:
			app.logDebug("handleUserInput: exit channel closed")
			return
		case event := <-eventChan:
			if !app.isRunning {
				app.logDebug("handleUserInput: app not running")
				return
			}

			switch ev := event.(type) {
			case *tcell.EventKey:
				app.handleKeyEvent(ev)
			case *tcell.EventMouse:
				app.handleMouseEvent(ev)
			case *tcell.EventResize:
				app.handleResize()
			}
		}
	}
}

// handleKeyEvent handles keyboard events
func (app *Application) handleKeyEvent(ev *tcell.EventKey) {
	// Debug log key events when debug mode is enabled
	if app.debugMode {
		if ev.Key() == tcell.KeyRune {
			app.logDebug("Key: Rune='%c'(0x%x), Mods=%v", ev.Rune(), ev.Rune(), ev.Modifiers())
		} else {
			app.logDebug("Key: Key=%v, Mods=%v", ev.Key(), ev.Modifiers())
		}

		// Log terminal state when key is pressed
		screen := app.terminal.GetScreen()
		if screen != nil {
			app.logDebug("Key press - Screen dirty: %v, DirtyLines: %d", screen.Dirty, len(screen.DirtyLines))
		}
	}

	// Check if menu is visible and handle its input first
	if app.mainMenu != nil && app.mainMenu.IsVisible() {
		if app.mainMenu.HandleKey(ev) {
			return
		}
	}

	// Check for exit combinations
	// Key=17 is tcell.KeyCtrlQ
	// Mods=3 means Ctrl+Shift (1+2=3)
	// Mods=2 means Ctrl only
	if ev.Key() == tcell.KeyCtrlQ && ev.Modifiers() == (tcell.ModCtrl|tcell.ModShift) {
		app.logDebug("Ctrl+Shift+Q exit detected! (Key=%v, Mods=%v)", ev.Key(), ev.Modifiers())
		app.logDebug("Calling app.Stop()...")
		go func() {
			if err := app.Stop(); err != nil {
				app.logDebug("Error stopping app: %v", err)
			}
		}()
		return
	}

	// Also check if it comes as Key=17 directly
	if ev.Key() == 17 && ev.Modifiers() == 3 { // 3 = Ctrl+Shift
		app.logDebug("Ctrl+Shift+Q exit detected! (raw Key=17, Mods=3)")
		app.logDebug("Calling app.Stop()...")
		go func() {
			if err := app.Stop(); err != nil {
				app.logDebug("Error stopping app: %v", err)
			}
		}()
		return
	}

	// Alternative: Allow simple Ctrl+Q as fallback
	if ev.Key() == tcell.KeyCtrlQ && ev.Modifiers() == tcell.ModCtrl {
		app.logDebug("Ctrl+Q exit detected!")
		app.logDebug("Calling app.Stop()...")
		go func() {
			if err := app.Stop(); err != nil {
				app.logDebug("Error stopping app: %v", err)
			}
		}()
		return
	}

	// Check for F1 key first - let shortcuts handle it if defined
	if ev.Key() == tcell.KeyF1 {
		app.logDebug("F1 key pressed - will be handled by shortcuts if enabled")
		// Don't return here - let it fall through to shortcut processing
	}

	// Check for F8 pause/resume
	if ev.Key() == tcell.KeyF8 {
		app.logDebug("F8 pause/resume key pressed")
		if app.isPaused {
			_ = app.Resume()
		} else {
			_ = app.Pause()
		}
		app.updateDisplay() // Force immediate display refresh
		return
	}

	// Check for menu shortcuts when menu is NOT visible
	// Using Alt+ combinations to avoid conflicts with bash and other terminal applications
	if !app.mainMenu.IsVisible() {
		// Check for Alt+ combinations
		if ev.Modifiers()&tcell.ModAlt != 0 {
			switch ev.Rune() {
			case 'c', 'C':
				// Alt+C - Clear Screen
				app.logDebug("Alt+C Clear Screen shortcut")
				if err := app.ClearScreen(); err != nil {
					app.updateStatusMessage(fmt.Sprintf("Clear screen failed: %v", err))
				} else {
					app.updateStatusMessage("Screen cleared")
				}
				return
			case 'h', 'H':
				// Alt+H - Clear History
				app.logDebug("Alt+H Clear History shortcut")
				app.terminal.ClearScrollback()
				app.updateStatusMessage("History cleared")
				return
			case 'r', 'R':
				// Alt+R - Reconnect
				app.logDebug("Alt+R Reconnect shortcut")
				if err := app.Reconnect(); err != nil {
					app.updateStatusMessage(fmt.Sprintf("Reconnect failed: %v", err))
				} else {
					app.updateStatusMessage("Reconnected successfully")
				}
				return
			case 's', 'S':
				// Alt+S - Save Session
				app.logDebug("Alt+S Save Session shortcut")
				if err := app.saveSessionToFile(); err != nil {
					app.updateStatusMessage(fmt.Sprintf("Save failed: %v", err))
				} else {
					filename := fmt.Sprintf("session_%s.txt", time.Now().Format("20060102_150405"))
					app.updateStatusMessage(fmt.Sprintf("Session saved to %s", filename))
				}
				return
			}
		}
	}

	// Handle scrolling keys - Shift+PageUp/Up enters scroll mode
	switch ev.Key() {
	case tcell.KeyPgUp:
		if ev.Modifiers()&tcell.ModShift != 0 {
			// Shift+PageUp - scroll up one page and enter scroll mode
			if !app.terminal.IsScrolling() {
				app.terminal.EnterScrollMode()
			}
			height := app.terminal.GetState().Height
			app.terminal.ScrollUp(height)
			app.updateDisplay()
			return
		}
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			// Ctrl+PageUp - scroll up one page (alternative)
			height := app.terminal.GetState().Height
			app.terminal.ScrollUp(height)
			app.updateDisplay()
			return
		}
	case tcell.KeyPgDn:
		if ev.Modifiers()&tcell.ModShift != 0 {
			// Shift+PageDown - scroll down one page in scroll mode
			if !app.terminal.IsScrolling() {
				app.terminal.EnterScrollMode()
			}
			height := app.terminal.GetState().Height
			app.terminal.ScrollDown(height)
			app.updateDisplay()
			return
		}
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			// Ctrl+PageDown - scroll down one page (alternative)
			height := app.terminal.GetState().Height
			app.terminal.ScrollDown(height)
			app.updateDisplay()
			return
		}
	case tcell.KeyUp:
		if ev.Modifiers()&tcell.ModShift != 0 {
			// Shift+Up - scroll up one line and enter scroll mode
			if !app.terminal.IsScrolling() {
				app.terminal.EnterScrollMode()
			}
			app.terminal.ScrollUp(1)
			app.updateDisplay()
			return
		}
	case tcell.KeyDown:
		if ev.Modifiers()&tcell.ModShift != 0 {
			// Shift+Down - scroll down one line in scroll mode
			if !app.terminal.IsScrolling() {
				app.terminal.EnterScrollMode()
			}
			app.terminal.ScrollDown(1)
			app.updateDisplay()
			return
		}
	case tcell.KeyHome:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			// Ctrl+Home - scroll to top
			app.terminal.ScrollToTop()
			app.updateDisplay()
			return
		}
	case tcell.KeyEnd:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			// Ctrl+End - scroll to bottom (stay in scroll mode)
			app.terminal.ScrollToBottom()
			app.updateDisplay()
			return
		}
	}

	// If we're in scroll mode, handle scroll-specific keys
	if app.terminal.IsScrolling() {
		handled := false
		switch ev.Key() {
		case tcell.KeyF1:
			// F1 should still work in scroll mode to show menu
			// Let it fall through to normal processing
			// Don't set handled=true, so it continues to shortcut processing
			handled = false
		case tcell.KeyEscape:
			// ESC exits scroll mode
			app.terminal.ExitScrollMode()
			app.updateDisplay()
			return
		case tcell.KeyEnter:
			// Enter exits scroll mode
			app.terminal.ExitScrollMode()
			app.updateDisplay()
			return
		case tcell.KeyRune:
			// 'q' also exits scroll mode for convenience
			if ev.Rune() == 'q' || ev.Rune() == 'Q' {
				app.terminal.ExitScrollMode()
				app.updateDisplay()
				return
			}
			// Vi-style navigation in scroll mode
			switch ev.Rune() {
			case 'j', 'J': // Down
				app.terminal.ScrollDown(1)
				handled = true
			case 'k', 'K': // Up
				app.terminal.ScrollUp(1)
				handled = true
			case 'h', 'H': // Left (not used in vertical scroll)
				handled = true
			case 'l', 'L': // Right (not used in vertical scroll)
				handled = true
			case 'g', 'G': // Top/Bottom (stay in scroll mode)
				if ev.Modifiers()&tcell.ModShift != 0 { // G - go to bottom
					app.terminal.ScrollToBottom()
				} else { // g - go to top
					app.terminal.ScrollToTop()
				}
				handled = true
			case 'd', 'D': // Half page down
				height := app.terminal.GetState().Height
				app.terminal.ScrollDown(height / 2)
				handled = true
			case 'u', 'U': // Half page up
				height := app.terminal.GetState().Height
				app.terminal.ScrollUp(height / 2)
				handled = true
			case 'f', 'F': // Page down (forward)
				height := app.terminal.GetState().Height
				app.terminal.ScrollDown(height)
				handled = true
			case 'b', 'B': // Page up (backward)
				height := app.terminal.GetState().Height
				app.terminal.ScrollUp(height)
				handled = true
			}
		case tcell.KeyUp:
			app.terminal.ScrollUp(1)
			handled = true
		case tcell.KeyDown:
			app.terminal.ScrollDown(1)
			handled = true
		case tcell.KeyLeft, tcell.KeyRight:
			// Ignore horizontal movement in vertical scroll
			handled = true
		case tcell.KeyPgUp:
			height := app.terminal.GetState().Height
			app.terminal.ScrollUp(height)
			handled = true
		case tcell.KeyPgDn:
			height := app.terminal.GetState().Height
			app.terminal.ScrollDown(height)
			handled = true
		case tcell.KeyHome:
			app.terminal.ScrollToTop()
			handled = true
		case tcell.KeyEnd:
			app.terminal.ScrollToBottom()
			handled = true
		}

		if handled {
			app.updateDisplay()
			return
		}

		// F1 key should pass through to shortcuts even in scroll mode
		if ev.Key() != tcell.KeyF1 {
			// Other keys don't exit scroll mode, just ignore them
			return
		}
		// F1 continues to shortcut processing below
	}

	// Check shortcuts first
	if app.config.EnableShortcuts && app.shortcuts.IsEnabled() {
		app.logDebug("Processing shortcuts, enabled=%v", app.shortcuts.IsEnabled())
		handled, err := app.shortcuts.ProcessKeyEvent(ev.Key(), ev.Rune(), ev.Modifiers())
		if err != nil {
			app.logDebug("Shortcut error: %v", err)
		}
		if handled {
			app.logDebug("Shortcut handled")
			return
		}
	}

	// Process as terminal input using shared processor
	data := app.inputProcessor.ProcessKeyEvent(ev)

	if len(data) > 0 && !app.isPaused {
		// Local echo - display the input locally if enabled
		if app.localEcho && app.terminal != nil {
			// Process the input locally to show it on screen
			_ = app.terminal.ProcessOutput(data)
		}

		// Send to serial port
		if app.serialPort != nil && app.serialPort.IsOpen() {
			n, _ := app.serialPort.Write(data)

			// Save to history
			if app.historyMgr != nil {
				_ = app.historyMgr.Write(data[:n], history.DirectionInput)
			}

			// Update session stats
			if app.session != nil {
				app.session.UpdateStats(int64(n), 0)
			}
		}
	}
}

// handleMouseEvent handles mouse events
func (app *Application) handleMouseEvent(ev *tcell.EventMouse) {
	// Only process mouse events if mouse is enabled (terminal requested it)
	mouseMode := app.terminal.GetState().MouseMode

	// Only process mouse events when terminal has requested mouse mode
	if mouseMode == terminal.MouseModeOff {
		// Mouse mode is off, don't process any mouse events
		// This preserves text selection when tcell mouse is disabled
		return
	}

	if !app.config.EnableMouse {
		// Mouse support disabled in config
		return
	}

	// Use shared input processor to maintain mouse button state
	data := app.inputProcessor.ProcessMouseEvent(ev)

	if len(data) > 0 {
		// app.logDebug("Mouse sequence generated: %X (%d bytes)", data, len(data))
		if !app.isPaused {
			// Send to serial port
			if app.serialPort != nil && app.serialPort.IsOpen() {
				_, err := app.serialPort.Write(data)
				if err != nil {
					app.logDebug("Failed to send mouse sequence: %v", err)
				}
				// Commented out for performance
				// else {
				// 	app.logDebug("Sent %d bytes of mouse sequence", n)
				// }
			}
		}
		// Commented out for performance
		// else {
		// 	app.logDebug("Terminal paused, not sending mouse sequence")
		// }
	}
	// Commented out for performance
	// else {
	// 	app.logDebug("No mouse sequence generated (mouse mode may be off)")
	// }
}

// handleResize handles terminal resize events
func (app *Application) handleResize() {
	width, height := app.screen.Size()
	// Reserve 1 line for status bar
	terminalHeight := height - 1
	_ = app.terminal.Resize(width, terminalHeight)

	// Only send terminal size update if explicitly configured
	// Most serial devices don't support this and it causes garbage output
	if app.config.SendWindowSizeOnResize {
		if app.serialPort != nil && app.serialPort.IsOpen() && !app.isPaused {
			// Send the actual terminal size (without status bar)
			sizeSeq := fmt.Sprintf("\x1b[8;%d;%dt", terminalHeight, width)
			_, _ = app.serialPort.Write([]byte(sizeSeq))

			app.logDebug("Window resized to %dx%d, sent size update to remote", width, terminalHeight)
		}
	} else {
		app.logDebug("Window resized to %dx%d (not sending to remote)", width, terminalHeight)
	}

	app.screen.Clear()
	app.updateDisplay()
}

// updateUI updates the terminal display
func (app *Application) updateUI() {
	defer app.wg.Done()

	// Create a ticker for minimum refresh interval (to handle rapid updates)
	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS max
	defer ticker.Stop()

	lastUpdate := time.Now()
	pendingUpdate := false
	updateCount := 0
	rateLimitWarning := false
	lastPendingTime := time.Now()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-app.updateNotify:
			// Mark that we have a pending update
			pendingUpdate = true
			lastPendingTime = time.Now()

			// Log pending update
			if len(app.updateNotify) > 10 {
				app.logDebug("Update queue size: %d", len(app.updateNotify))
			}

			// Drain extra notifications to prevent channel overflow
			for len(app.updateNotify) > 50 {
				<-app.updateNotify
				if !rateLimitWarning {
					app.logDebug("WARNING: UI update rate limit - dropping updates (queue size: %d)", len(app.updateNotify))
					rateLimitWarning = true
				}
			}
		case <-ticker.C:
			// Force update if pending for too long (prevent data stuck in buffer)
			if pendingUpdate && time.Since(lastPendingTime) > 20*time.Millisecond {
				// Reduced from 30ms to 20ms for better responsiveness
				app.logDebug("Force update - pending for %v", time.Since(lastPendingTime))
				app.updateDisplay()
				lastUpdate = time.Now()
				pendingUpdate = false
				rateLimitWarning = false
				updateCount = 0
			} else if pendingUpdate && time.Since(lastUpdate) >= 16*time.Millisecond {
				// Normal update with rate limiting
				updateCount++
				// Safety check - if we're updating too frequently, skip some frames
				if updateCount > 100 && time.Since(lastUpdate) < time.Second {
					app.logDebug("Skipping frame due to high update rate: %d updates/sec", updateCount)
					continue
				}
				if updateCount > 100 {
					updateCount = 0
				}

				app.updateDisplay()
				lastUpdate = time.Now()
				pendingUpdate = false
				rateLimitWarning = false
			} else if pendingUpdate {
				// Log if update is pending but not executed
				if app.debugMode && time.Since(lastPendingTime) > 100*time.Millisecond {
					app.logDebug("Update pending but not executed - waiting %v, last update %v ago",
						time.Since(lastPendingTime), time.Since(lastUpdate))
				}
			}
		}
	}
}

// requestUIUpdate requests a UI update
func (app *Application) requestUIUpdate() {
	select {
	case app.updateNotify <- struct{}{}:
		// Notification sent
	default:
		// Channel full, update already pending
	}
}

// forceImmediateUIUpdate forces an immediate UI update, bypassing the rate limiter
func (app *Application) forceImmediateUIUpdate() {
	// Get the screen to check if there's any unrendered content
	screen := app.terminal.GetScreen()
	state := app.terminal.GetState()
	if screen != nil {
		// If dirty bounds are invalid, we need to determine what to redraw
		if screen.DirtyMinY > screen.DirtyMaxY || len(screen.DirtyLines) == 0 {
			// Use cursor position as a hint - if cursor is not at start, there's likely content
			if state.CursorY > 0 || state.CursorX > 0 {
				// Mark from line 0 to cursor line as dirty to ensure all content is rendered
				endLine := state.CursorY
				if endLine >= screen.Height {
					endLine = screen.Height - 1
				}

				for y := 0; y <= endLine && y < len(screen.Buffer); y++ {
					screen.MarkLineDirty(y)
				}
			} else {
				// Even if cursor is at 0,0, mark entire screen to be safe
				for y := 0; y < screen.Height && y < len(screen.Buffer); y++ {
					screen.MarkLineDirty(y)
				}
			}
		}

		// Always mark screen as dirty to force a render pass
		screen.Dirty = true
	}

	app.updateDisplay()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// updateDisplay updates the screen with terminal content
func (app *Application) updateDisplay() {
	// Add panic recovery for display updates
	defer func() {
		if r := recover(); r != nil {
			app.logDebug("PANIC in updateDisplay: %v", r)
			fmt.Printf("Display update error: %v\n", r)
		}
	}()

	app.mu.RLock()
	defer app.mu.RUnlock()

	if !app.isRunning || app.screen == nil || app.terminal == nil {
		return
	}

	// Check if status message expired and needs redraw
	needsRedraw := false
	if app.statusMessage != "" && time.Since(app.statusTime) > 3*time.Second {
		app.statusMessage = ""
		needsRedraw = true
	}

	// Get terminal screen buffer
	screen := app.terminal.GetScreen()
	if screen == nil {
		return
	}

	// Check if screen was just cleared
	justCleared := screen.IsJustCleared()

	if !screen.Dirty && !needsRedraw && !justCleared {
		return
	}

	// Get terminal state
	state := app.terminal.GetState()

	// Get the appropriate buffer based on scroll mode
	var buffer [][]terminal.Cell
	if app.terminal.IsScrolling() {
		buffer = app.terminal.GetScrollbackView()
		// In scroll mode, redraw everything
		app.screen.Clear()
	} else {
		buffer = screen.Buffer
	}

	// Render cells (leave room for status bar at bottom)
	screenWidth, screenHeight := app.screen.Size()
	contentHeight := screenHeight - 1 // Reserve bottom line for status bar

	// Handle just cleared screen
	if justCleared {
		app.screen.Clear()
		// Clear the flag
		screen.ClearJustClearedFlag()
		// Force full redraw of current buffer to show any content (including prompt)
		for y := 0; y < contentHeight && y < len(buffer); y++ {
			for x := 0; x < screen.Width && x < len(buffer[y]); x++ {
				cell := buffer[y][x]
				app.renderCell(x, y, cell)
			}
		}
		// Clear dirty flags after full redraw
		screen.ClearDirty()
		// Continue to render status bar
	} else if app.terminal.IsScrolling() || needsRedraw {
		// Full redraw for scroll mode or when needed
		app.screen.Clear()
		for y := 0; y < contentHeight && y < len(buffer); y++ {
			for x := 0; x < screen.Width && x < len(buffer[y]); x++ {
				cell := buffer[y][x]
				app.renderCell(x, y, cell)
			}
		}
	} else {
		// Check if this is a full screen clear (all lines dirty and all spaces)
		isFullClear := false
		if screen.DirtyMinY == 0 && screen.DirtyMaxY >= screen.Height-1 {
			isFullClear = true
			nonSpaceCount := 0
			for y := 0; y < screen.Height && y < len(buffer); y++ {
				for x := 0; x < screen.Width && x < len(buffer[y]); x++ {
					if buffer[y][x].Char != ' ' && buffer[y][x].Char != 0 {
						nonSpaceCount++
					}
				}
			}

			// If all or mostly spaces (allow for prompt), consider it a clear
			totalCells := screen.Height * screen.Width
			if nonSpaceCount == 0 || (float64(nonSpaceCount)/float64(totalCells) < 0.01) {
				isFullClear = true
			} else {
				isFullClear = false
			}
		}

		if isFullClear {
			// Full screen clear detected - clear entire display
			app.screen.Clear()
			// Redraw empty screen
			for y := 0; y < contentHeight; y++ {
				for x := 0; x < screenWidth; x++ {
					app.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
				}
			}
			// Force immediate screen update
			app.screen.Show()
		} else {
			// Optimized: only redraw dirty cells
			// Check if we have any dirty regions
			if screen.DirtyMinY <= screen.DirtyMaxY {
				// Ensure bounds are within screen limits
				startY := screen.DirtyMinY
				if startY < 0 {
					startY = 0
				}
				endY := screen.DirtyMaxY
				if endY >= contentHeight {
					endY = contentHeight - 1
				}

				for y := startY; y <= endY && y < len(buffer); y++ {
					if !screen.IsLineDirty(y) {
						continue
					}

					// Check if this is a clear operation (entire line is spaces with dirty flag)
					allSpaces := true
					for x := 0; x < screen.Width && x < len(buffer[y]); x++ {
						if buffer[y][x].Char != ' ' || !buffer[y][x].Dirty {
							allSpaces = false
							break
						}
					}

					if allSpaces {
						// Clear the entire line for proper clearing
						for x := 0; x < screenWidth; x++ {
							app.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
						}
					} else {
						// Normal rendering of dirty cells
						for x := 0; x < screen.Width && x < len(buffer[y]); x++ {
							cell := buffer[y][x]
							if cell.Dirty {
								app.renderCell(x, y, cell)
							}
						}
					}
				}
			}
		}
	}

	// Always show status bar at bottom
	statusY := screenHeight - 1

	// Prepare status bar content
	var statusLeft, statusCenter, statusRight string

	// Left: Connection info (cache if unchanged)
	if app.cachedStatusLeft == "" || needsRedraw {
		if app.serialPort != nil && app.serialPort.IsOpen() {
			cfg := app.config.SerialConfig
			app.cachedStatusLeft = fmt.Sprintf(" %s %d ", cfg.Port, cfg.BaudRate)
		} else {
			app.cachedStatusLeft = " Disconnected "
		}
	}
	statusLeft = app.cachedStatusLeft

	// Center: Mode indicator or temporary status message
	if app.statusMessage != "" && time.Since(app.statusTime) < 3*time.Second {
		// Show temporary status message for 3 seconds
		statusCenter = fmt.Sprintf(" %s ", app.statusMessage)
	} else if app.terminal.IsScrolling() {
		current, total := app.terminal.GetScrollPosition()
		statusCenter = fmt.Sprintf(" SCROLL: %d/%d [j/k:↑↓ d/u:½Page f/b:Page g/G:Top/Bot ESC/Enter/q:Exit] ", current, total)
	} else if app.isPaused {
		statusCenter = " [Shift+PgUp/↑: Scroll] [F1: Menu] PAUSED [F8: Resume] "
	} else {
		// Show hint for scroll mode and pause
		statusCenter = " [Shift+PgUp/↑: Scroll] [F1: Menu] [F8: Pause] "
	}

	// Right: Session info (cache and update only when changed)
	if app.session != nil {
		currentSent := app.session.BytesSent
		currentRecv := app.session.BytesRecv
		if currentSent != app.cachedBytesSent || currentRecv != app.cachedBytesRecv || needsRedraw {
			app.cachedBytesSent = currentSent
			app.cachedBytesRecv = currentRecv
			app.cachedStatusRight = fmt.Sprintf(" TX:%d RX:%d ", currentSent, currentRecv)
		}
		statusRight = app.cachedStatusRight
	}

	// Draw status bar with different style
	statusStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite)

	// Fill entire bottom line
	for x := 0; x < screenWidth; x++ {
		app.screen.SetContent(x, statusY, ' ', nil, statusStyle)
	}

	// Draw left text
	x := 0
	for _, ch := range statusLeft {
		if x < screenWidth {
			app.screen.SetContent(x, statusY, ch, nil, statusStyle.Bold(true))
			x += runewidth.RuneWidth(ch)
		}
	}

	// Draw center text
	// Use runewidth to calculate actual display width
	centerWidth := runewidth.StringWidth(statusCenter)
	centerX := (screenWidth - centerWidth) / 2
	if centerX < 0 {
		centerX = 0
	}
	x = centerX
	pauseIndicator := "PAUSED [F8: Resume]"
	runeIndex := 0
	for _, ch := range statusCenter {
		if x < screenWidth {
			if app.statusMessage != "" && time.Since(app.statusTime) < 3*time.Second {
				// Highlight status message with green background
				app.screen.SetContent(x, statusY, ch, nil,
					statusStyle.Background(tcell.ColorDarkGreen).Bold(true))
			} else if app.terminal.IsScrolling() {
				// Highlight scroll mode
				app.screen.SetContent(x, statusY, ch, nil,
					statusStyle.Background(tcell.ColorDarkCyan).Bold(true))
			} else if app.isPaused {
				// Check if current character is part of the pause indicator
				pauseStart := strings.Index(statusCenter, pauseIndicator)
				// Convert string index to rune index
				runesBeforePause := len([]rune(statusCenter[:pauseStart]))
				pauseRuneCount := len([]rune(pauseIndicator))
				if pauseStart >= 0 && runeIndex >= runesBeforePause && runeIndex < runesBeforePause+pauseRuneCount {
					// Highlight only the pause indicator with red background
					app.screen.SetContent(x, statusY, ch, nil,
						statusStyle.Background(tcell.ColorDarkRed).Bold(true))
				} else {
					// Normal style for other parts
					app.screen.SetContent(x, statusY, ch, nil, statusStyle)
				}
			} else {
				app.screen.SetContent(x, statusY, ch, nil, statusStyle)
			}
			x += runewidth.RuneWidth(ch)
			runeIndex++
		}
	}

	// Draw right text
	rightWidth := runewidth.StringWidth(statusRight)
	rightX := screenWidth - rightWidth
	if rightX < 0 {
		rightX = 0
	}
	x = rightX
	for _, ch := range statusRight {
		if x < screenWidth {
			app.screen.SetContent(x, statusY, ch, nil, statusStyle)
			x += runewidth.RuneWidth(ch)
		}
	}

	// Show cursor (adjusted for status bar)
	if !app.terminal.IsScrolling() {
		if state.CursorX >= 0 && state.CursorX < screen.Width &&
			state.CursorY >= 0 && state.CursorY < contentHeight {
			app.screen.ShowCursor(state.CursorX, state.CursorY)
		}
	}

	// Show the screen
	app.screen.Show()

	// If menu is visible, redraw it on top
	if app.mainMenu != nil && app.mainMenu.IsVisible() {
		app.mainMenu.Draw()
	}

	// Clear dirty flags
	screen.ClearDirty()
}

// Pause pauses data flow
func (app *Application) Pause() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.isRunning {
		return fmt.Errorf("application is not running")
	}

	if !app.isPaused {
		app.isPaused = true
		// Notify pause through channel
		select {
		case app.pauseChan <- true:
		default:
		}
		// Mark screen as dirty to force redraw
		if app.terminal != nil {
			screen := app.terminal.GetScreen()
			if screen != nil {
				screen.Dirty = true
			}
		}
		// Request UI update to show paused state
		app.requestUIUpdate()
	}
	return nil
}

// Resume resumes data flow
func (app *Application) Resume() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.isRunning {
		return fmt.Errorf("application is not running")
	}

	if app.isPaused {
		app.isPaused = false
		// Notify resume through channel
		select {
		case app.pauseChan <- false:
		default:
		}
		// Mark screen as dirty to force redraw
		if app.terminal != nil {
			screen := app.terminal.GetScreen()
			if screen != nil {
				screen.Dirty = true
			}
		}
		// Request UI update to show resumed state
		app.requestUIUpdate()
	}
	return nil
}

// SaveHistory saves the current history to a file
func (app *Application) SaveHistory(filename string) error {
	if app.historyMgr == nil {
		return fmt.Errorf("history manager not initialized")
	}

	if filename == "" {
		filename = fmt.Sprintf("history_%s.log", time.Now().Format("20060102_150405"))
	}

	return app.historyMgr.SaveToFile(filename, app.config.HistoryFormat)
}

// ClearScreen clears the terminal screen
func (app *Application) ClearScreen() error {
	if app.terminal == nil {
		return fmt.Errorf("terminal not initialized")
	}

	// Lock to ensure thread safety
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logDebug("=== ClearScreen Start ===")

	// Exit scroll mode if active
	if app.terminal.IsScrolling() {
		app.logDebug("Exiting scroll mode")
		app.terminal.ExitScrollMode()
	}

	// Log terminal state before clear
	termState := app.terminal.GetState()
	app.logDebug("Before clear - Cursor: (%d, %d), Screen: %dx%d",
		termState.CursorX, termState.CursorY, termState.Width, termState.Height)

	// Send clear screen sequence
	clearSeq := []byte{0x1B, '[', '2', 'J', 0x1B, '[', 'H'}
	err := app.terminal.ProcessOutput(clearSeq)

	if err != nil {
		app.logDebug("ProcessOutput error: %v", err)
	}

	// Log terminal state after clear
	termStateAfter := app.terminal.GetState()
	app.logDebug("After clear - Cursor: (%d, %d)", termStateAfter.CursorX, termStateAfter.CursorY)

	// Force complete screen redraw
	if app.screen != nil {
		// Clear the physical screen
		app.screen.Clear()

		// Get the cleared terminal buffer
		screen := app.terminal.GetScreen()
		if screen != nil {
			app.logDebug("Screen buffer - Dirty: %v, DirtyLines count: %d, Bounds: Y(%d-%d) X(%d-%d)",
				screen.Dirty, len(screen.DirtyLines),
				screen.DirtyMinY, screen.DirtyMaxY, screen.DirtyMinX, screen.DirtyMaxX)

			// Check first few lines of buffer content
			for y := 0; y < 3 && y < len(screen.Buffer); y++ {
				lineEmpty := true
				for x := 0; x < len(screen.Buffer[y]) && x < 10; x++ {
					if screen.Buffer[y][x].Char != ' ' && screen.Buffer[y][x].Char != 0 {
						lineEmpty = false
						break
					}
				}
				app.logDebug("Line %d empty: %v", y, lineEmpty)
			}

			// Ensure screen bounds are correct
			screenWidth, screenHeight := app.screen.Size()
			contentHeight := screenHeight - 1 // Reserve bottom line for status bar

			app.logDebug("Clearing screen area: %dx%d (content height: %d)", screenWidth, screenHeight, contentHeight)

			// Redraw the cleared terminal buffer (should be all spaces)
			for y := 0; y < contentHeight && y < screen.Height && y < len(screen.Buffer); y++ {
				for x := 0; x < screenWidth && x < screen.Width && x < len(screen.Buffer[y]); x++ {
					// Force render spaces to clear any residual content
					app.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
				}
			}

			// Clear any remaining lines
			for y := screen.Height; y < contentHeight; y++ {
				for x := 0; x < screenWidth; x++ {
					app.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
				}
			}
		} else {
			app.logDebug("ERROR: screen is nil after GetScreen()")
		}

		// Show changes immediately
		app.screen.Show()
	} else {
		app.logDebug("ERROR: app.screen is nil")
	}

	// Force immediate UI update
	app.requestUIUpdate()

	app.logDebug("=== ClearScreen End ===")

	return err
}

// Disconnect disconnects from the serial port
func (app *Application) Disconnect() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.serialPort != nil && app.serialPort.IsOpen() {
		return app.serialPort.Close()
	}

	return nil
}

// Reconnect reconnects to the serial port
func (app *Application) Reconnect() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Disconnect first
	if app.serialPort != nil && app.serialPort.IsOpen() {
		app.serialPort.Close()
	}

	// Reconnect
	return app.serialPort.Open(app.config.SerialConfig)
}

// GetSession returns the current session
func (app *Application) GetSession() *Session {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.session
}

// GetStats returns application statistics
func (app *Application) GetStats() (bytesSent, bytesRecv int64, duration time.Duration) {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if app.session == nil {
		return 0, 0, 0
	}

	bytesSent, bytesRecv = app.session.GetStats()
	duration = time.Since(app.session.StartTime)

	return bytesSent, bytesRecv, duration
}

// IsRunning returns whether the application is running
func (app *Application) IsRunning() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.isRunning
}

// IsPaused returns whether the application is paused
func (app *Application) IsPaused() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.isPaused
}

// renderCell renders a single cell to the screen
func (app *Application) renderCell(x, y int, cell terminal.Cell) {
	// Bounds check
	width, height := app.screen.Size()
	if x < 0 || x >= width || y < 0 || y >= height {
		app.logDebug("renderCell out of bounds: x=%d, y=%d, screen=%dx%d", x, y, width, height)
		return
	}

	// Convert terminal colors to tcell colors
	style := tcell.StyleDefault

	// Set foreground color
	style = style.Foreground(convertColor(cell.Attributes.Foreground))

	// Set background color
	style = style.Background(convertColor(cell.Attributes.Background))

	// Apply attributes
	if cell.Attributes.Bold {
		style = style.Bold(true)
	}
	if cell.Attributes.Italic {
		style = style.Italic(true)
	}
	if cell.Attributes.Underline {
		style = style.Underline(true)
	}
	if cell.Attributes.Reverse {
		style = style.Reverse(true)
	}
	if cell.Attributes.Blink {
		style = style.Blink(true)
	}

	// Set the cell
	app.screen.SetContent(x, y, cell.Char, nil, style)
}

// convertColor converts terminal color to tcell color
func convertColor(color terminal.Color) tcell.Color {
	switch color {
	case terminal.ColorDefault:
		return tcell.ColorReset // Use terminal default color
	case terminal.ColorBlack:
		return tcell.ColorBlack
	case terminal.ColorRed:
		return tcell.ColorRed
	case terminal.ColorGreen:
		return tcell.ColorGreen
	case terminal.ColorYellow:
		return tcell.ColorYellow
	case terminal.ColorBlue:
		return tcell.ColorBlue
	case terminal.ColorMagenta:
		return tcell.ColorPurple
	case terminal.ColorCyan:
		return tcell.ColorTeal
	case terminal.ColorWhite:
		return tcell.ColorWhite
	case terminal.ColorBrightBlack:
		return tcell.ColorDarkGray
	case terminal.ColorBrightRed:
		return tcell.ColorRed
	case terminal.ColorBrightGreen:
		return tcell.ColorGreen
	case terminal.ColorBrightYellow:
		return tcell.ColorYellow
	case terminal.ColorBrightBlue:
		return tcell.ColorBlue
	case terminal.ColorBrightMagenta:
		return tcell.ColorPurple
	case terminal.ColorBrightCyan:
		return tcell.ColorTeal
	case terminal.ColorBrightWhite:
		return tcell.ColorWhite
	default:
		return tcell.ColorReset // Use terminal default for unknown colors
	}
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// setupMenu initializes the main menu
func (app *Application) setupMenu() {
	// Session Management
	app.mainMenu.AddItem("Clear Screen", "Alt+C", func() error {
		app.logDebug("Menu: Clear Screen")
		app.terminal.Clear()
		app.updateStatusMessage("Screen cleared")
		app.updateDisplay()
		return nil
	})

	app.mainMenu.AddItem("Clear History", "Alt+H", func() error {
		app.logDebug("Menu: Clear History")
		app.terminal.ClearScrollback()
		app.updateStatusMessage("History cleared")
		app.updateDisplay()
		return nil
	})

	app.mainMenu.AddSeparator()

	// File Operations
	app.mainMenu.AddItem("Save Session", "Alt+S", func() error {
		app.logDebug("Menu: Save Session")
		err := app.saveSessionToFile()
		if err != nil {
			app.updateStatusMessage(fmt.Sprintf("Failed: %v", err))
		}
		return err
	})

	app.mainMenu.AddSeparator()

	// Connection
	app.mainMenu.AddItem("Reconnect", "Alt+R", func() error {
		app.logDebug("Menu: Reconnect")
		err := app.reconnect()
		if err != nil {
			app.updateStatusMessage(fmt.Sprintf("Reconnect failed: %v", err))
		}
		return err
	})

	app.mainMenu.AddSeparator()

	// View Control
	lineWrapLabel := "Line Wrap: ON"
	if !app.lineWrap {
		lineWrapLabel = "Line Wrap: OFF"
	}
	app.mainMenu.AddItem(lineWrapLabel, "", func() error {
		app.logDebug("Menu: Toggle Line Wrap")
		app.lineWrap = !app.lineWrap

		// Update menu label
		newLabel := "Line Wrap: ON"
		if !app.lineWrap {
			newLabel = "Line Wrap: OFF"
		}
		idx := app.mainMenu.FindItemIndex("Line Wrap:")
		if idx >= 0 {
			app.mainMenu.UpdateItemLabel(idx, newLabel)
		}

		// Update status message
		if app.lineWrap {
			app.updateStatusMessage("Line wrap: ON")
		} else {
			app.updateStatusMessage("Line wrap: OFF")
		}

		// Update terminal line wrap setting
		if app.terminal != nil {
			app.terminal.SetLineWrap(app.lineWrap)
		}

		// Redraw menu
		app.mainMenu.Draw()
		return nil
	})

	localEchoLabel := "Local Echo: OFF"
	if app.localEcho {
		localEchoLabel = "Local Echo: ON"
	}
	app.mainMenu.AddItem(localEchoLabel, "", func() error {
		app.logDebug("Menu: Toggle Local Echo")
		app.localEcho = !app.localEcho

		// Update menu label
		newLabel := "Local Echo: ON"
		if !app.localEcho {
			newLabel = "Local Echo: OFF"
		}
		idx := app.mainMenu.FindItemIndex("Local Echo:")
		if idx >= 0 {
			app.mainMenu.UpdateItemLabel(idx, newLabel)
		}

		// Update status message
		if app.localEcho {
			app.updateStatusMessage("Local echo: ON")
		} else {
			app.updateStatusMessage("Local echo: OFF")
		}

		// Redraw menu
		app.mainMenu.Draw()
		return nil
	})

	app.mainMenu.AddSeparator()

	// Help
	app.mainMenu.AddItem("About", "", func() error {
		app.logDebug("Menu: About")
		// Show about info in status message
		aboutMsg := fmt.Sprintf("Serial Terminal v%s - Modern terminal emulator", app.config.Version)
		app.updateStatusMessage(aboutMsg)
		return nil
	})

	app.mainMenu.AddItem("Exit Application", "Ctrl+Q", func() error {
		app.logDebug("Menu: Exit")
		app.mainMenu.Hide() // Close menu before exiting
		go func() {
			_ = app.Stop()
		}()
		return nil
	})

	// Set close callback to restore screen and update display
	app.mainMenu.SetOnClose(func() {
		app.overlayMgr.RestoreScreen()
		// Force redraw after menu closes
		app.updateDisplay()
	})
}

// showMainMenu displays the main menu
func (app *Application) showMainMenu() {
	if app.mainMenu == nil || app.overlayMgr == nil {
		return
	}

	// Save current screen content
	app.overlayMgr.SaveScreen()

	// Show menu
	app.mainMenu.Show()
}

// hideMainMenu hides the main menu
func (app *Application) hideMainMenu() {
	if app.mainMenu == nil || app.overlayMgr == nil {
		return
	}

	if app.mainMenu.IsVisible() {
		app.mainMenu.Hide()
		app.overlayMgr.RestoreScreen()
		// Force redraw after hiding menu
		app.updateDisplay()
	}
}

// saveSessionToFile saves the current session to a file
func (app *Application) saveSessionToFile() error {
	// Generate filename with timestamp
	filename := fmt.Sprintf("session_%s.txt", time.Now().Format("20060102_150405"))

	// Create file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write session info
	fmt.Fprintf(file, "Serial Terminal Session\n")
	fmt.Fprintf(file, "========================\n")
	fmt.Fprintf(file, "Date: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Port: %s\n", app.config.SerialConfig.Port)
	fmt.Fprintf(file, "Settings: %d %d-%s-%d\n",
		app.config.SerialConfig.BaudRate,
		app.config.SerialConfig.DataBits,
		app.config.SerialConfig.Parity,
		app.config.SerialConfig.StopBits)
	fmt.Fprintf(file, "========================\n\n")

	// Write terminal content (including scrollback)
	lines := app.terminal.GetAllLines()
	for _, line := range lines {
		for _, cell := range line {
			if cell.Char != 0 {
				fmt.Fprintf(file, "%c", cell.Char)
			}
		}
		fmt.Fprintln(file)
	}

	app.logDebug("Session saved to %s", filename)

	// Show status message
	app.updateStatusMessage(fmt.Sprintf("Session saved to %s", filename))

	return nil
}

// reconnect disconnects and reconnects to the serial port
func (app *Application) reconnect() error {
	app.logDebug("Reconnecting...")

	// Close current connection
	if app.serialPort != nil && app.serialPort.IsOpen() {
		app.serialPort.Close()
	}

	// Small delay
	time.Sleep(500 * time.Millisecond)

	// Reopen connection
	err := app.serialPort.Open(app.config.SerialConfig)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	// Clear terminal
	app.terminal.Clear()

	// Update status
	app.updateStatusMessage("Reconnected successfully")

	return nil
}

// updateStatusMessage shows a temporary status message
func (app *Application) updateStatusMessage(message string) {
	app.statusMessage = message
	app.statusTime = time.Now()
	// Force redraw to show the message
	// Mark terminal as dirty to trigger redraw
	if app.terminal != nil && app.terminal.GetScreen() != nil {
		app.terminal.GetScreen().Dirty = true
	}
	app.updateDisplay()
	// If menu is visible, also redraw it on top
	if app.mainMenu != nil && app.mainMenu.IsVisible() {
		app.mainMenu.Draw()
	}
	app.logDebug("Status: %s", message)
}
