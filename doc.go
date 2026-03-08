// Package wain is a statically-compiled Go UI toolkit with GPU rendering via Rust.
//
// # Getting Started
//
// To use wain in your Go projects:
//
//	go get github.com/opd-ai/wain
//
// # Building Applications
//
// wain uses CGO to link a static Rust rendering library. For tagged releases,
// pre-built static libraries are provided for common platforms (x86_64, aarch64 Linux).
//
// Standard build with pre-built libraries:
//
//	go build .
//
// # Rebuilding from Source
//
// To rebuild the Rust backend from source, use the wain-build helper tool:
//
//	go install github.com/opd-ai/wain/cmd/wain-build@latest
//	wain-build
//	go build .
//
// Prerequisites for rebuilding:
//   - cargo (Rust build tool)
//   - musl-gcc (musl C compiler)
//   - musl target: rustup target add x86_64-unknown-linux-musl
//
// # Hello World
//
// A minimal wain application:
//
//	package main
//
//	import "github.com/opd-ai/wain"
//
//	func main() {
//		app := wain.NewApp()
//		defer app.Close()
//
//		window, _ := app.NewWindow("Hello", 800, 600)
//		window.Show()
//
//		app.Run()
//	}
//
// See the examples in cmd/ for more demonstrations.
package wain
