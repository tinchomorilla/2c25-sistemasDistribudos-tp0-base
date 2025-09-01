package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// SerializeMessage converts a message to custom protocol format
func SerializeMessage(message interface{}) ([]byte, error) {
	var data []byte

	switch msg := message.(type) {
	case *BetMessage:
		data = append(data, byte(MessageTypeBet))
		content := fmt.Sprintf("%s|%s|%s|%s|%d", msg.Nombre, msg.Apellido, msg.Documento, msg.Nacimiento, msg.Numero)
		data = append(data, []byte(content)...)

	case *BatchMessage:
		data = append(data, byte(MessageTypeBatch))
		content := fmt.Sprintf("%d|%d|%d", msg.Agency, boolToInt(msg.EOF), len(msg.Bets))
		for _, bet := range msg.Bets {
			content += fmt.Sprintf("|%s|%s|%s|%s|%d", bet.Nombre, bet.Apellido, bet.Documento, bet.Nacimiento, bet.Numero)
		}
		data = append(data, []byte(content)...)

	case *GetWinnersMessage:
		data = append(data, byte(MessageTypeGetWinners))
		content := fmt.Sprintf("%d", msg.Agency)
		data = append(data, []byte(content)...)

	default:
		return nil, fmt.Errorf("unsupported message type")
	}

	return data, nil
}

// DeserializeMessage parses custom protocol data into a message
func DeserializeMessage(data []byte) (interface{}, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("empty message")
	}

	msgType := MessageType(data[0])
	content := string(data[1:])

	switch msgType {
	case MessageTypeResponse:
		return parseResponseMessage(content)
	default:
		return nil, fmt.Errorf("unsupported message type: %d", msgType)
	}
}

// parseResponseMessage parses response message from pipe-delimited content
func parseResponseMessage(content string) (*ResponseMessage, error) {
	parts := strings.Split(content, "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid response format")
	}

	success := parts[0] == "1"
	errorMsg := parts[1]

	winnerCount, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid winner count: %w", err)
	}

	var winners []string
	for i := 3; i < 3+winnerCount && i < len(parts); i++ {
		if parts[i] != "" {
			winners = append(winners, parts[i])
		}
	}

	return &ResponseMessage{
		Type:    MessageTypeResponse,
		Success: success,
		Error:   errorMsg,
		Winners: winners,
	}, nil
}

// SendMessage serializes message and sends with length prefix
func SendMessage(conn net.Conn, message interface{}) error {
	data, err := SerializeMessage(message)
	if err != nil {
		return fmt.Errorf("error serializing message: %w", err)
	}

	// Send length prefix
	length := uint32(len(data))
	lengthBytes := make([]byte, LENGTH_PREFIX_BYTES)
	binary.BigEndian.PutUint32(lengthBytes, length)

	if _, err := conn.Write(lengthBytes); err != nil {
		return fmt.Errorf("error sending length: %w", err)
	}

	// Send message data
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("error sending data: %w", err)
	}

	return nil
}

// RecvMessage reads length-prefixed message and deserializes
func RecvMessage(conn net.Conn, v interface{}) error {
	// Read length prefix
	lengthBytes := make([]byte, LENGTH_PREFIX_BYTES)
	if _, err := readExact(conn, lengthBytes); err != nil {
		return fmt.Errorf("error reading length: %w", err)
	}

	length := binary.BigEndian.Uint32(lengthBytes)

	// Read message data
	data := make([]byte, length)
	if _, err := readExact(conn, data); err != nil {
		return fmt.Errorf("error reading data: %w", err)
	}

	// Deserialize message
	message, err := DeserializeMessage(data)
	if err != nil {
		return fmt.Errorf("error deserializing message: %w", err)
	}

	// Copy to destination
	switch dst := v.(type) {
	case *ResponseMessage:
		if src, ok := message.(*ResponseMessage); ok {
			*dst = *src
		} else {
			return fmt.Errorf("type mismatch: expected ResponseMessage")
		}
	default:
		return fmt.Errorf("unsupported destination type")
	}

	return nil
}

// readExact reads exactly len(buf) bytes from conn
func readExact(conn net.Conn, buf []byte) (int, error) {
	totalRead := 0
	for totalRead < len(buf) {
		n, err := conn.Read(buf[totalRead:])
		if err != nil {
			return totalRead, err
		}
		totalRead += n
	}
	return totalRead, nil
}