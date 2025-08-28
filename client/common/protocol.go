package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

// MessageType represents the type of message being sent
type MessageType string

const (
	MessageTypeBet      MessageType = "bet"
	MessageTypeBatch    MessageType = "batch" 
	MessageTypeResponse MessageType = "response"
)

// BetMessage represents a betting request from client to server
type BetMessage struct {
	Type       MessageType `json:"type"`
	Nombre     string      `json:"nombre"`
	Apellido   string      `json:"apellido"`
	Documento  string      `json:"documento"`
	Nacimiento string      `json:"nacimiento"`
	Numero     int         `json:"numero"`
}

// BatchMessage represents multiple bets sent together
type BatchMessage struct {
	Type MessageType  `json:"type"`
	Bets []BetMessage `json:"bets"`
}

// ResponseMessage represents server response to client
type ResponseMessage struct {
	Type    MessageType `json:"type"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
}

// NewBetMessage creates a new bet message from the provided data
func NewBetMessage(nombre, apellido, documento, nacimiento string, numero int) *BetMessage {
	return &BetMessage{
		Type:       MessageTypeBet,
		Nombre:     nombre,
		Apellido:   apellido,
		Documento:  documento,
		Nacimiento: nacimiento,
		Numero:     numero,
	}
}

// NewBatchMessage creates a new batch message from a slice of bets
func NewBatchMessage(bets []BetMessage) *BatchMessage {
	return &BatchMessage{
		Type: MessageTypeBatch,
		Bets: bets,
	}
}

// SendMessage serializes message to JSON and sends with newline
func SendMessage(conn net.Conn, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}
	data = append(data, '\n') // delimiter
	_, err = conn.Write(data)
	return err
}

// RecvMessage reads until newline and decodes JSON
func RecvMessage(conn net.Conn, v interface{}) error {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}
	return json.Unmarshal(line, v)
}
