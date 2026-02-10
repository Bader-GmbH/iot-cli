//go:build windows

package terminal

// handleResizeUnix is a no-op on Windows (no SIGWINCH support)
func (s *Session) handleResizeUnix() {
	// Windows doesn't support SIGWINCH
	// Terminal resize events would need to be handled differently on Windows
	// For now, we just return immediately
}
