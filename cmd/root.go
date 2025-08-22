package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Root command flags
	verbose bool

	// Root command
	rootCmd = &cobra.Command{
		Use:               "serial-terminal",
		Short:             "A cross-platform serial port terminal emulator",
		Version:           "1.0.0",
		Run:               runTerminal,
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(connectCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	// Can be extended in the future for global configuration
}

// runTerminal is the main entry point for the terminal
func runTerminal(cmd *cobra.Command, args []string) {
	// Always show help when root command is called without subcommands
	cmd.Help()
}
