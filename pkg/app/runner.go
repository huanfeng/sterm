package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sterm/pkg/serial"
)

// Runner provides a high-level interface to run the terminal application
type Runner struct {
	app    *Application
	config AppConfig
}

// NewRunner creates a new application runner
func NewRunner(serialConfig serial.SerialConfig) (*Runner, error) {
	// Create app config
	appConfig := DefaultAppConfig()
	appConfig.SerialConfig = serialConfig

	// Don't set fixed size - let the app detect from actual terminal
	// Setting to 0 will make the app use the actual terminal size
	appConfig.TerminalWidth = 0
	appConfig.TerminalHeight = 0

	return &Runner{
		config: appConfig,
	}, nil
}

// Run starts the application and blocks until it's stopped
func (r *Runner) Run() error {
	// Create application
	app, err := NewApplication(r.config)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	r.app = app

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start application
	if err := app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Print session info
	fmt.Printf("\n=== Serial Terminal Session Started ===\n")
	fmt.Printf("Port: %s\n", r.config.SerialConfig.Port)
	fmt.Printf("Settings: %d %d-%s-%d\n",
		r.config.SerialConfig.BaudRate,
		r.config.SerialConfig.DataBits,
		string(r.config.SerialConfig.Parity[0]),
		r.config.SerialConfig.StopBits)
	fmt.Printf("Press Ctrl+Shift+Q (or Ctrl+Q) to exit\n")
	fmt.Printf("Press F1 for help\n")
	fmt.Printf("=====================================\n\n")

	// Wait for signal or application to stop
	select {
	case <-sigChan:
		fmt.Println("\nReceived interrupt signal, shutting down...")
	case <-r.waitForStop():
		fmt.Println("\nApplication stopped")
	}

	// Stop application
	if err := app.Stop(); err != nil {
		return fmt.Errorf("failed to stop application: %w", err)
	}

	// Print session summary
	r.printSessionSummary()

	return nil
}

// waitForStop returns a channel that closes when the application stops
func (r *Runner) waitForStop() <-chan struct{} {
	stopChan := make(chan struct{})

	go func() {
		for r.app != nil && r.app.IsRunning() {
			// Check every 100ms
			time.Sleep(100 * time.Millisecond)
		}
		close(stopChan)
	}()

	return stopChan
}

// printSessionSummary prints a summary of the session
func (r *Runner) printSessionSummary() {
	if r.app == nil {
		return
	}

	bytesSent, bytesRecv, duration := r.app.GetStats()

	fmt.Printf("\n=== Session Summary ===\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Bytes Sent: %d\n", bytesSent)
	fmt.Printf("Bytes Received: %d\n", bytesRecv)
	fmt.Printf("=====================\n")
}

// Stop stops the running application
func (r *Runner) Stop() error {
	if r.app != nil {
		return r.app.Stop()
	}
	return nil
}

// AppOptions contains runtime options for the application
type AppOptions struct {
	SendWindowSize bool
	TerminalType   string
	DebugMode      bool
}

// RunInteractive runs the application in interactive mode with a UI
func RunInteractive(serialConfig serial.SerialConfig) error {
	runner, err := NewRunner(serialConfig)
	if err != nil {
		return err
	}

	return runner.Run()
}

// RunInteractiveWithOptions runs the application with additional options
func RunInteractiveWithOptions(serialConfig serial.SerialConfig, opts AppOptions) error {
	// Create app config
	appConfig := DefaultAppConfig()
	appConfig.SerialConfig = serialConfig

	// Apply options
	appConfig.SendWindowSizeOnConnect = opts.SendWindowSize
	appConfig.SendWindowSizeOnResize = opts.SendWindowSize
	appConfig.DebugMode = opts.DebugMode
	if opts.TerminalType != "" {
		appConfig.TerminalType = opts.TerminalType
	}

	// Don't set fixed size - let the app detect from actual terminal
	appConfig.TerminalWidth = 0
	appConfig.TerminalHeight = 0

	runner := &Runner{
		config: appConfig,
	}

	return runner.Run()
}

// RunHeadless runs the application in headless mode (no UI, just logging)
func RunHeadless(serialConfig serial.SerialConfig, logFile string) error {
	// Create app config
	appConfig := DefaultAppConfig()
	appConfig.SerialConfig = serialConfig

	// Create application
	app, err := NewApplication(appConfig)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	// Open log file if specified
	var logWriter *os.File
	if logFile != "" {
		logWriter, err = os.Create(logFile)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		defer logWriter.Close()
	}

	// Start application
	if err := app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}
	defer func() { _ = app.Stop() }()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run until interrupted
	<-sigChan

	return nil
}
