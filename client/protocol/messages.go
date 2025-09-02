package protocol

// MessageType represents the type of message being sent
type MessageType byte

const (
	MessageTypeBet        MessageType = 1
	MessageTypeBatch      MessageType = 2
	MessageTypeResponse   MessageType = 3
	MessageTypeGetWinners MessageType = 4
)

// BetMessage represents a betting request from client to server
type BetMessage struct {
	Type       MessageType
	Nombre     string
	Apellido   string
	Documento  string
	Nacimiento string
	Numero     int
}

// BatchMessage represents multiple bets sent together
type BatchMessage struct {
	Type   MessageType
	Agency int // Agency number (1-5)
	Bets   []BetMessage
	EOF    bool
}

// ResponseMessage represents server response to client
type ResponseMessage struct {
	Type    MessageType
	Success bool
	Error   string
	Winners []string // List of winner DNIs
}

// GetWinnersMessage represents a request to get winners for an agency
type GetWinnersMessage struct {
	Type   MessageType
	Agency int // Agency number (1-5)
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
func NewBatchMessage(agency int, bets []BetMessage, eof bool) *BatchMessage {
	return &BatchMessage{
		Type:   MessageTypeBatch,
		Agency: agency,
		Bets:   bets,
		EOF:    eof,
	}
}

// NewGetWinnersMessage creates a new get winners message
func NewGetWinnersMessage(agency int) *GetWinnersMessage {
	return &GetWinnersMessage{
		Type:   MessageTypeGetWinners,
		Agency: agency,
	}
}

// boolToInt converts boolean to integer
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}