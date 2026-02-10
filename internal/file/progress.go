package file

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// ProgressWriter wraps an io.Writer and reports progress
type ProgressWriter struct {
	writer    io.Writer
	total     int64
	current   int64
	filename  string
	startTime time.Time
	lastPrint time.Time
	mu        sync.Mutex
	quiet     bool
}

// NewProgressWriter creates a new progress writer
func NewProgressWriter(w io.Writer, total int64, filename string, quiet bool) *ProgressWriter {
	return &ProgressWriter{
		writer:    w,
		total:     total,
		filename:  filename,
		startTime: time.Now(),
		quiet:     quiet,
	}
}

// Write implements io.Writer
func (p *ProgressWriter) Write(data []byte) (int, error) {
	n, err := p.writer.Write(data)

	p.mu.Lock()
	p.current += int64(n)
	p.mu.Unlock()

	p.printProgress()
	return n, err
}

// printProgress prints the progress bar
func (p *ProgressWriter) printProgress() {
	if p.quiet {
		return
	}

	// Rate limit printing to every 100ms
	p.mu.Lock()
	if time.Since(p.lastPrint) < 100*time.Millisecond {
		p.mu.Unlock()
		return
	}
	p.lastPrint = time.Now()
	current := p.current
	p.mu.Unlock()

	// Calculate percentage
	var percent float64
	if p.total > 0 {
		percent = float64(current) / float64(p.total) * 100
	}

	// Calculate speed
	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(current) / elapsed
	}

	// Calculate ETA
	var eta string
	if speed > 0 && p.total > 0 {
		remaining := float64(p.total-current) / speed
		if remaining < 60 {
			eta = fmt.Sprintf("%.0fs", remaining)
		} else if remaining < 3600 {
			eta = fmt.Sprintf("%.0fm", remaining/60)
		} else {
			eta = fmt.Sprintf("%.1fh", remaining/3600)
		}
	} else {
		eta = "--"
	}

	// Build progress bar
	barWidth := 30
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 80 {
		barWidth = 40
	}

	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format sizes
	currentStr := formatBytes(current)
	totalStr := formatBytes(p.total)
	speedStr := formatBytes(int64(speed)) + "/s"

	// Print progress (with carriage return to overwrite)
	fmt.Printf("\r[%s] %.1f%% (%s / %s) %s ETA %s  ",
		bar, percent, currentStr, totalStr, speedStr, eta)
}

// Finish prints the final progress and newline
func (p *ProgressWriter) Finish() {
	if p.quiet {
		return
	}

	p.mu.Lock()
	current := p.current
	p.mu.Unlock()

	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(current) / elapsed
	}

	// Clear line and print final status
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	fmt.Printf("  %s  %s  %s/s  %.1fs\n",
		p.filename,
		formatBytes(current),
		formatBytes(int64(speed)),
		elapsed)
}

// ProgressReader wraps an io.Reader and reports progress
type ProgressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	filename  string
	startTime time.Time
	lastPrint time.Time
	mu        sync.Mutex
	quiet     bool
}

// NewProgressReader creates a new progress reader
func NewProgressReader(r io.Reader, total int64, filename string, quiet bool) *ProgressReader {
	return &ProgressReader{
		reader:    r,
		total:     total,
		filename:  filename,
		startTime: time.Now(),
		quiet:     quiet,
	}
}

// Read implements io.Reader
func (p *ProgressReader) Read(data []byte) (int, error) {
	n, err := p.reader.Read(data)

	p.mu.Lock()
	p.current += int64(n)
	p.mu.Unlock()

	p.printProgress()
	return n, err
}

// printProgress prints the progress bar (same as ProgressWriter)
func (p *ProgressReader) printProgress() {
	if p.quiet {
		return
	}

	p.mu.Lock()
	if time.Since(p.lastPrint) < 100*time.Millisecond {
		p.mu.Unlock()
		return
	}
	p.lastPrint = time.Now()
	current := p.current
	p.mu.Unlock()

	var percent float64
	if p.total > 0 {
		percent = float64(current) / float64(p.total) * 100
	}

	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(current) / elapsed
	}

	var eta string
	if speed > 0 && p.total > 0 {
		remaining := float64(p.total-current) / speed
		if remaining < 60 {
			eta = fmt.Sprintf("%.0fs", remaining)
		} else if remaining < 3600 {
			eta = fmt.Sprintf("%.0fm", remaining/60)
		} else {
			eta = fmt.Sprintf("%.1fh", remaining/3600)
		}
	} else {
		eta = "--"
	}

	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	currentStr := formatBytes(current)
	totalStr := formatBytes(p.total)
	speedStr := formatBytes(int64(speed)) + "/s"

	fmt.Printf("\r[%s] %.1f%% (%s / %s) %s ETA %s  ",
		bar, percent, currentStr, totalStr, speedStr, eta)
}

// Finish prints the final progress and newline
func (p *ProgressReader) Finish() {
	if p.quiet {
		return
	}

	p.mu.Lock()
	current := p.current
	p.mu.Unlock()

	elapsed := time.Since(p.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(current) / elapsed
	}

	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	fmt.Printf("  %s  %s  %s/s  %.1fs\n",
		p.filename,
		formatBytes(current),
		formatBytes(int64(speed)),
		elapsed)
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatBytes is exported for use by commands
func FormatBytes(bytes int64) string {
	return formatBytes(bytes)
}
