// Package shm implements Wayland shared memory support.
package shm

import (
	"fmt"
	"syscall"
	"unsafe"
)

// CreateMemfd creates an anonymous shared memory file descriptor using memfd_create.
// The name is used for debugging purposes only (visible in /proc/pid/fd/).
func CreateMemfd(name string) (int, error) {
	const sysMemfdCreate = 319 // SYS_memfd_create on x86_64 Linux
	const mfdCloexec = 0x0001  // MFD_CLOEXEC flag
	nameBytes := append([]byte(name), 0)

	fd, _, errno := syscall.Syscall(sysMemfdCreate, uintptr(unsafe.Pointer(&nameBytes[0])), uintptr(mfdCloexec), 0)
	if errno != 0 {
		return -1, fmt.Errorf("memfd_create failed: %w", errno)
	}

	return int(fd), nil
}

// MmapFile memory-maps a file descriptor for read/write access.
func MmapFile(fd, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	data, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	return data, nil
}

// MunmapFile unmaps memory previously mapped with MmapFile.
func MunmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if err := syscall.Munmap(data); err != nil {
		return fmt.Errorf("munmap failed: %w", err)
	}

	return nil
}
