package wire_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

func TestDecodeHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    wire.Header
		wantErr bool
	}{
		{
			name:  "valid header",
			input: []byte{0x01, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x10, 0x00},
			want: wire.Header{
				ObjectID: 1,
				Opcode:   10,
				Size:     16,
			},
			wantErr: false,
		},
		{
			name:  "minimum size",
			input: []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00},
			want: wire.Header{
				ObjectID: 2,
				Opcode:   0,
				Size:     8,
			},
			wantErr: false,
		},
		{
			name:    "too short",
			input:   []byte{0x01, 0x00, 0x00},
			want:    wire.Header{},
			wantErr: true,
		},
		{
			name:    "size too small",
			input:   []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00},
			want:    wire.Header{},
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    wire.Header{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeHeader(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEncodeHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		header  wire.Header
		want    []byte
		wantErr bool
	}{
		{
			name: "valid header",
			header: wire.Header{
				ObjectID: 1,
				Opcode:   10,
				Size:     16,
			},
			want:    []byte{0x01, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x10, 0x00},
			wantErr: false,
		},
		{
			name: "minimum size",
			header: wire.Header{
				ObjectID: 2,
				Opcode:   0,
				Size:     8,
			},
			want:    []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00},
			wantErr: false,
		},
		{
			name: "size too small",
			header: wire.Header{
				ObjectID: 1,
				Opcode:   0,
				Size:     4,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "size too large",
			header: wire.Header{
				ObjectID: 1,
				Opcode:   0,
				Size:     5000,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeHeader(&buf, tc.header)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHeaderRoundtrip(t *testing.T) {
	t.Parallel()
	headers := []wire.Header{
		{ObjectID: 1, Opcode: 0, Size: 8},
		{ObjectID: 42, Opcode: 15, Size: 100},
		{ObjectID: 0xFFFFFFFF, Opcode: 0xFFFF, Size: 4096},
	}

	for _, original := range headers {
		var buf bytes.Buffer
		if err := wire.EncodeHeader(&buf, original); err != nil {
			t.Fatalf("encode error: %v", err)
		}

		decoded, err := wire.DecodeHeader(&buf)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if decoded != original {
			t.Errorf("roundtrip failed: got %+v, want %+v", decoded, original)
		}
	}
}

func TestDecodeInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    int32
		wantErr bool
	}{
		{name: "zero", input: []byte{0x00, 0x00, 0x00, 0x00}, want: 0},
		{name: "positive", input: []byte{0x2A, 0x00, 0x00, 0x00}, want: 42},
		{name: "negative", input: []byte{0xFF, 0xFF, 0xFF, 0xFF}, want: -1},
		{name: "max int32", input: []byte{0xFF, 0xFF, 0xFF, 0x7F}, want: 0x7FFFFFFF},
		{name: "min int32", input: []byte{0x00, 0x00, 0x00, 0x80}, want: -0x80000000},
		{name: "too short", input: []byte{0x00, 0x00}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeInt32(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEncodeInt32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value int32
		want  []byte
	}{
		{name: "zero", value: 0, want: []byte{0x00, 0x00, 0x00, 0x00}},
		{name: "positive", value: 42, want: []byte{0x2A, 0x00, 0x00, 0x00}},
		{name: "negative", value: -1, want: []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{name: "max int32", value: 0x7FFFFFFF, want: []byte{0xFF, 0xFF, 0xFF, 0x7F}},
		{name: "min int32", value: -0x80000000, want: []byte{0x00, 0x00, 0x00, 0x80}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := wire.EncodeInt32(&buf, tc.value); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInt32Roundtrip(t *testing.T) {
	t.Parallel()
	values := []int32{0, 1, -1, 42, -42, 0x7FFFFFFF, -0x80000000}

	for _, v := range values {
		var buf bytes.Buffer
		if err := wire.EncodeInt32(&buf, v); err != nil {
			t.Fatalf("encode error: %v", err)
		}

		got, err := wire.DecodeInt32(&buf)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if got != v {
			t.Errorf("roundtrip failed: got %d, want %d", got, v)
		}
	}
}

func TestDecodeUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    uint32
		wantErr bool
	}{
		{name: "zero", input: []byte{0x00, 0x00, 0x00, 0x00}, want: 0},
		{name: "small", input: []byte{0x2A, 0x00, 0x00, 0x00}, want: 42},
		{name: "max uint32", input: []byte{0xFF, 0xFF, 0xFF, 0xFF}, want: 0xFFFFFFFF},
		{name: "too short", input: []byte{0x00, 0x00}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeUint32(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEncodeUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value uint32
		want  []byte
	}{
		{name: "zero", value: 0, want: []byte{0x00, 0x00, 0x00, 0x00}},
		{name: "small", value: 42, want: []byte{0x2A, 0x00, 0x00, 0x00}},
		{name: "max uint32", value: 0xFFFFFFFF, want: []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := wire.EncodeUint32(&buf, tc.value); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUint32Roundtrip(t *testing.T) {
	t.Parallel()
	values := []uint32{0, 1, 42, 0xFFFFFFFF}

	for _, v := range values {
		var buf bytes.Buffer
		if err := wire.EncodeUint32(&buf, v); err != nil {
			t.Fatalf("encode error: %v", err)
		}

		got, err := wire.DecodeUint32(&buf)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if got != v {
			t.Errorf("roundtrip failed: got %d, want %d", got, v)
		}
	}
}

func TestDecodeFixed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    float64
		wantErr bool
	}{
		{name: "zero", input: []byte{0x00, 0x00, 0x00, 0x00}, want: 0.0},
		{name: "one", input: []byte{0x00, 0x01, 0x00, 0x00}, want: 1.0},
		{name: "half", input: []byte{0x80, 0x00, 0x00, 0x00}, want: 0.5},
		{name: "negative", input: []byte{0x00, 0xFF, 0xFF, 0xFF}, want: -1.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeFixed(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			const epsilon = 0.001
			if got < tc.want-epsilon || got > tc.want+epsilon {
				t.Errorf("got %f, want %f", got, tc.want)
			}
		})
	}
}

func TestFixedRoundtrip(t *testing.T) {
	t.Parallel()
	values := []float64{0.0, 1.0, -1.0, 0.5, 3.14, -2.71}

	for _, v := range values {
		var buf bytes.Buffer
		if err := wire.EncodeFixed(&buf, v); err != nil {
			t.Fatalf("encode error: %v", err)
		}

		got, err := wire.DecodeFixed(&buf)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		const epsilon = 0.01
		if got < v-epsilon || got > v+epsilon {
			t.Errorf("roundtrip failed: got %f, want %f", got, v)
		}
	}
}

func TestDecodeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:  "empty string",
			input: []byte{0x00, 0x00, 0x00, 0x00},
			want:  "",
		},
		{
			name: "hello",
			input: []byte{
				0x06, 0x00, 0x00, 0x00,
				'h', 'e', 'l', 'l', 'o', 0x00,
				0x00, 0x00,
			},
			want: "hello",
		},
		{
			name: "test",
			input: []byte{
				0x05, 0x00, 0x00, 0x00,
				't', 'e', 's', 't', 0x00,
				0x00, 0x00, 0x00,
			},
			want: "test",
		},
		{
			name: "single char",
			input: []byte{
				0x02, 0x00, 0x00, 0x00,
				'a', 0x00, 0x00, 0x00,
			},
			want: "a",
		},
		{
			name: "missing null terminator",
			input: []byte{
				0x05, 0x00, 0x00, 0x00,
				't', 'e', 's', 't', 'x',
				0x00, 0x00, 0x00,
			},
			wantErr: true,
		},
		{
			name:    "truncated",
			input:   []byte{0x06, 0x00, 0x00, 0x00, 'h', 'e'},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeString(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEncodeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{
			name:  "empty",
			input: "",
			want:  []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "hello",
			input: "hello",
			want: []byte{
				0x06, 0x00, 0x00, 0x00,
				'h', 'e', 'l', 'l', 'o', 0x00,
				0x00, 0x00,
			},
		},
		{
			name:  "test",
			input: "test",
			want: []byte{
				0x05, 0x00, 0x00, 0x00,
				't', 'e', 's', 't', 0x00,
				0x00, 0x00, 0x00,
			},
		},
		{
			name:  "a",
			input: "a",
			want: []byte{
				0x02, 0x00, 0x00, 0x00,
				'a', 0x00, 0x00, 0x00,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := wire.EncodeString(&buf, tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStringRoundtrip(t *testing.T) {
	t.Parallel()
	strings := []string{"", "a", "hello", "test", "wayland"}

	for _, s := range strings {
		var buf bytes.Buffer
		if err := wire.EncodeString(&buf, s); err != nil {
			t.Fatalf("encode error for %q: %v", s, err)
		}

		got, err := wire.DecodeString(&buf)
		if err != nil {
			t.Fatalf("decode error for %q: %v", s, err)
		}

		if got != s {
			t.Errorf("roundtrip failed: got %q, want %q", got, s)
		}
	}
}

func TestDecodeArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr bool
	}{
		{
			name:  "empty array",
			input: []byte{0x00, 0x00, 0x00, 0x00},
			want:  nil,
		},
		{
			name: "single byte",
			input: []byte{
				0x01, 0x00, 0x00, 0x00,
				0x42, 0x00, 0x00, 0x00,
			},
			want: []byte{0x42},
		},
		{
			name: "four bytes",
			input: []byte{
				0x04, 0x00, 0x00, 0x00,
				0x01, 0x02, 0x03, 0x04,
			},
			want: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name: "five bytes with padding",
			input: []byte{
				0x05, 0x00, 0x00, 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05,
				0x00, 0x00, 0x00,
			},
			want: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
		{
			name:    "truncated",
			input:   []byte{0x04, 0x00, 0x00, 0x00, 0x01},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeArray(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEncodeArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "empty",
			input: nil,
			want:  []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:  "single byte",
			input: []byte{0x42},
			want: []byte{
				0x01, 0x00, 0x00, 0x00,
				0x42, 0x00, 0x00, 0x00,
			},
		},
		{
			name:  "four bytes",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			want: []byte{
				0x04, 0x00, 0x00, 0x00,
				0x01, 0x02, 0x03, 0x04,
			},
		},
		{
			name:  "five bytes",
			input: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			want: []byte{
				0x05, 0x00, 0x00, 0x00,
				0x01, 0x02, 0x03, 0x04, 0x05,
				0x00, 0x00, 0x00,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := wire.EncodeArray(&buf, tc.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestArrayRoundtrip(t *testing.T) {
	t.Parallel()
	arrays := [][]byte{
		nil,
		{},
		{0x42},
		{0x01, 0x02, 0x03, 0x04},
		{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	for i, arr := range arrays {
		var buf bytes.Buffer
		if err := wire.EncodeArray(&buf, arr); err != nil {
			t.Fatalf("array %d: encode error: %v", i, err)
		}

		got, err := wire.DecodeArray(&buf)
		if err != nil {
			t.Fatalf("array %d: decode error: %v", i, err)
		}

		if !bytes.Equal(got, arr) {
			t.Errorf("array %d: roundtrip failed: got %v, want %v", i, got, arr)
		}
	}
}

func TestDecodeEOF(t *testing.T) {
	t.Parallel()
	r := bytes.NewReader(nil)
	_, err := wire.DecodeHeader(r)
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}
