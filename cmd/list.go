package cmd

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"
	
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
	// Get list of available ports
	ports, err := serial.ListPorts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing ports: %v\n", err)
		os.Exit(1)
	}
	
	if len(ports) == 0 {
		fmt.Println("No serial ports found.")
		return
	}
	
	// Display based on format
	switch listFormat {
	case "csv":
		printPortsCSV(ports)
	case "json":
		printPortsJSON(ports)
	default:
		printPortsTable(ports)
	}
}

func printPortsTable(ports []string) {
	fmt.Printf("Found %d serial port(s) on %s:\n\n", len(ports), runtime.GOOS)
	
	if listDetails {
		// Create a tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PORT\tSTATUS\tDESCRIPTION")
		fmt.Fprintln(w, "----\t------\t-----------")
		
		for _, port := range ports {
			status := "Available"
			description := getPortDescription(port)
			
			// Try to open port to check if it's in use
			sp := serial.NewSerialPort()
			err := sp.Open(serial.SerialConfig{
				Port:     port,
				BaudRate: 9600,
				DataBits: 8,
				StopBits: 1,
				Parity:   "none",
				Timeout:  1,
			})
			
			if err != nil {
				status = "In Use"
			} else {
				sp.Close()
			}
			
			fmt.Fprintf(w, "%s\t%s\t%s\n", port, status, description)
		}
		
		w.Flush()
	} else {
		// Simple list
		for _, port := range ports {
			fmt.Printf("  %s\n", port)
		}
	}
	
	fmt.Println("\nUse 'serial-terminal connect <port>' to connect to a specific port.")
}

func printPortsCSV(ports []string) {
	fmt.Println("port,status,description")
	for _, port := range ports {
		status := "available"
		
		// Try to open port to check if it's in use
		sp := serial.NewSerialPort()
		err := sp.Open(serial.SerialConfig{
			Port:     port,
			BaudRate: 9600,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  1,
		})
		
		if err != nil {
			status = "in_use"
		} else {
			sp.Close()
		}
		
		fmt.Printf("%s,%s,%s\n", port, status, getPortDescription(port))
	}
}

func printPortsJSON(ports []string) {
	fmt.Println("[")
	for i, port := range ports {
		status := "available"
		
		// Try to open port to check if it's in use
		sp := serial.NewSerialPort()
		err := sp.Open(serial.SerialConfig{
			Port:     port,
			BaudRate: 9600,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  1,
		})
		
		if err != nil {
			status = "in_use"
		} else {
			sp.Close()
		}
		
		fmt.Printf("  {\n")
		fmt.Printf("    \"port\": \"%s\",\n", port)
		fmt.Printf("    \"status\": \"%s\",\n", status)
		fmt.Printf("    \"description\": \"%s\"\n", getPortDescription(port))
		
		if i < len(ports)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

func getPortDescription(port string) string {
	// This is a placeholder - in a real implementation, we would
	// query system information to get device descriptions
	switch runtime.GOOS {
	case "windows":
		return "USB Serial Device"
	case "linux":
		if len(port) > 8 && port[:8] == "/dev/tty" {
			if port[8:9] == "U" {
				return "USB Serial Device"
			}
			if port[8:9] == "S" {
				return "System Serial Port"
			}
		}
		return "Serial Device"
	case "darwin":
		if len(port) > 8 {
			if port[:8] == "/dev/cu." {
				return "USB Serial Device (Callout)"
			}
			if port[:9] == "/dev/tty." {
				return "USB Serial Device (TTY)"
			}
		}
		return "Serial Device"
	default:
		return "Serial Device"
	}
}