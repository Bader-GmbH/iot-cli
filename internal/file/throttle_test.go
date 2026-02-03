package file

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseBandwidthLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "bytes",
			input: "1024",
			want:  1024,
		},
		{
			name:  "bytes with B suffix",
			input: "1024B",
			want:  1024,
		},
		{
			name:  "kilobytes K",
			input: "1K",
			want:  1024,
		},
		{
			name:  "kilobytes KB",
			input: "1KB",
			want:  1024,
		},
		{
			name:  "kilobytes 500K",
			input: "500K",
			want:  512000,
		},
		{
			name:  "megabytes M",
			input: "1M",
			want:  1048576,
		},
		{
			name:  "megabytes MB",
			input: "1MB",
			want:  1048576,
		},
		{
			name:  "megabytes 10M",
			input: "10M",
			want:  10485760,
		},
		{
			name:  "gigabytes G",
			input: "1G",
			want:  1073741824,
		},
		{
			name:  "gigabytes GB",
			input: "1GB",
			want:  1073741824,
		},
		{
			name:  "lowercase",
			input: "1m",
			want:  1048576,
		},
		{
			name:  "with whitespace",
			input: "  500K  ",
			want:  512000,
		},
		{
			name:  "decimal value",
			input: "1.5M",
			want:  1572864,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "negative value",
			input:   "-100K",
			wantErr: true,
		},
		{
			name:    "zero value",
			input:   "0K",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBandwidthLimit(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseBandwidthLimit(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseBandwidthLimit(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got != tt.want {
				t.Errorf("ParseBandwidthLimit(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewThrottledReader_ZeroLimit(t *testing.T) {
	original := strings.NewReader("test data")
	reader := NewThrottledReader(original, 0)

	// With 0 limit, should return the original reader unchanged
	if reader != original {
		t.Error("NewThrottledReader with 0 limit should return original reader")
	}
}

func TestNewThrottledReader_NegativeLimit(t *testing.T) {
	original := strings.NewReader("test data")
	reader := NewThrottledReader(original, -100)

	// With negative limit, should return the original reader unchanged
	if reader != original {
		t.Error("NewThrottledReader with negative limit should return original reader")
	}
}

func TestNewThrottledReader_ReturnsThrottledReader(t *testing.T) {
	original := strings.NewReader("test data")
	reader := NewThrottledReader(original, 1024)

	// With positive limit, should return ThrottledReader
	if _, ok := reader.(*ThrottledReader); !ok {
		t.Error("NewThrottledReader with positive limit should return *ThrottledReader")
	}
}

func TestThrottledReader_ReadsAllData(t *testing.T) {
	data := "hello world"
	reader := NewThrottledReader(strings.NewReader(data), 1024*1024) // 1MB/s - fast enough for test

	result, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != data {
		t.Errorf("got %q, want %q", string(result), data)
	}
}

func TestNewThrottledWriter_ZeroLimit(t *testing.T) {
	var buf bytes.Buffer
	writer := NewThrottledWriter(&buf, 0)

	// With 0 limit, should return the original writer unchanged
	if writer != &buf {
		t.Error("NewThrottledWriter with 0 limit should return original writer")
	}
}

func TestNewThrottledWriter_NegativeLimit(t *testing.T) {
	var buf bytes.Buffer
	writer := NewThrottledWriter(&buf, -100)

	// With negative limit, should return the original writer unchanged
	if writer != &buf {
		t.Error("NewThrottledWriter with negative limit should return original writer")
	}
}

func TestNewThrottledWriter_ReturnsThrottledWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewThrottledWriter(&buf, 1024)

	// With positive limit, should return ThrottledWriter
	if _, ok := writer.(*ThrottledWriter); !ok {
		t.Error("NewThrottledWriter with positive limit should return *ThrottledWriter")
	}
}

func TestThrottledWriter_WritesAllData(t *testing.T) {
	var buf bytes.Buffer
	data := "hello world"
	writer := NewThrottledWriter(&buf, 1024*1024) // 1MB/s - fast enough for test

	n, err := writer.Write([]byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}

	if buf.String() != data {
		t.Errorf("got %q, want %q", buf.String(), data)
	}
}

func TestCopyWithContext_Success(t *testing.T) {
	data := "test data for copy"
	src := strings.NewReader(data)
	var dst bytes.Buffer

	ctx := context.Background()
	written, err := CopyWithContext(ctx, &dst, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, len(data))
	}

	if dst.String() != data {
		t.Errorf("got %q, want %q", dst.String(), data)
	}
}

func TestCopyWithContext_Cancelled(t *testing.T) {
	// Create a large reader that will take time to read
	data := strings.Repeat("x", 1024*1024) // 1MB
	src := strings.NewReader(data)
	var dst bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := CopyWithContext(ctx, &dst, src)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestCopyWithContext_Timeout(t *testing.T) {
	// Use a slow reader to test timeout
	src := &slowReader{data: strings.Repeat("x", 1024), delay: 100 * time.Millisecond}
	var dst bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := CopyWithContext(ctx, &dst, src)

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestCopyWithContext_EmptyReader(t *testing.T) {
	src := strings.NewReader("")
	var dst bytes.Buffer

	ctx := context.Background()
	written, err := CopyWithContext(ctx, &dst, src)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if written != 0 {
		t.Errorf("written = %d, want 0", written)
	}
}

// slowReader is a reader that delays each read operation
type slowReader struct {
	data  string
	pos   int
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	time.Sleep(r.delay)

	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBytes(tt.input)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}