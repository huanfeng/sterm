package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"
	
	"github.com/spf13/cobra"
	"serial-terminal/pkg/app"
	"serial-terminal/pkg/config"
	"serial-terminal/pkg/serial"
)

var (
	// Config command flags
	configPort     string
	configBaudRate int
	configDataBits int
	configStopBits int
	configParity   string
	configTimeout  int
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage serial port configurations",
	Long: `Manage saved serial port configurations.
	
This command allows you to save, load, list, and delete serial port
configurations for quick access to frequently used settings.`,
}

// saveCmd saves a configuration
var saveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a serial port configuration",
	Long: `Save the current serial port configuration with a given name.
	
Example:
  serial-terminal config save mydevice -p COM3 -b 115200`,
	Args: cobra.ExactArgs(1),
	Run:  runSaveConfig,
}

// loadCmd loads a configuration
var loadCmd = &cobra.Command{
	Use:   "load <name>",
	Short: "Load and connect using a saved configuration",
	Long: `Load a saved configuration and immediately connect to the serial port.
	
Example:
  serial-terminal config load mydevice`,
	Args: cobra.ExactArgs(1),
	Run:  runLoadConfig,
}

// listConfigCmd lists all configurations
var listConfigCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved configurations",
	Long:  `Display a list of all saved serial port configurations.`,
	Run:   runListConfigs,
}

// deleteCmd deletes a configuration
var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a saved configuration",
	Long: `Delete a saved serial port configuration.
	
Example:
  serial-terminal config delete mydevice`,
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	Run:     runDeleteConfig,
}

// showCmd shows details of a configuration
var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a saved configuration",
	Long: `Display detailed information about a saved configuration.
	
Example:
  serial-terminal config show mydevice`,
	Args: cobra.ExactArgs(1),
	Run:  runShowConfig,
}

func init() {
	// Add subcommands to config
	configCmd.AddCommand(saveCmd)
	configCmd.AddCommand(loadCmd)
	configCmd.AddCommand(listConfigCmd)
	configCmd.AddCommand(deleteCmd)
	configCmd.AddCommand(showCmd)
	
	// Add flags for save command
	saveCmd.Flags().StringVarP(&configPort, "port", "p", "", "serial port")
	saveCmd.Flags().IntVarP(&configBaudRate, "baud", "b", 115200, "baud rate")
	saveCmd.Flags().IntVarP(&configDataBits, "data", "d", 8, "data bits")
	saveCmd.Flags().IntVarP(&configStopBits, "stop", "s", 1, "stop bits")
	saveCmd.Flags().StringVar(&configParity, "parity", "none", "parity")
	saveCmd.Flags().IntVarP(&configTimeout, "timeout", "t", 10, "timeout in seconds")
	saveCmd.MarkFlagRequired("port")
}

func runSaveConfig(cmd *cobra.Command, args []string) {
	name := args[0]
	
	// Create configuration
	cfg := serial.SerialConfig{
		Port:     configPort,
		BaudRate: configBaudRate,
		DataBits: configDataBits,
		StopBits: configStopBits,
		Parity:   configParity,
		Timeout:  time.Duration(configTimeout) * time.Second,
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Save configuration
	configManager := config.NewFileConfigManager("")
	if err := configManager.SaveConfig(name, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving configuration: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Configuration '%s' saved successfully.\n", name)
	fmt.Printf("  Port: %s\n", cfg.Port)
	fmt.Printf("  Baud Rate: %d\n", cfg.BaudRate)
	fmt.Printf("  Data Bits: %d\n", cfg.DataBits)
	fmt.Printf("  Stop Bits: %d\n", cfg.StopBits)
	fmt.Printf("  Parity: %s\n", cfg.Parity)
	fmt.Printf("  Timeout: %v\n", cfg.Timeout)
}

func runLoadConfig(cmd *cobra.Command, args []string) {
	name := args[0]
	
	// Load configuration
	configManager := config.NewFileConfigManager("")
	cfg, err := configManager.LoadConfig(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration '%s': %v\n", name, err)
		os.Exit(1)
	}
	
	fmt.Printf("Loading configuration '%s'...\n", name)
	fmt.Printf("Connecting to %s at %d baud...\n", cfg.Port, cfg.BaudRate)
	
	// Launch terminal with loaded configuration
	if err := app.RunInteractive(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error running terminal: %v\n", err)
		os.Exit(1)
	}
}

func runListConfigs(cmd *cobra.Command, args []string) {
	// List all configurations
	configManager := config.NewFileConfigManager("")
	configs, err := configManager.ListConfigs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing configurations: %v\n", err)
		os.Exit(1)
	}
	
	if len(configs) == 0 {
		fmt.Println("No saved configurations found.")
		fmt.Println("\nUse 'serial-terminal config save <name>' to save a configuration.")
		return
	}
	
	fmt.Printf("Found %d saved configuration(s):\n\n", len(configs))
	
	// Create a tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPORT\tBAUD\tLAST USED\tCREATED")
	fmt.Fprintln(w, "----\t----\t----\t---------\t-------")
	
	for _, cfg := range configs {
		lastUsed := "Never"
		if !cfg.LastUsedAt.IsZero() {
			lastUsed = cfg.LastUsedAt.Format("2006-01-02 15:04")
		}
		
		created := cfg.CreatedAt.Format("2006-01-02 15:04")
		
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			cfg.Name,
			cfg.Config.Port,
			cfg.Config.BaudRate,
			lastUsed,
			created)
	}
	
	w.Flush()
	
	fmt.Println("\nUse 'serial-terminal config load <name>' to connect using a configuration.")
	fmt.Println("Use 'serial-terminal config show <name>' to see full details.")
}

func runDeleteConfig(cmd *cobra.Command, args []string) {
	name := args[0]
	
	// Delete configuration
	configManager := config.NewFileConfigManager("")
	if err := configManager.DeleteConfig(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting configuration '%s': %v\n", name, err)
		os.Exit(1)
	}
	
	fmt.Printf("Configuration '%s' deleted successfully.\n", name)
}

func runShowConfig(cmd *cobra.Command, args []string) {
	name := args[0]
	
	// Load configuration info
	configManager := config.NewFileConfigManager("")
	configs, err := configManager.ListConfigs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configurations: %v\n", err)
		os.Exit(1)
	}
	
	// Find the specific configuration
	var found *config.ConfigInfo
	for _, cfg := range configs {
		if cfg.Name == name {
			found = &cfg
			break
		}
	}
	
	if found == nil {
		fmt.Fprintf(os.Stderr, "Configuration '%s' not found.\n", name)
		os.Exit(1)
	}
	
	// Display configuration details
	fmt.Printf("Configuration: %s\n", found.Name)
	fmt.Println("=" + repeatString("=", len(found.Name)+14))
	fmt.Printf("Port:        %s\n", found.Config.Port)
	fmt.Printf("Baud Rate:   %d\n", found.Config.BaudRate)
	fmt.Printf("Data Bits:   %d\n", found.Config.DataBits)
	fmt.Printf("Stop Bits:   %d\n", found.Config.StopBits)
	fmt.Printf("Parity:      %s\n", found.Config.Parity)
	fmt.Printf("Timeout:     %d seconds\n", found.Config.Timeout)
	fmt.Println()
	fmt.Printf("Created:     %s\n", found.CreatedAt.Format(time.RFC3339))
	
	if !found.LastUsedAt.IsZero() {
		fmt.Printf("Last Used:   %s\n", found.LastUsedAt.Format(time.RFC3339))
	} else {
		fmt.Printf("Last Used:   Never\n")
	}
	
	fmt.Println("\nUse 'serial-terminal config load " + name + "' to connect using this configuration.")
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}