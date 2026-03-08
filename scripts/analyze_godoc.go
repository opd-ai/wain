package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MethodInfo holds metadata about a method definition in a Go source file.
type MethodInfo struct {
	filename   string
	lineNum    int
	receiver   string
	name       string
	signature  string
	hasComment bool
}

func analyzeFile(filepath string) ([]MethodInfo, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var methods []MethodInfo
	lineNum := 0
	var prevLine string

	methodRegex := regexp.MustCompile(`^func\s+\(([a-z][a-zA-Z0-9_]*)\s+\*?([A-Z][a-zA-Z0-9_]*)\)\s+([A-Z][a-zA-Z0-9_]*)\s*\(`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := methodRegex.FindStringSubmatch(line); matches != nil {
			receiver := matches[2]
			methodName := matches[3]

			// Check if previous line (or 1-2 lines back) has a comment
			hasComment := strings.HasPrefix(strings.TrimSpace(prevLine), "//")

			// Get the full line for signature
			signature := line

			methods = append(methods, MethodInfo{
				filename:   filepath,
				lineNum:    lineNum,
				receiver:   receiver,
				name:       methodName,
				signature:  signature,
				hasComment: hasComment,
			})
		}

		prevLine = line
	}

	return methods, scanner.Err()
}

func main() {
	flag.Parse()

	packages := []string{
		"internal/render/backend",
		"internal/ui/widgets",
		"internal/render/display",
	}

	allMethods := []MethodInfo{}

	for _, pkg := range packages {
		err := filepath.Walk(pkg, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			methods, err := analyzeFile(path)
			if err == nil {
				allMethods = append(allMethods, methods...)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}

	// Print undocumented methods
	for _, m := range allMethods {
		if !m.hasComment {
			fmt.Printf("%s:%d\n", m.filename, m.lineNum)
			fmt.Printf("  Method: (%s) %s\n", m.receiver, m.name)
			fmt.Printf("  Signature: %s\n", m.signature)
			fmt.Println()
		}
	}
}
