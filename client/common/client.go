package common

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

// Client Entity
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

// validateBetRecord validates a CSV record and returns true if valid
func (c *Client) validateBetRecord(record []string, numero_aux *int) error {
	// Check basic format: must have exactly 5 fields
	if len(record) != EXPECTED_CSV_FIELDS {
		return fmt.Errorf("expected %d fields, got %d", EXPECTED_CSV_FIELDS, len(record))
	}

	// Check for empty or whitespace-only fields
	for i, field := range record[:CSV_INDEX_NUMERO] {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			return fmt.Errorf("field %d is empty or whitespace", i)
		}
		record[i] = trimmed // Update with trimmed value
	}

	// Validate field lengths
	if len(record[CSV_INDEX_NOMBRE]) > MAX_NOMBRE_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", FIELD_NOMBRE, MAX_NOMBRE_BYTES, len(record[CSV_INDEX_NOMBRE]))
	}
	if len(record[CSV_INDEX_APELLIDO]) > MAX_APELLIDO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", FIELD_APELLIDO, MAX_APELLIDO_BYTES, len(record[CSV_INDEX_APELLIDO]))
	}
	if len(record[CSV_INDEX_DOCUMENTO]) > MAX_DOCUMENTO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", FIELD_DOCUMENTO, MAX_DOCUMENTO_BYTES, len(record[CSV_INDEX_DOCUMENTO]))
	}
	if len(record[CSV_INDEX_NACIMIENTO]) > MAX_NACIMIENTO_BYTES {
		return fmt.Errorf("%s exceeds %d bytes: %d", FIELD_NACIMIENTO, MAX_NACIMIENTO_BYTES, len(record[CSV_INDEX_NACIMIENTO]))
	}

	// Validate numero field
	numero, err := strconv.Atoi(strings.TrimSpace(record[CSV_INDEX_NUMERO]))
	if err != nil {
		return fmt.Errorf("invalid %s: %s", FIELD_NUMERO, record[CSV_INDEX_NUMERO])
	}
	if numero < MIN_NUMERO || numero > MAX_NUMERO {
		return fmt.Errorf("%s out of range [%d-%d]: %d", FIELD_NUMERO, MIN_NUMERO, MAX_NUMERO, numero)
	}

	*numero_aux = numero
	return nil
}


// StartClientWithCSVStreaming processes CSV file in streaming mode without loading all records in memory
func (c *Client) StartClientWithCSVStreaming() {
	file, err := os.Open(c.config.CSVFile)
	if err != nil {
		log.Errorf("action: open_csv | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var currentBatch []BetMessage
	recordCount := 0
	validCount := 0
	var numero_apostado int

	for {
		if c.shutdownRequested {
			break
		}

		// Check if we've reached the maximum number of valid records
		if validCount >= MAX_CSV_RECORDS {
			log.Infof("action: read_csv_streaming | result: limit_reached | client_id: %v | max_records: %d | stopping early",
				c.config.ID, MAX_CSV_RECORDS)
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
		bet := BetMessage{
			Type:       MessageTypeBet,
			Nombre:     record[CSV_INDEX_NOMBRE],
			Apellido:   record[CSV_INDEX_APELLIDO],
			Documento:  record[CSV_INDEX_DOCUMENTO],
			Nacimiento: record[CSV_INDEX_NACIMIENTO],
			Numero:     numero_apostado,
		}

		// Check if adding this bet would exceed size or count limits
		testBatch := append(currentBatch, bet)
		batchSize := c.calculateMessageSize(testBatch)

		if len(currentBatch) >= c.config.BatchMaxAmount || batchSize > MAX_BATCH_SIZE_BYTES {
			// Send current batch first
			if len(currentBatch) > 0 {
				c.sendBatch(currentBatch)
			}
			// Start new batch with current bet
			currentBatch = []BetMessage{bet}
		} else {
			// Add to current batch
			currentBatch = testBatch
		}

		validCount++
	}

	// Send remaining bets in the last batch
	if len(currentBatch) > 0 && !c.shutdownRequested {
		c.sendBatch(currentBatch)
	}

	log.Infof("action: csv_streaming_finished | result: success | client_id: %v | total_records: %d | valid_records: %d | skipped: %d",
		c.config.ID, recordCount, validCount, recordCount-validCount)
}

// calculateMessageSize estimates the custom protocol size of a batch message
func (c *Client) calculateMessageSize(bets []BetMessage) int {
	batch := NewBatchMessage(c.config.Agency, bets, false)
	data, err := SerializeMessage(batch)
	if err != nil {
		return 0
	}
	return len(data) + LENGTH_PREFIX_BYTES // +4 for length prefix
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

// sendBatch sends a batch of bets to the server
func (c *Client) sendBatch(bets []BetMessage) {
	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close() // close connection after sending batch

	// Create batch message with agency number
	batchMessage := NewBatchMessage(c.config.Agency, bets, false)

	// Send batch message
	if err := SendMessage(c.conn, batchMessage); err != nil {
		log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}

	// Wait for server response
	var response ResponseMessage
	if err := RecvMessage(c.conn, &response); err != nil {
		if c.shutdownRequested {
			return
		}
		log.Errorf("action: receive_response | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}

	// Log result based on server response
	if response.Success {
		log.Infof("action: apuesta_enviada | result: success | batch_size: %v",
			len(bets))
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | batch_size: %v | error: %v",
			len(bets), response.Error)
	}
}
