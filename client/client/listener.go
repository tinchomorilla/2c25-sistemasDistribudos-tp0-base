package client

import (
	"net"
	"time"

	"github.com/op/go-logging"
	"github.com/tinchomorilla/2c25-sistemasDistribudos-tp0-base/client/protocol"
)

// Listener handles requesting and receiving winners from the server
type Listener struct {
	conn     net.Conn
	agency   int
	clientID string
	log      *logging.Logger
}

// NewListener creates a new Listener instance
func NewListener(conn net.Conn, agency int, clientID string) *Listener {
	return &Listener{
		conn:     conn,
		agency:   agency,
		clientID: clientID,
		log:      log,
	}
}

// RequestWinners requests the list of winners from the server using the existing connection
func (l *Listener) RequestWinners(shutdownRequested *bool) {
	l.log.Infof("action: request_winners | result: in_progress | client_id: %v | msg: waiting for lottery", l.clientID)

	const maxRetries = 10
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if *shutdownRequested {
			return
		}

		l.log.Infof("action: request_winners | result: in_progress | attempt: %d | client_id: %v", attempt, l.clientID)

		if attempt > 1 {
			time.Sleep(retryDelay)
		}

		// Use existing connection for this attempt
		success := l.tryGetWinners(attempt, shutdownRequested)

		if success {
			return // Successfully got winners
		}

		// If this was the last attempt, we've exhausted all retries
		if attempt == maxRetries {
			l.log.Errorf("action: consulta_ganadores | result: fail | max_attempts_reached")
			return
		}
	}
}

// tryGetWinners attempts to get winners using the current connection
func (l *Listener) tryGetWinners(attempt int, shutdownRequested *bool) bool {
	// Create get winners message
	winnersMessage := protocol.NewGetWinnersMessage(l.agency)

	// Send get winners message
	if err := protocol.SendMessage(l.conn, winnersMessage); err != nil {
		l.log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
			attempt, l.clientID, err)
		return false
	}

	// Wait for server response
	var response protocol.ResponseMessage
	if err := protocol.RecvMessage(l.conn, &response); err != nil {
		if *shutdownRequested {
			return false
		}
		l.log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
			attempt, l.clientID, err)
		return false
	}

	// Check server response
	if response.Success {
		l.log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d",
			len(response.Winners))
		return true
	} else {
		l.log.Infof("action: request_winners | result: in_progress | attempt: %d | client_id: %v | error: %v",
			attempt, l.clientID, response.Error)
		return false
	}
}
