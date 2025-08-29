package common

import (
	"encoding/csv"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	LoopAmount     int
	LoopPeriod     time.Duration
	CSVFile        string
	BatchMaxAmount int
	Agency         int
}

// Client Entity that encapsulates how
type Client struct {
	config            ClientConfig
	conn              net.Conn
	shutdownRequested bool
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
		client.shutdownRequested = true

		if client.conn != nil {
			client.conn.Close()
			log.Infof("action: shutdown | result: success | client_id: %v | msg: socket closed", client.config.ID)
		}
	}()

	return client
}

// BetRecord represents a bet record from CSV
type BetRecord struct {
	Nombre     string
	Apellido   string
	Documento  string
	Nacimiento string
	Numero     int
}

// readBetsFromCSV reads bet records from a CSV file
func (c *Client) readBetsFromCSV() ([]BetRecord, error) {
	file, err := os.Open(c.config.CSVFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var bets []BetRecord

	for {

		if c.shutdownRequested {
			break
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Assuming CSV format: nombre,apellido,documento,nacimiento,numero
		if len(record) != 5 {
			continue // Skip malformed records
		}

		numero, err := strconv.Atoi(record[4])
		if err != nil {
			continue // Skip records with invalid numbers
		}

		bet := BetRecord{
			Nombre:     record[0],
			Apellido:   record[1],
			Documento:  record[2],
			Nacimiento: record[3],
			Numero:     numero,
		}
		bets = append(bets, bet)
	}

	return bets, nil
}

// calculateMessageSize estimates the custom protocol size of a batch message
func (c *Client) calculateMessageSize(bets []BetMessage) int {
	batch := NewBatchMessage(c.config.Agency, bets, false) // Use false for size calculation
	data, err := SerializeMessage(batch)
	if err != nil {
		return 0
	}
	return len(data) + 4 // +4 for length prefix
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

// StartClientLoopWithCSV reads bets from CSV and sends them in batches (Exercise 6 & 7)
func (c *Client) StartClientWithCSV() {
	log.Infof("action: start_csv_client | result: in_progress | client_id: %v | csv_file: %s", c.config.ID, c.config.CSVFile)

	// Read bets from CSV file
	betsFromCSV, err := c.readBetsFromCSV()
	if err != nil {
		log.Errorf("action: read_csv | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	log.Infof("action: read_csv | result: success | client_id: %v | bets_count: %d", c.config.ID, len(betsFromCSV))

	// Convert CSV records to BetMessages and send in batches
	var currentBatch []BetMessage
	const MAX_BATCH_SIZE_BYTES = 8 * 1024 // 8kB limit

	for _, betRecord := range betsFromCSV {
		if c.shutdownRequested {
			break
		}

		// Create bet message from CSV record
		betMsg := BetMessage{
			Type:       MessageTypeBet,
			Nombre:     betRecord.Nombre,
			Apellido:   betRecord.Apellido,
			Documento:  betRecord.Documento,
			Nacimiento: betRecord.Nacimiento,
			Numero:     betRecord.Numero,
		}

		// Check if adding this bet would exceed size or count limits
		testBatch := append(currentBatch, betMsg)
		batchSize := c.calculateMessageSize(testBatch)

		if len(testBatch) > c.config.BatchMaxAmount || batchSize > MAX_BATCH_SIZE_BYTES {
			c.sendBatch(currentBatch, false)
			// Start new batch with current bet
			currentBatch = []BetMessage{betMsg}
		} else {
			// Add to current batch
			currentBatch = testBatch
		}
	}

	if len(currentBatch) > 0 && !c.shutdownRequested {
		c.sendBatch(currentBatch, true) // EOF = true for the last batch
	}

	// After sending all bets with EOF, request winners
	if !c.shutdownRequested {
		c.requestWinners()
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

// sendBatch sends a batch of bets to the server
func (c *Client) sendBatch(bets []BetMessage, eof bool) {
	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close()

	// Create batch message with agency number
	batchMessage := NewBatchMessage(c.config.Agency, bets, eof)

	// Send batch message
	if err := SendMessage(c.conn, batchMessage); err != nil {
		log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}

}

// requestWinners requests the list of winners from the server
func (c *Client) requestWinners() {
	log.Infof("action: request_winners | result: in_progress | client_id: %v | msg: waiting for lottery", c.config.ID)

	const maxRetries = 10
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if c.shutdownRequested {
			return
		}

		log.Infof("action: request_winners | result: in_progress | attempt: %d | client_id: %v", attempt, c.config.ID)
		time.Sleep(retryDelay)

		// Create connection for this attempt (server closes it after each request)
		if err := c.createClientSocket(); err != nil {
			log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
				attempt, c.config.ID, err)
			continue
		}

		// Ensure connection is closed when we're done with this attempt
		success := c.tryGetWinners(attempt)
		c.conn.Close()

		if success {
			return // Successfully got winners
		}

		// If this was the last attempt, we've exhausted all retries
		if attempt == maxRetries {
			log.Errorf("action: consulta_ganadores | result: fail | max_attempts_reached")
			return
		}
	}
}

// tryGetWinners attempts to get winners using the current connection
func (c *Client) tryGetWinners(attempt int) bool {
	// Create get winners message
	winnersMessage := NewGetWinnersMessage(c.config.Agency)

	// Send get winners message
	if err := SendMessage(c.conn, winnersMessage); err != nil {
		log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
			attempt, c.config.ID, err)
		return false
	}

	// Wait for server response
	var response ResponseMessage
	if err := RecvMessage(c.conn, &response); err != nil {
		if c.shutdownRequested {
			return false
		}
		log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
			attempt, c.config.ID, err)
		return false
	}

	// Check server response
	if response.Success {
		// Success! Print amount of winners
		log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d",
			len(response.Winners))
		return true
	} else {
		// Server returned an error (probably lottery not ready yet)
		log.Infof("action: request_winners | result: retry | attempt: %d | client_id: %v | error: %v",
			attempt, c.config.ID, response.Error)
		return false
	}
}
