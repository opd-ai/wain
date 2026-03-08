package demo

import (
	"fmt"
	"os"
	"strings"
)

// PrintUsageAndExit displays usage information and exits the program.
// This provides consistent help output across all demonstration binaries.
func PrintUsageAndExit(name, description string, examples []string) {
	fmt.Printf("%s - %s\n\n", name, description)
	fmt.Println("Usage:")
	for _, example := range examples {
		fmt.Printf("  %s\n", example)
	}
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/opd-ai/wain")
	os.Exit(0)
}

// CheckHelpFlag checks if --help or -h was passed and calls PrintUsageAndExit if so.
func CheckHelpFlag(name, description string, examples []string) {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "help" {
			PrintUsageAndExit(name, description, examples)
		}
	}
}

// FormatExample creates a formatted usage example string.
func FormatExample(command, comment string) string {
	if comment == "" {
		return command
	}
	padding := strings.Repeat(" ", max(0, 40-len(command)))
	return fmt.Sprintf("%s%s# %s", command, padding, comment)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
