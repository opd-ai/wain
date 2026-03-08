// Package backend provides a unified renderer interface with automatic GPU detection
// and fallback to software rasterizer. It abstracts the underlying GPU or software
// rendering backend, implementing an automatic fallback chain (Intel → AMD → Software)
// for maximum compatibility.
package backend
