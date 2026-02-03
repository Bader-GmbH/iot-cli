package terminal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

// Session manages a terminal session over WebSocket
type Session struct {
	conn        *websocket.Conn
	sessionID   string
	accessToken string
	tenantID    string
	done        chan struct{}
	oldState    *term.State
}

// Connect establishes a WebSocket connection to the terminal session
func Connect(ctx context.Context, baseURL, sessionID, accessToken, tenantID string) (*Session, error) {
	// Convert HTTP URL to WebSocket URL
	wsURL, err := buildWebSocketURL(baseURL, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to build WebSocket URL: %w", err)
	}

	// Set up headers for authentication
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+accessToken)
	headers.Set("X-Tenant-ID", tenantID)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("WebSocket connection failed (status %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("WebSocket connection failed: %w", err)
	}

	return &Session{
		conn:        conn,
		sessionID:   sessionID,
		accessToken: accessToken,
		tenantID:    tenantID,
		done:        make(chan struct{}),
	}, nil
}

// buildWebSocketURL converts the API base URL to a WebSocket URL
func buildWebSocketURL(baseURL, sessionID string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Convert http(s) to ws(s)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}

	// Set the path to the terminal WebSocket endpoint
	u.Path = "/ws/terminal"
	q := u.Query()
	q.Set("sessionId", sessionID)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// Run starts the terminal session, piping stdin/stdout through the WebSocket
func (s *Session) Run() error {
	// Set terminal to raw mode
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("failed to set raw mode: %w", err)
		}
		s.oldState = oldState
		defer s.restoreTerminal()
	}

	// Handle Ctrl+C and other signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sigChan
		s.Close()
	}()

	// Handle window size changes
	go s.handleResize()

	// Send initial terminal size
	s.sendSize()

	// Start reading from WebSocket and writing to stdout
	go s.readLoop()

	// Read from stdin and write to WebSocket
	s.writeLoop()

	return nil
}

// readLoop reads from WebSocket and writes to stdout
func (s *Session) readLoop() {
	defer s.Close()

	for {
		select {
		case <-s.done:
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return
				}
				// Connection closed
				return
			}

			// Write to stdout
			os.Stdout.Write(message)
		}
	}
}

// writeLoop reads from stdin and writes to WebSocket
func (s *Session) writeLoop() {
	buf := make([]byte, 1024)

	for {
		select {
		case <-s.done:
			return
		default:
			n, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF {
					s.Close()
					return
				}
				continue
			}

			if n > 0 {
				err := s.conn.WriteMessage(websocket.BinaryMessage, buf[:n])
				if err != nil {
					s.Close()
					return
				}
			}
		}
	}
}

// sendSize sends the terminal size to the remote
func (s *Session) sendSize() {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return
	}

	width, height, err := term.GetSize(fd)
	if err != nil {
		return
	}

	// Send resize message (protocol: 0x04 + width(2) + height(2))
	msg := []byte{0x04, byte(width >> 8), byte(width), byte(height >> 8), byte(height)}
	s.conn.WriteMessage(websocket.BinaryMessage, msg)
}

// handleResize watches for terminal resize signals
func (s *Session) handleResize() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	for {
		select {
		case <-s.done:
			return
		case <-sigChan:
			s.sendSize()
		}
	}
}

// restoreTerminal restores the terminal to its original state
func (s *Session) restoreTerminal() {
	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
}

// Close closes the terminal session
func (s *Session) Close() {
	select {
	case <-s.done:
		// Already closed
		return
	default:
		close(s.done)
	}

	s.restoreTerminal()

	if s.conn != nil {
		s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		s.conn.Close()
	}
}
