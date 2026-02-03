package file

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ThrottledReader wraps an io.Reader with rate limiting
type ThrottledReader struct {
	reader      io.Reader
	bytesPerSec int64
	bucket      int64
	lastTime    time.Time
}

// NewThrottledReader creates a new throttled reader
// bytesPerSec of 0 means unlimited
func NewThrottledReader(r io.Reader, bytesPerSec int64) io.Reader {
	if bytesPerSec <= 0 {
		return r
	}

	return &ThrottledReader{
		reader:      r,
		bytesPerSec: bytesPerSec,
		bucket:      bytesPerSec, // Start with a full bucket
		lastTime:    time.Now(),
	}
}

// Read implements io.Reader with rate limiting
func (t *ThrottledReader) Read(p []byte) (int, error) {
	// Refill bucket based on time elapsed
	now := time.Now()
	elapsed := now.Sub(t.lastTime).Seconds()
	t.lastTime = now

	t.bucket += int64(elapsed * float64(t.bytesPerSec))
	if t.bucket > t.bytesPerSec {
		t.bucket = t.bytesPerSec
	}

	// Limit read size to available bucket
	toRead := len(p)
	if int64(toRead) > t.bucket {
		toRead = int(t.bucket)
	}

	if toRead == 0 {
		// Sleep a bit to allow bucket to refill
		time.Sleep(10 * time.Millisecond)
		return 0, nil
	}

	n, err := t.reader.Read(p[:toRead])
	t.bucket -= int64(n)

	return n, err
}

// ThrottledWriter wraps an io.Writer with rate limiting
type ThrottledWriter struct {
	writer      io.Writer
	bytesPerSec int64
	bucket      int64
	lastTime    time.Time
}

// NewThrottledWriter creates a new throttled writer
// bytesPerSec of 0 means unlimited
func NewThrottledWriter(w io.Writer, bytesPerSec int64) io.Writer {
	if bytesPerSec <= 0 {
		return w
	}

	return &ThrottledWriter{
		writer:      w,
		bytesPerSec: bytesPerSec,
		bucket:      bytesPerSec,
		lastTime:    time.Now(),
	}
}

// Write implements io.Writer with rate limiting
func (t *ThrottledWriter) Write(p []byte) (int, error) {
	written := 0

	for written < len(p) {
		// Refill bucket based on time elapsed
		now := time.Now()
		elapsed := now.Sub(t.lastTime).Seconds()
		t.lastTime = now

		t.bucket += int64(elapsed * float64(t.bytesPerSec))
		if t.bucket > t.bytesPerSec {
			t.bucket = t.bytesPerSec
		}

		// Calculate how much we can write
		remaining := len(p) - written
		toWrite := remaining
		if int64(toWrite) > t.bucket {
			toWrite = int(t.bucket)
		}

		if toWrite == 0 {
			// Sleep a bit to allow bucket to refill
			time.Sleep(10 * time.Millisecond)
			continue
		}

		n, err := t.writer.Write(p[written : written+toWrite])
		written += n
		t.bucket -= int64(n)

		if err != nil {
			return written, err
		}
	}

	return written, nil
}

// ParseBandwidthLimit parses a bandwidth limit string like "1M", "500K", "10M"
// Returns bytes per second
func ParseBandwidthLimit(s string) (int64, error) {
	if s == "" {
		return 0, nil // Unlimited
	}

	s = strings.TrimSpace(strings.ToUpper(s))

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(s, "G"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "M"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "K"):
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		numStr = strings.TrimSuffix(s, "B")
	default:
		numStr = s
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bandwidth limit %q: %w", s, err)
	}

	if num <= 0 {
		return 0, fmt.Errorf("bandwidth limit must be positive, got %q", s)
	}

	return int64(num * float64(multiplier)), nil
}

// CopyWithContext copies from src to dst with context cancellation support
func CopyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, rerr := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return written, nil
			}
			return written, rerr
		}
	}
}
