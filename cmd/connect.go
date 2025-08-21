package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"
	
	"github.com/spf13/cobra"
	"serial-terminal/pkg/app"
	"serial-terminal/pkg/config"
	"serial-terminal/pkg/serial"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect <port|config>",
	Short: "Connect to a serial port",
	Long: `Connect to a serial port directly or using a saved configuration.
	
You can specify either:
  - A port name (e.g., COM3, /dev/ttyUSB0) with optional parameters
  - A saved configuration name
	
Examples:
  # Connect to COM3 with default settings
  serial-terminal connect COM3
  
  # Connect to /dev/ttyUSB0 with custom baud rate
  serial-terminal connect /dev/ttyUSB0 -b 9600
  
  # Connect using a saved configuration
  serial-terminal connect mydevice`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"open"},
	Run:     runConnect,
}

func init() {
	// Add flags for connect command
	connectCmd.Flags().IntVarP(&baudRate, "baud", "b", 115200, "baud rate")
	connectCmd.Flags().IntVarP(&dataBits, "data", "d", 8, "data bits (5, 6, 7, or 8)")
	connectCmd.Flags().IntVarP(&stopBits, "stop", "s", 1, "stop bits (1 or 2)")
	connectCmd.Flags().StringVar(&parity, "parity", "none", "parity (none, odd, even, mark, space)")
	connectCmd.Flags().IntVarP(&timeout, "timeout", "t", 10, "read timeout in seconds")
}

func runConnect(cmd *cobra.Command, args []string) {
	target := args[0]
	var serialConfig serial.SerialConfig
	
	// Check if target is a port or a configuration name
	if isSerialPort(target) {
		// Direct port connection
		serialConfig = serial.SerialConfig{
			Port:     target,
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
		
		if verbose {
			fmt.Printf("Connecting to port %s...\n", target)
			fmt.Printf("  Baud Rate: %d\n", baudRate)
			fmt.Printf("  Data Bits: %d\n", dataBits)
			fmt.Printf("  Stop Bits: %d\n", stopBits)
			fmt.Printf("  Parity: %s\n", parity)
		}
	} else {
		// Try to load as configuration
		configManager := config.NewFileConfigManager("")
		cfg, err := configManager.LoadConfig(target)
		if err != nil {
			// Not a valid configuration, check if it might be a port
			// that doesn't exist yet
			fmt.Fprintf(os.Stderr, "Error: '%s' is neither a valid port nor a saved configuration.\n", target)
			fmt.Fprintf(os.Stderr, "\nAvailable ports:\n")
			
			// List available ports
			ports, _ := serial.ListPorts()
			if len(ports) == 0 {
				fmt.Fprintf(os.Stderr, "  No serial ports found.\n")
			} else {
				for _, p := range ports {
					fmt.Fprintf(os.Stderr, "  - %s\n", p)
				}
			}
			
			// List available configurations
			configs, _ := configManager.ListConfigs()
			if len(configs) > 0 {
				fmt.Fprintf(os.Stderr, "\nAvailable configurations:\n")
				for _, c := range configs {
					fmt.Fprintf(os.Stderr, "  - %s (port: %s)\n", c.Name, c.Config.Port)
				}
			}
			
			os.Exit(1)
		}
		
		serialConfig = cfg
		
		if verbose {
			fmt.Printf("Loading configuration '%s'...\n", target)
			fmt.Printf("  Port: %s\n", cfg.Port)
			fmt.Printf("  Baud Rate: %d\n", cfg.BaudRate)
		}
		
		// Update last used time
		configManager.UpdateLastUsed(target)
	}
	
	// Test connection
	testConnection(serialConfig)
	
	// Launch terminal UI
	fmt.Println("\nStarting terminal session...")
	fmt.Println("Press Ctrl+Shift+Q to exit (customizable in settings)")
	
	if err := app.RunInteractive(serialConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error running terminal: %v\n", err)
		os.Exit(1)
	}
}

func isSerialPort(name string) bool {
	// Check if the name looks like a serial port
	lower := strings.ToLower(name)
	
	// Windows COM ports
	if strings.HasPrefix(lower, "com") {
		return true
	}
	
	// Unix-like serial devices
	if strings.HasPrefix(name, "/dev/") {
		return true
	}
	
	// Check if it exists in the list of available ports
	ports, err := serial.ListPorts()
	if err == nil {
		for _, port := range ports {
			if strings.EqualFold(port, name) {
				return true
			}
		}
	}
	
	return false
}

func testConnection(cfg serial.SerialConfig) {
	fmt.Printf("\nTesting connection to %s...\n", cfg.Port)
	
	// Try to open the port
	sp := serial.NewSerialPort()
	err := sp.Open(cfg)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: Failed to open serial port: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPossible solutions:\n")
		
		// Provide helpful error messages based on the error
		errStr := err.Error()
		if strings.Contains(errStr, "permission") || strings.Contains(errStr, "access") {
			fmt.Fprintf(os.Stderr, "  - Check if you have permission to access the port\n")
			fmt.Fprintf(os.Stderr, "  - On Linux: Add your user to the 'dialout' group: sudo usermod -a -G dialout $USER\n")
			fmt.Fprintf(os.Stderr, "  - On macOS: Check System Preferences > Security & Privacy\n")
		}
		
		if strings.Contains(errStr, "busy") || strings.Contains(errStr, "use") {
			fmt.Fprintf(os.Stderr, "  - The port may be in use by another application\n")
			fmt.Fprintf(os.Stderr, "  - Close other terminal programs or serial monitors\n")
		}
		
		if strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such") {
			fmt.Fprintf(os.Stderr, "  - The specified port does not exist\n")
			fmt.Fprintf(os.Stderr, "  - Use 'serial-terminal list' to see available ports\n")
		}
		
		os.Exit(1)
	}
	
	// Successfully opened
	fmt.Println("✓ Connection successful!")
	
	// Get port configuration for display
	actualConfig := sp.GetConfig()
	fmt.Printf("  Port: %s\n", actualConfig.Port)
	fmt.Printf("  Settings: %d %d-%s-%d\n", 
		actualConfig.BaudRate,
		actualConfig.DataBits,
		string(actualConfig.Parity[0]),
		actualConfig.StopBits)
	
	// Close the test connection
	sp.Close()
}