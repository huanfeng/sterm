package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"serial-terminal/pkg/serial"
)

var (
	listDetails bool
	listFormat  string
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available serial ports",
	Long: `List all available serial ports on the system.
	
This command scans the system for available serial ports and displays
them in a formatted list. On different platforms:
  - Windows: Lists COM ports
  - Linux: Lists /dev/tty* devices
  - macOS: Lists /dev/cu.* and /dev/tty.* devices`,
	Aliases: []string{"ls", "ports"},
	Run:     runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "show detailed port information")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "output format (table, csv, json)")
}

func runList(cmd *cobra.Command, args []string) {
	// Get detailed list of available ports
	portInfos, err := serial.GetDetailedPortsList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing ports: %v\n", err)
		os.Exit(1)
	}

	if len(portInfos) == 0 {
		fmt.Println("No serial ports found.")
		return
	}

	// Display based on format
	switch listFormat {
	case "csv":
		printPortsCSV(portInfos)
	case "json":
		printPortsJSON(portInfos)
	default:
		printPortsTable(portInfos)
	}
}

func printPortsTable(portInfos []serial.PortInfo) {
	fmt.Printf("Found %d serial port(s):\n", len(portInfos))

	if listDetails {
		// Show detailed information if available
		for _, portInfo := range portInfos {
			fmt.Printf("  %s", portInfo.Name)

			// Add USB details if available
			if portInfo.IsUSB {
				fmt.Printf(" [USB]")
				if portInfo.VID != "" || portInfo.PID != "" {
					fmt.Printf(" VID:%s PID:%s", portInfo.VID, portInfo.PID)
				}
				if portInfo.Product != "" {
					fmt.Printf(" - %s", portInfo.Product)
				}
				if portInfo.SerialNumber != "" {
					fmt.Printf(" (SN: %s)", portInfo.SerialNumber)
				}
			}
			fmt.Println()
		}
	} else {
		// Simple list with indentation for table format
		for _, portInfo := range portInfos {
			fmt.Printf("  %s\n", portInfo.Name)
		}
	}

	fmt.Println("\nUse 'serial-terminal connect <port>' or 'serial-terminal c <port>' to connect.")
}

func printPortsCSV(portInfos []serial.PortInfo) {
	if listDetails {
		fmt.Println("port,is_usb,vid,pid,product,serial_number")
		for _, portInfo := range portInfos {
			fmt.Printf("%s,%t,%s,%s,%s,%s\n",
				portInfo.Name,
				portInfo.IsUSB,
				portInfo.VID,
				portInfo.PID,
				portInfo.Product,
				portInfo.SerialNumber)
		}
	} else {
		fmt.Println("port")
		for _, portInfo := range portInfos {
			fmt.Printf("%s\n", portInfo.Name)
		}
	}
}

func printPortsJSON(portInfos []serial.PortInfo) {
	if listDetails {
		// Output full JSON objects with details
		fmt.Println("[")
		for i, portInfo := range portInfos {
			fmt.Printf("  {\n")
			fmt.Printf("    \"name\": \"%s\"", portInfo.Name)

			if portInfo.IsUSB {
				fmt.Printf(",\n    \"is_usb\": true")
				if portInfo.VID != "" {
					fmt.Printf(",\n    \"vid\": \"%s\"", portInfo.VID)
				}
				if portInfo.PID != "" {
					fmt.Printf(",\n    \"pid\": \"%s\"", portInfo.PID)
				}
				if portInfo.Product != "" {
					fmt.Printf(",\n    \"product\": \"%s\"", portInfo.Product)
				}
				if portInfo.SerialNumber != "" {
					fmt.Printf(",\n    \"serial_number\": \"%s\"", portInfo.SerialNumber)
				}
			}

			fmt.Printf("\n  }")
			if i < len(portInfos)-1 {
				fmt.Printf(",")
			}
			fmt.Printf("\n")
		}
		fmt.Println("]")
	} else {
		// Simple array of port names
		fmt.Println("[")
		for i, portInfo := range portInfos {
			fmt.Printf("  \"%s\"", portInfo.Name)

			if i < len(portInfos)-1 {
				fmt.Printf(",\n")
			} else {
				fmt.Printf("\n")
			}
		}
		fmt.Println("]")
	}
}
