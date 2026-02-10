//go:build !windows

package terminal

import (
	"os"
	"os/signal"
	"syscall"
)

// handleResizeUnix handles terminal resize signals on Unix systems
func (s *Session) handleResizeUnix() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	for {
		select {
		case <-s.done:
			signal.Stop(sigChan)
			return
		case <-sigChan:
			s.sendSize()
		}
	}
}
