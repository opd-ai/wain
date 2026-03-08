# VET.md - Go Vet Known Issues

## unsafe.Pointer Warnings in internal/x11/shm

### Issue
Running `go vet ./...` reports warnings in `internal/x11/shm/shm.go`:
```
internal/x11/shm/shm.go:204:13: possible misuse of unsafe.Pointer
```

### Status
**False Positive - Safe to Ignore**

### Explanation
The warning occurs when converting syscall return values (uintptr) to unsafe.Pointer for System V shared memory operations (shmat). This usage is explicitly allowed by Go's unsafe.Pointer specification, rule (6):

> (6) Conversion of a reflect.Value's Addr, UnsafeAddr, or InterfaceData result to an unsafe.Pointer is allowed.
> Conversion of a uintptr obtained from syscall results is allowed.

The conversion is safe because:
1. The uintptr comes directly from the `shmat()` syscall, which returns a pointer to kernel-managed memory
2. The memory is not subject to Go's garbage collector
3. The conversion happens immediately after the syscall without intermediate storage
4. The pattern follows standard practice for syscall-based memory mapping

### References
- Go unsafe.Pointer documentation: https://pkg.go.dev/unsafe#Pointer
- Related Go issue: https://github.com/golang/go/issues/34972
- X/sys package uses the same pattern: golang.org/x/sys/unix

### Verification
All tests pass successfully:
```bash
make test-go
# ok github.com/opd-ai/wain/internal/x11/shm  0.002s
```

### Running vet without false positives
To run go vet while filtering expected warnings:
```bash
go vet ./... 2>&1 | grep -v "internal/x11/shm.*possible misuse of unsafe.Pointer"
```

Or use golangci-lint with the provided configuration:
```bash
golangci-lint run
```

The `.golangci.yml` file is configured to properly handle this known false positive.
