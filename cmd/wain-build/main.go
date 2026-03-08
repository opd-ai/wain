package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const usage = `wain-build - Build wain Rust backend from source

Usage:
  wain-build [options]

Description:
  Builds the Rust rendering library (librender_sys.a) from source and places
  it in the current working directory. This tool is for contributors or
  advanced users who want to rebuild the Rust backend instead of using the
  pre-built static libraries bundled with wain releases.

Options:
  -h, --help     Show this help message
  -v, --verbose  Enable verbose build output
  -o <dir>       Output directory (default: current working directory)

Prerequisites:
  - cargo        Rust build tool (install from https://rustup.rs)
  - musl-gcc     musl C compiler (install via package manager)
  - musl target  Run: rustup target add <arch>-unknown-linux-musl

Examples:
  # Build in current directory
  wain-build

  # Build with verbose output
  wain-build -v

  # Build to a specific directory
  wain-build -o /path/to/output

After building, run 'go build' to link the rebuilt Rust library.
`

var (
	verbose   bool
	outputDir string
)

func main() {
	flag.BoolVar(&verbose, "v", false, "enable verbose output")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose output")
	flag.StringVar(&outputDir, "o", ".", "output directory")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Find the wain module root
	moduleRoot, err := findModuleRoot()
	if err != nil {
		return fmt.Errorf("finding wain module: %w", err)
	}

	if verbose {
		fmt.Printf("Found wain module at: %s\n", moduleRoot)
	}

	// Detect architecture and musl target
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	muslTarget := arch + "-unknown-linux-musl"

	if verbose {
		fmt.Printf("Detected architecture: %s\n", runtime.GOARCH)
		fmt.Printf("Target triple: %s\n", muslTarget)
	}

	// Check prerequisites
	if err := checkPrerequisites(muslTarget); err != nil {
		return err
	}

	// Build the Rust library
	rustDir := filepath.Join(moduleRoot, "render-sys")
	if err := buildRust(rustDir, muslTarget); err != nil {
		return fmt.Errorf("building Rust library: %w", err)
	}

	// Build the dl_find_object stub
	stubSrc := filepath.Join(moduleRoot, "internal", "render", "dl_find_object_stub.c")
	stubObj := "dl_find_object_stub.o"
	if err := buildStub(stubSrc, stubObj); err != nil {
		return fmt.Errorf("building dl_find_object stub: %w", err)
	}

	// Copy outputs to destination
	rustLib := filepath.Join(rustDir, "target", muslTarget, "release", "librender_sys.a")
	if err := copyOutputs(rustLib, stubObj, outputDir); err != nil {
		return fmt.Errorf("copying outputs: %w", err)
	}

	fmt.Println("✓ Build successful")
	fmt.Printf("  librender_sys.a → %s\n", filepath.Join(outputDir, "librender_sys.a"))
	fmt.Printf("  dl_find_object_stub.o → %s\n", filepath.Join(outputDir, "dl_find_object_stub.o"))
	fmt.Println("\nYou can now run 'go build' to link the rebuilt Rust library.")
	return nil
}

func findModuleRoot() (string, error) {
	// Try to find go.mod walking up from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// First check if we're in the wain module
	for {
		gomod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gomod); err == nil {
			if strings.Contains(string(data), "module github.com/opd-ai/wain") {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If not found, try to find it via go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/opd-ai/wain")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}

	return "", fmt.Errorf("could not find wain module (not in a wain project or module not downloaded)")
}

func checkPrerequisites(muslTarget string) error {
	// Check for cargo
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found. Install Rust from https://rustup.rs")
	}

	// Check for musl-gcc
	if _, err := exec.LookPath("musl-gcc"); err != nil {
		return fmt.Errorf("musl-gcc not found. Install musl-tools via your package manager")
	}

	// Check for musl target
	cmd := exec.Command("rustup", "target", "list", "--installed")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("checking installed targets: %w", err)
	}

	if !strings.Contains(string(out), muslTarget) {
		return fmt.Errorf("musl target not installed. Run: rustup target add %s", muslTarget)
	}

	if verbose {
		fmt.Println("✓ All prerequisites found")
	}

	return nil
}

func buildRust(rustDir, muslTarget string) error {
	fmt.Println("Building Rust library...")

	args := []string{"build", "--release", "--target", muslTarget}
	if !verbose {
		args = append(args, "--quiet")
	}

	cmd := exec.Command("cargo", args...)
	cmd.Dir = rustDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if verbose {
		fmt.Printf("  Running: cargo %s\n", strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func buildStub(stubSrc, stubObj string) error {
	fmt.Println("Building dl_find_object stub...")

	cmd := exec.Command("musl-gcc", "-c", stubSrc, "-o", stubObj)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if verbose {
		fmt.Printf("  Running: musl-gcc -c %s -o %s\n", stubSrc, stubObj)
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func copyOutputs(rustLib, stubObj, destDir string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Copy Rust library
	destRustLib := filepath.Join(destDir, "librender_sys.a")
	if err := copyFile(rustLib, destRustLib); err != nil {
		return fmt.Errorf("copying Rust library: %w", err)
	}

	// Copy stub object
	destStubObj := filepath.Join(destDir, "dl_find_object_stub.o")
	if err := copyFile(stubObj, destStubObj); err != nil {
		return fmt.Errorf("copying stub object: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
