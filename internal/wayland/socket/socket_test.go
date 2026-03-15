package socket

import (
	"bytes"
	"io"
	"os"
	"syscall"
	"testing"
)

// TestMakePair verifies that MakePair creates two connected sockets.
func TestMakePair(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	if c1.Fd() < 0 {
		t.Errorf("c1 fd is invalid: %d", c1.Fd())
	}

	if c2.Fd() < 0 {
		t.Errorf("c2 fd is invalid: %d", c2.Fd())
	}

	// Verify they are connected by sending data
	msg := []byte("test")
	if _, err := c1.Write(msg); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	buf := make([]byte, 10)
	if n, err := c2.Read(buf); err != nil {
		t.Fatalf("Read failed: %v", err)
	} else if !bytes.Equal(buf[:n], msg) {
		t.Errorf("Read wrong data: got %q, want %q", buf[:n], msg)
	}
}

// TestBasicReadWrite verifies basic socket I/O without FD passing.
func TestBasicReadWrite(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	tests := []struct {
		name string
		data []byte
	}{
		{"small", []byte("hello")},
		{"medium", bytes.Repeat([]byte("x"), 256)},
		{"large", bytes.Repeat([]byte("test"), 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := c1.Write(tt.data)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Write returned %d, want %d", n, len(tt.data))
			}

			buf := make([]byte, len(tt.data)+10)
			n, err = c2.Read(buf)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Read returned %d, want %d", n, len(tt.data))
			}

			if !bytes.Equal(buf[:n], tt.data) {
				t.Errorf("Read data mismatch:\ngot:  %q\nwant: %q", buf[:n], tt.data)
			}
		})
	}
}

// TestSendRecvFD verifies single file descriptor passing.
func TestSendRecvFD(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	tmpfile, err := os.CreateTemp("", "socket-test-")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	testData := []byte("test content for fd passing")
	if _, err := tmpfile.Write(testData); err != nil {
		tmpfile.Close()
		t.Fatalf("Write to tmpfile failed: %v", err)
	}
	if _, err := tmpfile.Seek(0, io.SeekStart); err != nil {
		tmpfile.Close()
		t.Fatalf("Seek failed: %v", err)
	}

	sendFD := int(tmpfile.Fd())
	msg := []byte("message with fd")

	n, err := c1.SendFD(msg, sendFD)
	if err != nil {
		tmpfile.Close()
		t.Fatalf("SendFD failed: %v", err)
	}
	if n != len(msg) {
		tmpfile.Close()
		t.Errorf("SendFD wrote %d bytes, want %d", n, len(msg))
	}

	buf := make([]byte, 100)
	n, recvFD, err := c2.RecvFD(buf)
	if err != nil {
		tmpfile.Close()
		t.Fatalf("RecvFD failed: %v", err)
	}
	defer syscall.Close(recvFD)

	if n != len(msg) {
		tmpfile.Close()
		t.Errorf("RecvFD read %d bytes, want %d", n, len(msg))
	}
	if !bytes.Equal(buf[:n], msg) {
		tmpfile.Close()
		t.Errorf("RecvFD data mismatch:\ngot:  %q\nwant: %q", buf[:n], msg)
	}

	if recvFD < 0 {
		tmpfile.Close()
		t.Fatalf("RecvFD returned invalid fd: %d", recvFD)
	}

	recvFile := os.NewFile(uintptr(recvFD), "received")
	content, err := io.ReadAll(recvFile)
	if err != nil {
		tmpfile.Close()
		t.Fatalf("ReadAll from received fd failed: %v", err)
	}

	if !bytes.Equal(content, testData) {
		tmpfile.Close()
		t.Errorf("File content mismatch:\ngot:  %q\nwant: %q", content, testData)
	}

	tmpfile.Close()
}

// TestSendRecvMultipleFDs verifies sending multiple file descriptors.
func TestSendRecvMultipleFDs(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	testCases := []struct {
		name  string
		count int
	}{
		{"one", 1},
		{"two", 2},
		{"five", 5},
		{"ten", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var fds []int

			for i := 0; i < tc.count; i++ {
				f, err := os.CreateTemp("", "socket-test-multi-")
				if err != nil {
					t.Fatalf("CreateTemp failed: %v", err)
				}
				defer os.Remove(f.Name())
				defer f.Close()

				content := []byte{byte(i)}
				if _, err := f.Write(content); err != nil {
					t.Fatalf("Write failed: %v", err)
				}
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					t.Fatalf("Seek failed: %v", err)
				}

				fds = append(fds, int(f.Fd()))
			}

			msg := []byte("multi-fd message")
			n, err := c1.SendMsg(msg, fds)
			if err != nil {
				t.Fatalf("SendMsg failed: %v", err)
			}
			if n != len(msg) {
				t.Errorf("SendMsg wrote %d bytes, want %d", n, len(msg))
			}

			buf := make([]byte, 100)
			n, recvFDs, err := c2.RecvMsg(buf, tc.count)
			if err != nil {
				t.Fatalf("RecvMsg failed: %v", err)
			}
			defer func() {
				for _, fd := range recvFDs {
					syscall.Close(fd)
				}
			}()

			if n != len(msg) {
				t.Errorf("RecvMsg read %d bytes, want %d", n, len(msg))
			}
			if !bytes.Equal(buf[:n], msg) {
				t.Errorf("RecvMsg data mismatch:\ngot:  %q\nwant: %q", buf[:n], msg)
			}

			if len(recvFDs) != tc.count {
				t.Fatalf("RecvMsg received %d fds, want %d", len(recvFDs), tc.count)
			}

			for i, fd := range recvFDs {
				if fd < 0 {
					t.Errorf("RecvMsg fd[%d] is invalid: %d", i, fd)
					continue
				}

				f := os.NewFile(uintptr(fd), "received")
				content, err := io.ReadAll(f)
				if err != nil {
					t.Errorf("ReadAll from fd[%d] failed: %v", i, err)
					continue
				}

				expected := []byte{byte(i)}
				if !bytes.Equal(content, expected) {
					t.Errorf("fd[%d] content mismatch:\ngot:  %v\nwant: %v", i, content, expected)
				}
			}
		})
	}
}

// TestSendMsgNoFDs verifies that SendMsg works without file descriptors.
func TestSendMsgNoFDs(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	msg := []byte("message without fds")
	n, err := c1.SendMsg(msg, nil)
	if err != nil {
		t.Fatalf("SendMsg failed: %v", err)
	}
	if n != len(msg) {
		t.Errorf("SendMsg wrote %d bytes, want %d", n, len(msg))
	}

	buf := make([]byte, 100)
	n, fds, err := c2.RecvMsg(buf, 10)
	if err != nil {
		t.Fatalf("RecvMsg failed: %v", err)
	}

	if n != len(msg) {
		t.Errorf("RecvMsg read %d bytes, want %d", n, len(msg))
	}
	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("RecvMsg data mismatch:\ngot:  %q\nwant: %q", buf[:n], msg)
	}
	if len(fds) != 0 {
		t.Errorf("RecvMsg received %d fds, want 0", len(fds))
		for _, fd := range fds {
			syscall.Close(fd)
		}
	}
}

// TestRecvFDErrors verifies error handling for RecvFD.
func TestRecvFDErrors(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	t.Run("no_fd_sent", func(t *testing.T) {
		msg := []byte("message without fd")
		if _, err := c1.Write(msg); err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		buf := make([]byte, 100)
		_, _, err := c2.RecvFD(buf)
		if err != ErrNoFileDescriptors {
			t.Errorf("RecvFD error = %v, want %v", err, ErrNoFileDescriptors)
		}
	})

	t.Run("multiple_fds_sent", func(t *testing.T) {
		f1, err := os.CreateTemp("", "socket-test-multi1-")
		if err != nil {
			t.Fatalf("CreateTemp failed: %v", err)
		}
		defer os.Remove(f1.Name())
		defer f1.Close()

		f2, err := os.CreateTemp("", "socket-test-multi2-")
		if err != nil {
			t.Fatalf("CreateTemp failed: %v", err)
		}
		defer os.Remove(f2.Name())
		defer f2.Close()

		msg := []byte("message with two fds")
		fds := []int{int(f1.Fd()), int(f2.Fd())}

		if _, err := c1.SendMsg(msg, fds); err != nil {
			t.Fatalf("SendMsg failed: %v", err)
		}

		buf := make([]byte, 100)
		_, fd, err := c2.RecvFD(buf)
		// RecvMsg(data, 1) enforces the maxFDs limit by closing extras,
		// so RecvFD sees exactly 1 FD and succeeds.
		if err != nil {
			t.Errorf("RecvFD error = %v, want nil (extras closed by RecvMsg)", err)
		}
		if fd >= 0 {
			syscall.Close(fd)
		}
	})
}

// TestSendTooManyFDs verifies the MaxFDsPerMessage limit.
func TestSendTooManyFDs(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	fds := make([]int, MaxFDsPerMessage+1)
	for i := range fds {
		fds[i] = 0
	}

	msg := []byte("too many fds")
	_, err = c1.SendMsg(msg, fds)
	if err != ErrTooManyFileDescriptors {
		t.Errorf("SendMsg error = %v, want %v", err, ErrTooManyFileDescriptors)
	}
}

// TestBidirectionalFDPassing verifies FD passing in both directions.
func TestBidirectionalFDPassing(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	f1, err := os.CreateTemp("", "socket-test-bidir1-")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := os.CreateTemp("", "socket-test-bidir2-")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	data1 := []byte("from c1 to c2")
	if _, err := f1.Write(data1); err != nil {
		t.Fatalf("Write to f1 failed: %v", err)
	}
	if _, err := f1.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek f1 failed: %v", err)
	}

	data2 := []byte("from c2 to c1")
	if _, err := f2.Write(data2); err != nil {
		t.Fatalf("Write to f2 failed: %v", err)
	}
	if _, err := f2.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek f2 failed: %v", err)
	}

	msg1 := []byte("c1 -> c2")
	if _, err := c1.SendFD(msg1, int(f1.Fd())); err != nil {
		t.Fatalf("c1.SendFD failed: %v", err)
	}

	msg2 := []byte("c2 -> c1")
	if _, err := c2.SendFD(msg2, int(f2.Fd())); err != nil {
		t.Fatalf("c2.SendFD failed: %v", err)
	}

	buf := make([]byte, 100)
	n, fd, err := c2.RecvFD(buf)
	if err != nil {
		t.Fatalf("c2.RecvFD failed: %v", err)
	}
	defer syscall.Close(fd)

	if !bytes.Equal(buf[:n], msg1) {
		t.Errorf("c2 received wrong message:\ngot:  %q\nwant: %q", buf[:n], msg1)
	}

	recvFile1 := os.NewFile(uintptr(fd), "recv1")
	content1, err := io.ReadAll(recvFile1)
	if err != nil {
		t.Fatalf("ReadAll from recv1 failed: %v", err)
	}
	if !bytes.Equal(content1, data1) {
		t.Errorf("c2 received wrong file content:\ngot:  %q\nwant: %q", content1, data1)
	}

	n, fd, err = c1.RecvFD(buf)
	if err != nil {
		t.Fatalf("c1.RecvFD failed: %v", err)
	}
	defer syscall.Close(fd)

	if !bytes.Equal(buf[:n], msg2) {
		t.Errorf("c1 received wrong message:\ngot:  %q\nwant: %q", buf[:n], msg2)
	}

	recvFile2 := os.NewFile(uintptr(fd), "recv2")
	content2, err := io.ReadAll(recvFile2)
	if err != nil {
		t.Fatalf("ReadAll from recv2 failed: %v", err)
	}
	if !bytes.Equal(content2, data2) {
		t.Errorf("c1 received wrong file content:\ngot:  %q\nwant: %q", content2, data2)
	}
}

// TestFdMethod verifies the Fd accessor method.
func TestFdMethod(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	fd1 := c1.Fd()
	fd2 := c2.Fd()

	if fd1 < 0 {
		t.Errorf("c1.Fd() returned invalid fd: %d", fd1)
	}
	if fd2 < 0 {
		t.Errorf("c2.Fd() returned invalid fd: %d", fd2)
	}
	if fd1 == fd2 {
		t.Errorf("c1.Fd() and c2.Fd() returned same fd: %d", fd1)
	}
}

// TestAddrMethods verifies LocalAddr and RemoteAddr methods.
func TestAddrMethods(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c1.Close()
	defer c2.Close()

	if c1.LocalAddr() == nil {
		t.Error("c1.LocalAddr() returned nil")
	}
	if c1.RemoteAddr() == nil {
		t.Error("c1.RemoteAddr() returned nil")
	}
	if c2.LocalAddr() == nil {
		t.Error("c2.LocalAddr() returned nil")
	}
	if c2.RemoteAddr() == nil {
		t.Error("c2.RemoteAddr() returned nil")
	}
}

// TestClose verifies that Close properly closes the socket.
func TestClose(t *testing.T) {
	c1, c2, err := MakePair()
	if err != nil {
		t.Fatalf("MakePair failed: %v", err)
	}
	defer c2.Close()

	if err := c1.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	buf := make([]byte, 10)
	_, err = c1.Read(buf)
	if err == nil {
		t.Error("Read after Close succeeded, should have failed")
	}

	_, err = c1.Write([]byte("test"))
	if err == nil {
		t.Error("Write after Close succeeded, should have failed")
	}
}
