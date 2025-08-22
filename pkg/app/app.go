// Package app provides the main application controller
package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"serial-terminal/pkg/config"
	"serial-terminal/pkg/history"
	"serial-terminal/pkg/serial"
	"serial-terminal/pkg/terminal"
)

// Application represents the main application controller
type Application struct {
	// Core components
	serialPort serial.SerialPort
	terminal   *terminal.TerminalEmulator
	configMgr  config.ConfigManager
	historyMgr history.HistoryManager

	// UI components
	screen    tcell.Screen
	shortcuts *terminal.ShortcutManager

	// Session management
	session *Session

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// State
	isRunning bool
	isPaused  bool

	// Configuration
	config AppConfig

	// Debug
	debugLog *os.File
}

// AppConfig contains application configuration
type AppConfig struct {
	SerialConfig    serial.SerialConfig
	TerminalWidth   int
	TerminalHeight  int
	HistorySize     int
	EnableMouse     bool
	EnableShortcuts bool
	SaveHistory     bool
	HistoryFormat   history.FileFormat
}

// DefaultAppConfig returns default application configuration
func DefaultAppConfig() AppConfig {
	return AppConfig{
		SerialConfig:    serial.DefaultConfig(),
		TerminalWidth:   80,
		TerminalHeight:  24,
		HistorySize:     10 * 1024 * 1024, // 10MB
		EnableMouse:     true,
		EnableShortcuts: true,
		SaveHistory:     true,
		HistoryFormat:   history.FormatTimestamped,
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
		app.debugLog.Sync() // Ensure it's written immediately
	}
}

// NewApplication creates a new application instance
func NewApplication(config AppConfig) (*Application, error) {
	// Validate configuration
	if err := config.SerialConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid serial config: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create debug log file (optional, won't fail if can't create)
	debugLog, _ := os.Create("serial-terminal-debug.log")

	// Create components
	app := &Application{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
		isPaused:  false,
		debugLog:  debugLog,
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

	app.screen = screen

	// Get terminal dimensions
	width, height := screen.Size()
	if app.config.TerminalWidth > 0 && app.config.TerminalHeight > 0 {
		width = app.config.TerminalWidth
		height = app.config.TerminalHeight
	}

	// Create terminal emulator
	app.terminal = terminal.NewTerminalEmulator(
		nil, // Will set input/output later
		nil,
		width,
		height,
	)

	// Create shortcut manager
	app.shortcuts = terminal.NewShortcutManager()
	app.setupShortcuts()

	return nil
}

// setupShortcuts sets up application shortcuts
func (app *Application) setupShortcuts() {
	// Exit shortcut - use Ctrl+Shift+Q to avoid conflict with terminal
	app.shortcuts.SetShortcutHandler("exit", func() error {
		return app.Stop()
	})

	// Save history shortcut
	app.shortcuts.SetShortcutHandler("save", func() error {
		return app.SaveHistory("")
	})

	// Clear screen shortcut
	app.shortcuts.SetShortcutHandler("clear", func() error {
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
	app.shortcuts.SetShortcutHandler("disconnect", func() error {
		return app.Disconnect()
	})

	// Help shortcut
	app.shortcuts.SetShortcutHandler("help", func() error {
		app.ShowHelp()
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

	// Start data flow goroutines
	app.wg.Add(3)
	go app.handleSerialInput()
	go app.handleSerialOutput()
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
		app.screen.PostEvent(tcell.NewEventResize(0, 0))
	}

	// Close serial port first to stop I/O
	if app.serialPort != nil && app.serialPort.IsOpen() {
		app.logDebug("Closing serial port")
		app.serialPort.Close()
	}

	// Stop terminal
	if app.terminal != nil {
		app.terminal.Stop()
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

	// Save history if configured
	if app.config.SaveHistory && app.historyMgr != nil && app.session != nil {
		filename := fmt.Sprintf("session_%s.log", app.session.ID)
		app.historyMgr.SaveToFile(filename, app.config.HistoryFormat)
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

	buffer := make([]byte, 4096)

	for {
		select {
		case <-app.ctx.Done():
			return
		default:
			if app.isPaused {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Read from serial port
			n, err := app.serialPort.Read(buffer)
			if err != nil {
				// Log error but continue
				continue
			}

			if n > 0 {
				data := buffer[:n]

				// Process in terminal
				app.terminal.ProcessOutput(data)

				// Save to history
				if app.historyMgr != nil {
					app.historyMgr.Write(data, history.DirectionOutput)
				}

				// Update session stats
				if app.session != nil {
					app.session.UpdateStats(0, int64(n))
				}
			}
		}
	}
}

// handleSerialOutput reads data from terminal and sends to serial port
func (app *Application) handleSerialOutput() {
	defer app.wg.Done()

	// This would typically receive data from terminal input
	// For now, we'll handle it in handleUserInput
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
				app.screen.PostEvent(tcell.NewEventResize(0, 0))
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
	// Debug log ALL key events
	if ev.Key() == tcell.KeyRune {
		app.logDebug("Key: Rune='%c'(0x%x), Mods=%v", ev.Rune(), ev.Rune(), ev.Modifiers())
	} else {
		app.logDebug("Key: Key=%v, Mods=%v", ev.Key(), ev.Modifiers())
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

	// Check for F1 help key
	if ev.Key() == tcell.KeyF1 {
		app.logDebug("F1 help key pressed")
		// TODO: Show help
	}

	// Check for F8 pause/resume
	if ev.Key() == tcell.KeyF8 {
		app.logDebug("F8 pause/resume key pressed")
		if app.isPaused {
			app.Resume()
		} else {
			app.Pause()
		}
		return
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

	// Process as terminal input
	inputProcessor := terminal.NewInputProcessor(app.terminal)
	data := inputProcessor.ProcessKeyEvent(ev)

	if len(data) > 0 && !app.isPaused {
		// Send to serial port
		if app.serialPort != nil && app.serialPort.IsOpen() {
			n, _ := app.serialPort.Write(data)

			// Save to history
			if app.historyMgr != nil {
				app.historyMgr.Write(data[:n], history.DirectionInput)
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
	if !app.config.EnableMouse {
		return
	}

	inputProcessor := terminal.NewInputProcessor(app.terminal)
	data := inputProcessor.ProcessMouseEvent(ev)

	if len(data) > 0 && !app.isPaused {
		// Send to serial port
		if app.serialPort != nil && app.serialPort.IsOpen() {
			app.serialPort.Write(data)
		}
	}
}

// handleResize handles terminal resize events
func (app *Application) handleResize() {
	width, height := app.screen.Size()
	app.terminal.Resize(width, height)
	app.screen.Clear()
	app.updateDisplay()
}

// updateUI updates the terminal display
func (app *Application) updateUI() {
	defer app.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			app.updateDisplay()
		}
	}
}

// updateDisplay updates the screen with terminal content
func (app *Application) updateDisplay() {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if !app.isRunning || app.screen == nil || app.terminal == nil {
		return
	}

	// Get terminal screen buffer
	screen := app.terminal.GetScreen()
	if screen == nil || !screen.Dirty {
		return
	}

	// Clear tcell screen
	app.screen.Clear()

	// Get terminal state
	state := app.terminal.GetState()

	// Render each cell
	for y := 0; y < screen.Height && y < len(screen.Buffer); y++ {
		for x := 0; x < screen.Width && x < len(screen.Buffer[y]); x++ {
			cell := screen.Buffer[y][x]

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
	}

	// Show cursor
	if state.CursorX >= 0 && state.CursorX < screen.Width &&
		state.CursorY >= 0 && state.CursorY < screen.Height {
		app.screen.ShowCursor(state.CursorX, state.CursorY)
	}

	// Show the screen
	app.screen.Show()

	// Mark screen as clean
	screen.Dirty = false
}

// Pause pauses data flow
func (app *Application) Pause() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.isRunning {
		return fmt.Errorf("application is not running")
	}

	app.isPaused = true
	return nil
}

// Resume resumes data flow
func (app *Application) Resume() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.isRunning {
		return fmt.Errorf("application is not running")
	}

	app.isPaused = false
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

	// Send clear screen sequence
	clearSeq := []byte{0x1B, '[', '2', 'J', 0x1B, '[', 'H'}
	return app.terminal.ProcessOutput(clearSeq)
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

// ShowHelp displays help information in the terminal
func (app *Application) ShowHelp() {
	helpText := `
╔════════════════════════════════════════╗
║        Serial Terminal Help            ║
╠════════════════════════════════════════╣
║ Shortcuts:                             ║
║   Ctrl+Shift+Q : Exit                  ║
║   (or Ctrl+Q)                          ║
║   Ctrl+Shift+S : Save history          ║
║   Ctrl+Shift+C : Clear screen          ║
║   F1          : Show this help         ║
║   F8          : Pause/Resume data flow ║
╠════════════════════════════════════════╣
║ Press any key to continue...           ║
╚════════════════════════════════════════╝
`
	// Save current screen content
	width, height := app.screen.Size()

	// Clear and show help
	app.screen.Clear()
	lines := strings.Split(helpText, "\n")
	startY := (height - len(lines)) / 2
	if startY < 0 {
		startY = 0
	}

	for i, line := range lines {
		if startY+i >= height {
			break
		}
		startX := (width - len(line)) / 2
		if startX < 0 {
			startX = 0
		}
		for j, ch := range line {
			if startX+j < width {
				app.screen.SetContent(startX+j, startY+i, ch, nil, tcell.StyleDefault)
			}
		}
	}
	app.screen.Show()

	// Wait for any key
	app.screen.PollEvent()

	// Restore screen
	app.updateDisplay()
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
