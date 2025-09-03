package common

import (
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	Nombre        string
	Apellido      string
	Documento     string
	Nacimiento    string
	Numero        int
}

// Client Entity that encapsulates how
type Client struct {
	config            ClientConfig
	conn              net.Conn
}

// NewClient Initializes a new client receiving the configuration
func NewClient(config ClientConfig) *Client {
	client := &Client{config: config}

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		log.Infof("action: shutdown | result: in_progress | client_id: %v | msg: received SIGTERM", client.config.ID)

		if client.conn != nil {
			client.conn.Close()
			log.Infof("action: shutdown | result: success | client_id: %v | msg: socket closed", client.config.ID)
		}
	}()

	return client
}

// CreateClientSocket Initializes client socket
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send bet messages until threshold or shutdown signal
func (c *Client) StartClientLoop() {
	defer c.conn.Close()

	if err := c.createClientSocket(); err != nil {
		log.Errorf("action: create_socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
		
	}

	// Create bet message
	betMessage := NewBetMessage(
		c.config.Nombre,
		c.config.Apellido,
		c.config.Documento,
		c.config.Nacimiento,
		c.config.Numero,
	)

	// Send bet message
	if err := SendMessage(c.conn, betMessage); err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	// Wait for server response
	var response ResponseMessage
	if err := RecvMessage(c.conn, &response); err != nil {
		// Check if server closed the connection gracefully
		if err == io.EOF {
			log.Infof("action: receive_message | result: server_shutdown | client_id: %v | msg: server closed connection",
				c.config.ID,
			)
		} else {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
		} 
		return
	}

	// Log result based on server response
	if response.Success {
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
			c.config.Documento,
			c.config.Numero,
		)
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %v | numero: %v | error: %v",
			c.config.Documento,
			c.config.Numero,
			response.Error,
		)
	}
	
}
