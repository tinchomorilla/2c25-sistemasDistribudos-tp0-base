package client

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/tinchomorilla/2c25-sistemasDistribudos-tp0-base/client/common"
	"github.com/tinchomorilla/2c25-sistemasDistribudos-tp0-base/client/protocol"
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

	// Goroutine to handle shutdown signal
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

// calculateMessageSize estimates the custom protocol size of a batch message
func (c *Client) calculateMessageSize(bets []protocol.BetMessage) int {
	batch := protocol.NewBatchMessage(c.config.Agency, bets, false) // Use false for size calculation
	data, err := protocol.SerializeMessage(batch)
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

// validateBetRecord validates a CSV record and returns true if valid
func (c *Client) validateBetRecord(record []string, numero_aux *int) error {
	// Check basic format: must have exactly 5 fields
	if len(record) != common.EXPECTED_CSV_FIELDS {
		return fmt.Errorf("expected %d fields, got %d", common.EXPECTED_CSV_FIELDS, len(record))
	}

	// Check for empty or whitespace-only fields
	for i, field := range record[:common.CSV_INDEX_NUMERO] {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			return fmt.Errorf("field %d is empty or whitespace", i)
		}
		record[i] = trimmed // Update with trimmed value
	}

	// Validate field lengths
	if len(record[common.CSV_INDEX_NOMBRE]) > common.MAX_NOMBRE_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", common.FIELD_NOMBRE, common.MAX_NOMBRE_BYTES, len(record[common.CSV_INDEX_NOMBRE]))
	}
	if len(record[common.CSV_INDEX_APELLIDO]) > common.MAX_APELLIDO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", common.FIELD_APELLIDO, common.MAX_APELLIDO_BYTES, len(record[common.CSV_INDEX_APELLIDO]))
	}
	if len(record[common.CSV_INDEX_DOCUMENTO]) > common.MAX_DOCUMENTO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", common.FIELD_DOCUMENTO, common.MAX_DOCUMENTO_BYTES, len(record[common.CSV_INDEX_DOCUMENTO]))
	}
	if len(record[common.CSV_INDEX_NACIMIENTO]) > common.MAX_NACIMIENTO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", common.FIELD_NACIMIENTO, common.MAX_NACIMIENTO_BYTES, len(record[common.CSV_INDEX_NACIMIENTO]))
	}

	// Validate numero field
	numero, err := strconv.Atoi(strings.TrimSpace(record[common.CSV_INDEX_NUMERO]))
	if err != nil {
		return fmt.Errorf("invalid %s: %s", common.FIELD_NUMERO, record[common.CSV_INDEX_NUMERO])
	}
	if numero < common.MIN_NUMERO || numero > common.MAX_NUMERO {
		return fmt.Errorf("%s out of range [%d-%d]: %d", common.FIELD_NUMERO, common.MIN_NUMERO, common.MAX_NUMERO, numero)
	}

	*numero_aux = numero
	return nil
}

// StartClientWithCSV processes CSV file in streaming mode without loading all records in memory
func (c *Client) StartClientWithCSV() {
	// Open connection once at the beginning
	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close() // Close connection when function exits

	file, err := os.Open(c.config.CSVFile)
	if err != nil {
		log.Errorf("action: open_csv | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var currentBatch []protocol.BetMessage
	recordCount := 0
	validCount := 0
	var numero_apostado int

	for {
		if c.shutdownRequested {
			break
		}

		// Check if we've reached the maximum number of valid records
		if validCount >= common.MAX_CSV_RECORDS {
			log.Infof("action: read_csv_streaming | result: limit_reached | client_id: %v | max_records: %d | stopping early",
				c.config.ID, common.MAX_CSV_RECORDS)
			break
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("action: read_csv_streaming | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		recordCount++

		// Validate record using our validation function
		if err := c.validateBetRecord(record, &numero_apostado); err != nil {
			log.Debugf("action: validate_record | result: fail | line: %d | error: %v",
				recordCount, err)
			continue // Skip invalid records
		}

		// Create bet record with trimmed fields
		bet := protocol.BetMessage{
			Type:       protocol.MessageTypeBet,
			Nombre:     record[common.CSV_INDEX_NOMBRE],
			Apellido:   record[common.CSV_INDEX_APELLIDO],
			Documento:  record[common.CSV_INDEX_DOCUMENTO],
			Nacimiento: record[common.CSV_INDEX_NACIMIENTO],
			Numero:     numero_apostado,
		}

		// Check if adding this bet would exceed size or count limits
		testBatch := append(currentBatch, bet)
		batchSize := c.calculateMessageSize(testBatch)

		if len(testBatch) > c.config.BatchMaxAmount || batchSize > common.MAX_BATCH_SIZE_BYTES {
			c.sendBatch(currentBatch, false)
			// Start new batch with current bet
			currentBatch = []protocol.BetMessage{bet}
		} else {
			// Add to current batch
			currentBatch = testBatch
		}

		validCount++
	}

	if len(currentBatch) > 0 && !c.shutdownRequested {
		c.sendBatch(currentBatch, true) // EOF = true for the last batch
	}

	// After sending all bets with EOF, request winners
	if !c.shutdownRequested {
		c.requestWinners()
	}

	log.Infof("action: csv_streaming_finished | result: success | client_id: %v | total_records: %d | valid_records: %d | skipped: %d",
		c.config.ID, recordCount, validCount, recordCount-validCount)
}

// sendBatch sends a batch of bets to the server using an existing connection
func (c *Client) sendBatch(bets []protocol.BetMessage, eof bool) error {
	// Create batch message with agency number
	batchMessage := protocol.NewBatchMessage(c.config.Agency, bets, eof)

	for i := 0; i < common.MAX_SEND_RETRIES; i++ {
		// Try to send the message
		err := protocol.SendMessage(c.conn, batchMessage)
		if err == nil {
			return nil // Batch sent successfully
		}
		// Error occurred, log it
		log.Errorf("action: send_batch | result: fail | attempt: %d | client_id: %v | error: %v",
			i+1, c.config.ID, err)
	}

	return fmt.Errorf("failed to send batch after %d attempts", common.MAX_SEND_RETRIES)
}

// requestWinners requests the list of winners from the server using the existing connection
func (c *Client) requestWinners() {
	log.Infof("action: request_winners | result: in_progress | client_id: %v | msg: waiting for lottery", c.config.ID)

	const maxRetries = 10
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if c.shutdownRequested {
			return
		}

		log.Infof("action: request_winners | result: in_progress | attempt: %d | client_id: %v", attempt, c.config.ID)

		if attempt > 1 {
			time.Sleep(retryDelay)
		}

		// Use existing connection for this attempt
		success := c.tryGetWinners(attempt)

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
	winnersMessage := protocol.NewGetWinnersMessage(c.config.Agency)

	// Send get winners message
	if err := protocol.SendMessage(c.conn, winnersMessage); err != nil {
		log.Errorf("action: request_winners | result: fail | attempt: %d | client_id: %v | error: %v",
			attempt, c.config.ID, err)
		return false
	}

	// Wait for server response
	var response protocol.ResponseMessage
	if err := protocol.RecvMessage(c.conn, &response); err != nil {
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
		log.Infof("action: request_winners | result: in_progress | attempt: %d | client_id: %v | error: %v",
			attempt, c.config.ID, response.Error)
		return false
	}
}
