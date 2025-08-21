package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Serial Terminal Application")
	fmt.Println("Version: 0.1.0")
	
	// TODO: Initialize application components
	// This will be implemented in later tasks
	
	if len(os.Args) > 1 {
		fmt.Printf("Arguments: %v\n", os.Args[1:])
	}
}