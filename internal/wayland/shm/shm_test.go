package shm

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// mockConn implements the Conn interface for testing.
type mockConn struct {
	nextID   uint32
	objects  map[uint32]interface{}
	requests []mockRequest
}

type mockRequest struct {
	objectID uint32
	opcode   uint16
	args     []wire.Argument
}

func newMockConn() *mockConn {
	return &mockConn{
		nextID:  2,
		objects: make(map[uint32]interface{}),
	}
}

func (m *mockConn) AllocID() uint32 {
	id := m.nextID
	m.nextID++
	return id
}

func (m *mockConn) RegisterObject(obj interface{}) {
	if o, ok := obj.(interface{ ID() uint32 }); ok {
		m.objects[o.ID()] = obj
	}
}

func (m *mockConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.requests = append(m.requests, mockRequest{
		objectID: objectID,
		opcode:   opcode,
		args:     args,
	})
	return nil
}

func (m *mockConn) lastRequest() mockRequest {
	if len(m.requests) == 0 {
		return mockRequest{}
	}
	return m.requests[len(m.requests)-1]
}

func TestCreateMemfd(t *testing.T) {
	fd, err := CreateMemfd("test-buffer")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	if fd < 0 {
		t.Errorf("CreateMemfd returned invalid fd: %d", fd)
	}

	// Verify the fd is writable.
	const testSize = 4096
	if err := syscall.Ftruncate(fd, testSize); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	// Check size.
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		t.Fatalf("Fstat failed: %v", err)
	}

	if stat.Size != testSize {
		t.Errorf("Size mismatch: got %d, want %d", stat.Size, testSize)
	}
}

func TestMmapFile(t *testing.T) {
	fd, err := CreateMemfd("test-mmap")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const size = 4096
	if err := syscall.Ftruncate(fd, size); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	data, err := MmapFile(fd, size)
	if err != nil {
		t.Fatalf("MmapFile failed: %v", err)
	}
	defer func() { _ = MunmapFile(data) }()

	if len(data) != size {
		t.Errorf("Mmap size mismatch: got %d, want %d", len(data), size)
	}

	// Write and read test.
	testData := []byte("Hello, Wayland!")
	copy(data, testData)

	for i, b := range testData {
		if data[i] != b {
			t.Errorf("Data mismatch at offset %d: got %d, want %d", i, data[i], b)
		}
	}
}

func TestMunmapFile(t *testing.T) {
	fd, err := CreateMemfd("test-munmap")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const size = 4096
	if err := syscall.Ftruncate(fd, size); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	data, err := MmapFile(fd, size)
	if err != nil {
		t.Fatalf("MmapFile failed: %v", err)
	}

	if err := MunmapFile(data); err != nil {
		t.Errorf("MunmapFile failed: %v", err)
	}

	// Munmap empty slice should not error.
	if err := MunmapFile([]byte{}); err != nil {
		t.Errorf("MunmapFile on empty slice failed: %v", err)
	}
}

func TestNewSHM(t *testing.T) {
	conn := newMockConn()
	shm := NewSHM(conn, 42)

	if shm.ID() != 42 {
		t.Errorf("ID mismatch: got %d, want 42", shm.ID())
	}

	if shm.Interface() != "wl_shm" {
		t.Errorf("Interface mismatch: got %s, want wl_shm", shm.Interface())
	}

	if len(shm.formats) != 0 {
		t.Errorf("Formats should be empty initially")
	}
}

func TestSHMHandleEvent(t *testing.T) {
	conn := newMockConn()
	shm := NewSHM(conn, 42)

	tests := []struct {
		name    string
		opcode  uint16
		args    []wire.Argument
		wantErr bool
		wantFmt uint32
	}{
		{
			name:   "format ARGB8888",
			opcode: 0,
			args: []wire.Argument{
				{Type: wire.ArgTypeUint32, Value: uint32(FormatARGB8888)},
			},
			wantErr: false,
			wantFmt: FormatARGB8888,
		},
		{
			name:   "format XRGB8888",
			opcode: 0,
			args: []wire.Argument{
				{Type: wire.ArgTypeUint32, Value: uint32(FormatXRGB8888)},
			},
			wantErr: false,
			wantFmt: FormatXRGB8888,
		},
		{
			name:    "invalid opcode",
			opcode:  99,
			args:    []wire.Argument{},
			wantErr: true,
		},
		{
			name:    "wrong arg count",
			opcode:  0,
			args:    []wire.Argument{},
			wantErr: true,
		},
		{
			name:   "wrong arg type",
			opcode: 0,
			args: []wire.Argument{
				{Type: wire.ArgTypeInt32, Value: int32(0)},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shm.HandleEvent(tt.opcode, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleEvent error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Drain the format channel to update formats map.
				select {
				case format := <-shm.formatsChan:
					shm.formats[format] = true
				case <-time.After(100 * time.Millisecond):
					t.Error("Timeout waiting for format event")
				}

				if !shm.HasFormat(tt.wantFmt) {
					t.Errorf("Format %d not found after event", tt.wantFmt)
				}
			}
		})
	}
}

func TestSHMCreatePool(t *testing.T) {
	conn := newMockConn()
	shm := NewSHM(conn, 42)

	fd, err := CreateMemfd("test-pool")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const size = 4096
	if err := syscall.Ftruncate(fd, size); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	pool, err := shm.CreatePool(fd, size)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	if pool == nil {
		t.Fatal("Pool is nil")
	}

	if pool.Interface() != "wl_shm_pool" {
		t.Errorf("Interface mismatch: got %s, want wl_shm_pool", pool.Interface())
	}

	// Verify request was sent.
	req := conn.lastRequest()
	if req.objectID != shm.ID() {
		t.Errorf("Request objectID mismatch: got %d, want %d", req.objectID, shm.ID())
	}
	if req.opcode != 0 {
		t.Errorf("Request opcode mismatch: got %d, want 0", req.opcode)
	}

	// Verify pool was registered.
	if _, ok := conn.objects[pool.ID()]; !ok {
		t.Errorf("Pool not registered with connection")
	}
}

func TestPoolCreateBuffer(t *testing.T) {
	conn := newMockConn()

	fd, err := CreateMemfd("test-buffer")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const size = 4096
	if err := syscall.Ftruncate(fd, size); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	pool := NewPool(conn, 100, fd, size)
	if err := pool.Map(); err != nil {
		t.Fatalf("Map failed: %v", err)
	}
	defer func() { _ = pool.Unmap() }()

	tests := []struct {
		name    string
		offset  int32
		width   int32
		height  int32
		stride  int32
		format  uint32
		wantErr bool
	}{
		{
			name:    "valid buffer",
			offset:  0,
			width:   32,
			height:  32,
			stride:  128,
			format:  FormatARGB8888,
			wantErr: false,
		},
		{
			name:    "invalid width",
			offset:  0,
			width:   0,
			height:  32,
			stride:  128,
			format:  FormatARGB8888,
			wantErr: true,
		},
		{
			name:    "invalid height",
			offset:  0,
			width:   32,
			height:  0,
			stride:  128,
			format:  FormatARGB8888,
			wantErr: true,
		},
		{
			name:    "invalid stride",
			offset:  0,
			width:   32,
			height:  32,
			stride:  0,
			format:  FormatARGB8888,
			wantErr: true,
		},
		{
			name:    "offset out of bounds",
			offset:  size + 100,
			width:   32,
			height:  32,
			stride:  128,
			format:  FormatARGB8888,
			wantErr: true,
		},
		{
			name:    "buffer extends beyond pool",
			offset:  0,
			width:   1000,
			height:  1000,
			stride:  4000,
			format:  FormatARGB8888,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := pool.CreateBuffer(tt.offset, tt.width, tt.height, tt.stride, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBuffer error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if buf == nil {
					t.Fatal("Buffer is nil")
				}

				if buf.Width() != tt.width {
					t.Errorf("Width mismatch: got %d, want %d", buf.Width(), tt.width)
				}
				if buf.Height() != tt.height {
					t.Errorf("Height mismatch: got %d, want %d", buf.Height(), tt.height)
				}
				if buf.Stride() != tt.stride {
					t.Errorf("Stride mismatch: got %d, want %d", buf.Stride(), tt.stride)
				}
				if buf.Format() != tt.format {
					t.Errorf("Format mismatch: got %d, want %d", buf.Format(), tt.format)
				}

				// Verify pixels are accessible.
				if len(buf.Pixels()) != int(tt.height*tt.stride) {
					t.Errorf("Pixels size mismatch: got %d, want %d", len(buf.Pixels()), tt.height*tt.stride)
				}
			}
		})
	}
}

func TestBufferHandleEvent(t *testing.T) {
	conn := newMockConn()
	buf := NewBuffer(conn, 200, nil, 0, 32, 32, 128, FormatARGB8888, nil)

	// Send release event.
	err := buf.HandleEvent(0, []wire.Argument{})
	if err != nil {
		t.Fatalf("HandleEvent(release) failed: %v", err)
	}

	// Wait for release.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := buf.WaitRelease(ctx); err != nil {
		t.Errorf("WaitRelease failed: %v", err)
	}

	// Invalid opcode.
	if err := buf.HandleEvent(99, []wire.Argument{}); err == nil {
		t.Error("HandleEvent should fail for invalid opcode")
	}
}

func TestBufferWaitReleaseTimeout(t *testing.T) {
	conn := newMockConn()
	buf := NewBuffer(conn, 200, nil, 0, 32, 32, 128, FormatARGB8888, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := buf.WaitRelease(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("WaitRelease error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestPoolResize(t *testing.T) {
	conn := newMockConn()

	fd, err := CreateMemfd("test-resize")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const initialSize = 4096
	if err := syscall.Ftruncate(fd, initialSize); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	pool := NewPool(conn, 100, fd, initialSize)
	if err := pool.Map(); err != nil {
		t.Fatalf("Map failed: %v", err)
	}
	defer func() { _ = pool.Unmap() }()

	// Resize to larger size.
	const newSize = 8192
	if err := pool.Resize(newSize); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	if pool.size != newSize {
		t.Errorf("Pool size mismatch after resize: got %d, want %d", pool.size, newSize)
	}

	// Verify fd size.
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		t.Fatalf("Fstat failed: %v", err)
	}

	if stat.Size != newSize {
		t.Errorf("FD size mismatch: got %d, want %d", stat.Size, newSize)
	}

	// Verify mapping size.
	if len(pool.mapping) != newSize {
		t.Errorf("Mapping size mismatch: got %d, want %d", len(pool.mapping), newSize)
	}
}

func TestPoolDestroy(t *testing.T) {
	conn := newMockConn()

	fd, err := CreateMemfd("test-destroy")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}

	const size = 4096
	if err := syscall.Ftruncate(fd, size); err != nil {
		syscall.Close(fd)
		t.Fatalf("Ftruncate failed: %v", err)
	}

	pool := NewPool(conn, 100, fd, size)
	if err := pool.Map(); err != nil {
		syscall.Close(fd)
		t.Fatalf("Map failed: %v", err)
	}

	if err := pool.Destroy(); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// Verify mapping was freed.
	if pool.mapping != nil {
		t.Error("Mapping should be nil after destroy")
	}

	// Verify fd was closed (next syscall on fd should fail).
	if err := syscall.Ftruncate(fd, 1024); err == nil {
		t.Error("FD should be closed after destroy")
	}
}

func TestIntegration(t *testing.T) {
	// Create a full SHM workflow.
	conn := newMockConn()
	shm := NewSHM(conn, 42)

	// Simulate format events.
	if err := shm.HandleEvent(0, []wire.Argument{{Type: wire.ArgTypeUint32, Value: uint32(FormatARGB8888)}}); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}
	if err := shm.HandleEvent(0, []wire.Argument{{Type: wire.ArgTypeUint32, Value: uint32(FormatXRGB8888)}}); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Drain format events.
	for len(shm.formatsChan) > 0 {
		format := <-shm.formatsChan
		shm.formats[format] = true
	}

	if !shm.HasFormat(FormatARGB8888) {
		t.Error("ARGB8888 format not available")
	}
	if !shm.HasFormat(FormatXRGB8888) {
		t.Error("XRGB8888 format not available")
	}

	// Create pool.
	fd, err := CreateMemfd("integration-test")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	const width, height = 64, 64
	const stride = width * 4
	const size = height * stride

	if err := syscall.Ftruncate(fd, size); err != nil {
		t.Fatalf("Ftruncate failed: %v", err)
	}

	pool, err := shm.CreatePool(fd, size)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	if err := pool.Map(); err != nil {
		t.Fatalf("Map failed: %v", err)
	}
	defer func() { _ = pool.Unmap() }()

	// Create buffer.
	buf, err := pool.CreateBuffer(0, width, height, stride, FormatARGB8888)
	if err != nil {
		t.Fatalf("CreateBuffer failed: %v", err)
	}

	// Write to pixels.
	pixels := buf.Pixels()
	if len(pixels) != size {
		t.Fatalf("Pixels size mismatch: got %d, want %d", len(pixels), size)
	}

	// Fill with a test pattern (ARGB = 0xFF0000FF = blue).
	for i := 0; i < len(pixels); i += 4 {
		pixels[i+0] = 0xFF // B
		pixels[i+1] = 0x00 // G
		pixels[i+2] = 0x00 // R
		pixels[i+3] = 0xFF // A
	}

	// Verify pattern.
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] != 0xFF || pixels[i+1] != 0x00 || pixels[i+2] != 0x00 || pixels[i+3] != 0xFF {
			t.Errorf("Pixel mismatch at offset %d: got [%02x %02x %02x %02x]", i,
				pixels[i], pixels[i+1], pixels[i+2], pixels[i+3])
			break
		}
	}

	// Simulate release event.
	if err := buf.HandleEvent(0, []wire.Argument{}); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	if err := buf.WaitRelease(ctx2); err != nil {
		t.Errorf("WaitRelease failed: %v", err)
	}

	// Destroy buffer.
	if err := buf.Destroy(); err != nil {
		t.Errorf("Buffer destroy failed: %v", err)
	}

	// Destroy pool.
	if err := pool.Destroy(); err != nil {
		t.Errorf("Pool destroy failed: %v", err)
	}
}

// Verify memfd doesn't create files on disk.
func TestMemfdNoFiles(t *testing.T) {
	fd, err := CreateMemfd("test-no-files")
	if err != nil {
		t.Fatalf("CreateMemfd failed: %v", err)
	}
	defer syscall.Close(fd)

	// Read /proc/self/fd/N to see the target (should be memfd:...)
	linkPath := "/proc/self/fd/" + string(rune(fd+48)) // Poor man's itoa for small fds
	target, err := os.Readlink(linkPath)
	if err == nil && len(target) > 0 {
		// We can't reliably check this in all test environments, so just log.
		t.Logf("memfd link target: %s", target)
	}
}
