package cmd

import (
	"fmt"
	"os"
	"time"
	
	"github.com/spf13/cobra"
	"serial-terminal/pkg/app"
	"serial-terminal/pkg/config"
	"serial-terminal/pkg/serial"
)

var (
	// Root command flags
	cfgFile     string
	port        string
	baudRate    int
	dataBits    int
	stopBits    int
	parity      string
	timeout     int
	verbose     bool
	
	// Root command
	rootCmd = &cobra.Command{
		Use:   "serial-terminal",
		Short: "A cross-platform serial port terminal emulator",
		Long: `Serial Terminal is a powerful command-line tool that provides terminal emulation
for serial port communication. It supports VT100/ANSI terminal features, configuration
management, and comprehensive history recording.

Features:
  - Full VT100/ANSI terminal emulation
  - Mouse support for terminal applications
  - Configuration profiles for quick connections
  - History recording with multiple export formats
  - Cross-platform support (Windows, Linux, macOS)`,
		Version: "1.0.0",
		Run:     runTerminal,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	
	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	
	// Local flags (only available to root command)
	rootCmd.Flags().StringVarP(&port, "port", "p", "", "serial port to connect to")
	rootCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "baud rate")
	rootCmd.Flags().IntVarP(&dataBits, "data", "d", 8, "data bits (5, 6, 7, or 8)")
	rootCmd.Flags().IntVarP(&stopBits, "stop", "s", 1, "stop bits (1 or 2)")
	rootCmd.Flags().StringVar(&parity, "parity", "none", "parity (none, odd, even, mark, space)")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "read timeout in seconds")
	
	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(connectCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		if verbose {
			fmt.Printf("Using config file: %s\n", cfgFile)
		}
		// Load configuration here
	}
}

// runTerminal is the main entry point for the terminal
func runTerminal(cmd *cobra.Command, args []string) {
	// Check if port is specified
	if port == "" && cfgFile == "" {
		// No port or config specified, show available ports
		ports, err := serial.ListPorts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing ports: %v\n", err)
			os.Exit(1)
		}
		
		if len(ports) == 0 {
			fmt.Println("No serial ports found.")
			fmt.Println("\nUse 'serial-terminal --help' for usage information.")
		} else {
			fmt.Println("Available serial ports:")
			for _, p := range ports {
				fmt.Printf("  - %s\n", p)
			}
			fmt.Println("\nUse 'serial-terminal -p <port>' to connect to a port.")
			fmt.Println("Use 'serial-terminal --help' for more options.")
		}
		return
	}
	
	// Create serial configuration
	var serialConfig serial.SerialConfig
	
	if cfgFile != "" {
		// Load configuration from file
		configManager := config.NewFileConfigManager("")
		cfg, err := configManager.LoadConfig(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		serialConfig = cfg
	} else {
		// Use command line parameters
		serialConfig = serial.SerialConfig{
			Port:     port,
			BaudRate: baudRate,
			DataBits: dataBits,
			StopBits: stopBits,
			Parity:   parity,
			Timeout:  time.Duration(timeout) * time.Second,
		}
		
		// Validate configuration
		if err := serialConfig.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Start terminal application
	if verbose {
		fmt.Printf("Connecting to %s at %d baud...\n", serialConfig.Port, serialConfig.BaudRate)
	}
	
	// Launch the terminal application
	if err := app.RunInteractive(serialConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error running terminal: %v\n", err)
		os.Exit(1)
	}
}