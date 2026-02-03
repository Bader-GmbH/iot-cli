package terminal

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// Protocol constants for b-agent binary protocol
const (
	HeaderLength        = 120
	MessageTypeLength   = 32
	MessageIDLength     = 16
	PayloadDigestLength = 32
	SchemaVersion       = 1
)

// Message types
const (
	MessageTypeOutputStream = "output_stream_data"
)

// Payload types
const (
	PayloadTypeOutput            = 1
	PayloadTypeError             = 2
	PayloadTypeSize              = 3
	PayloadTypeParameter         = 4
	PayloadTypeHandshakeRequest  = 5
	PayloadTypeHandshakeResp     = 6
	PayloadTypeHandshakeComplete = 7
	PayloadTypeExitCode          = 12
)

// AgentMessage represents a parsed b-agent protocol message
type AgentMessage struct {
	HeaderLength   int
	MessageType    string
	SchemaVersion  int
	CreatedDate    int64
	SequenceNumber int64
	Flags          int64
	MessageID      []byte
	PayloadDigest  []byte
	PayloadType    int
	PayloadLength  int
	Payload        []byte
}

// ParseMessage parses a b-agent protocol message from raw bytes
func ParseMessage(data []byte) (*AgentMessage, error) {
	if len(data) < HeaderLength {
		return nil, errors.New("message too short")
	}

	msg := &AgentMessage{}
	offset := 0

	// Header length (4 bytes, big-endian)
	msg.HeaderLength = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// Message type (32 bytes, null-terminated string)
	msgTypeBytes := data[offset : offset+MessageTypeLength]
	msg.MessageType = trimNullBytes(msgTypeBytes)
	offset += MessageTypeLength

	// Schema version (4 bytes)
	msg.SchemaVersion = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// Created date (8 bytes, milliseconds)
	msg.CreatedDate = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// Sequence number (8 bytes)
	msg.SequenceNumber = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// Flags (8 bytes)
	msg.Flags = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// Message ID (16 bytes UUID)
	msg.MessageID = make([]byte, MessageIDLength)
	copy(msg.MessageID, data[offset:offset+MessageIDLength])
	offset += MessageIDLength

	// Payload digest (32 bytes SHA-256)
	msg.PayloadDigest = make([]byte, PayloadDigestLength)
	copy(msg.PayloadDigest, data[offset:offset+PayloadDigestLength])
	offset += PayloadDigestLength

	// Payload type (4 bytes)
	msg.PayloadType = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// Payload length (4 bytes)
	msg.PayloadLength = int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// Payload
	if len(data) >= offset+msg.PayloadLength {
		msg.Payload = make([]byte, msg.PayloadLength)
		copy(msg.Payload, data[offset:offset+msg.PayloadLength])
	}

	return msg, nil
}

// IsOutput returns true if this is an output message (terminal data)
func (m *AgentMessage) IsOutput() bool {
	return m.PayloadType == PayloadTypeOutput
}

// IsHandshakeComplete returns true if this is a handshake complete message
func (m *AgentMessage) IsHandshakeComplete() bool {
	return m.PayloadType == PayloadTypeHandshakeComplete
}

// IsExitCode returns true if this is an exit code message
func (m *AgentMessage) IsExitCode() bool {
	return m.PayloadType == PayloadTypeExitCode
}

// trimNullBytes removes null bytes from a byte slice and returns a string
func trimNullBytes(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// BuildMessage builds a b-agent protocol message
func BuildMessage(messageType string, payloadType int, payload []byte, sequenceNumber int64) []byte {
	buf := make([]byte, HeaderLength+len(payload))
	offset := 0

	// Header length (4 bytes, big-endian)
	binary.BigEndian.PutUint32(buf[offset:], uint32(HeaderLength))
	offset += 4

	// Message type (32 bytes, null-padded)
	copy(buf[offset:offset+MessageTypeLength], messageType)
	offset += MessageTypeLength

	// Schema version (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:], uint32(SchemaVersion))
	offset += 4

	// Created date (8 bytes, milliseconds)
	binary.BigEndian.PutUint64(buf[offset:], uint64(time.Now().UnixMilli()))
	offset += 8

	// Sequence number (8 bytes)
	binary.BigEndian.PutUint64(buf[offset:], uint64(sequenceNumber))
	offset += 8

	// Flags (8 bytes)
	binary.BigEndian.PutUint64(buf[offset:], 0)
	offset += 8

	// Message ID (16 bytes UUID)
	_, _ = rand.Read(buf[offset : offset+MessageIDLength])
	offset += MessageIDLength

	// Payload digest (32 bytes SHA-256)
	digest := sha256.Sum256(payload)
	copy(buf[offset:offset+PayloadDigestLength], digest[:])
	offset += PayloadDigestLength

	// Payload type (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:], uint32(payloadType))
	offset += 4

	// Payload length (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(payload)))
	offset += 4

	// Payload
	copy(buf[offset:], payload)

	return buf
}

// BuildInputMessage builds a terminal input message
func BuildInputMessage(input []byte, sequenceNumber int64) []byte {
	return BuildMessage(MessageTypeOutputStream, PayloadTypeOutput, input, sequenceNumber)
}

// BuildResizeMessage builds a terminal resize message
func BuildResizeMessage(cols, rows int, sequenceNumber int64) []byte {
	payload := []byte(fmt.Sprintf(`{"cols":%d,"rows":%d}`, cols, rows))
	return BuildMessage(MessageTypeOutputStream, PayloadTypeSize, payload, sequenceNumber)
}
