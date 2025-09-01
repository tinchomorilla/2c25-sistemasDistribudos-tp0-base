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

// validateBetRecord validates a CSV record and returns true if valid
func (c *Client) validateBetRecord(record []string, numero_aux *int) error {
	// Field length limits (in bytes)
	const (
		MAX_NOMBRE_BYTES     = 50
		MAX_APELLIDO_BYTES   = 50
		MAX_DOCUMENTO_BYTES  = 20
		MAX_NACIMIENTO_BYTES = 10
		MIN_NUMERO           = 0
		MAX_NUMERO           = 999999
	)

	// Check basic format: must have exactly 5 fields
	if len(record) != 5 {
		return fmt.Errorf("expected 5 fields, got %d", len(record))
	}

	// Check for empty or whitespace-only fields
	for i, field := range record[:4] {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			return fmt.Errorf("field %d is empty or whitespace", i)
		}
		record[i] = trimmed // Update with trimmed value
	}

	// Validate field lengths
	if len(record[0]) > MAX_NOMBRE_BYTES {
		return fmt.Errorf("nombre exceeds %d bytes: %d", MAX_NOMBRE_BYTES, len(record[0]))
	}
	if len(record[1]) > MAX_APELLIDO_BYTES {
		return fmt.Errorf("apellido exceeds %d bytes: %d", MAX_APELLIDO_BYTES, len(record[1]))
	}
	if len(record[2]) > MAX_DOCUMENTO_BYTES {
		return fmt.Errorf("documento exceeds %d bytes: %d", MAX_DOCUMENTO_BYTES, len(record[2]))
	}
	if len(record[3]) > MAX_NACIMIENTO_BYTES {
		return fmt.Errorf("nacimiento exceeds %d bytes: %d", MAX_NACIMIENTO_BYTES, len(record[3]))
	}

	// Validate numero field
	numero, err := strconv.Atoi(strings.TrimSpace(record[4]))
	if err != nil {
		return fmt.Errorf("invalid numero: %s", record[4])
	}
	if numero < MIN_NUMERO || numero > MAX_NUMERO {
		return fmt.Errorf("numero out of range [%d-%d]: %d", MIN_NUMERO, MAX_NUMERO, numero)
	}

	*numero_aux = numero
	return nil
}

// readBetsFromCSV reads bet records from a CSV file
func (c *Client) readBetsFromCSV() ([]BetMessage, error) {
	file, err := os.Open(c.config.CSVFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var bets []BetMessage
	recordCount := 0
	validCount := 0
	var numero_apostado int
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
			Nombre:     record[0],
			Apellido:   record[1],
			Documento:  record[2],
			Nacimiento: record[3],
			Numero:     numero_apostado,
		}
		bets = append(bets, bet)
		validCount++
	}

	log.Infof("action: read_csv | result: success | client_id: %v | total_records: %d | valid_records: %d | skipped: %d",
		c.config.ID, recordCount, validCount, recordCount-validCount)

	return bets, nil
}

// calculateMessageSize estimates the custom protocol size of a batch message
func (c *Client) calculateMessageSize(bets []BetMessage) int {
	batch := NewBatchMessage(c.config.Agency, bets, false)
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

// StartClientLoopWithCSV reads bets from CSV and sends them in batches (Exercise 6)
func (c *Client) StartClientWithCSV() {
	// Read bets from CSV file
	betsFromCSV, err := c.readBetsFromCSV()
	if err != nil {
		log.Errorf("action: read_csv | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	// Convert CSV records to BetMessages and send in batches
	var currentBatch []BetMessage
	const MAX_BATCH_SIZE_BYTES = 8 * 1024 // 8kB limit

	for _, betMessage := range betsFromCSV {
		if c.shutdownRequested {
			break
		}

		// Check if adding this bet would exceed size or count limits
		testBatch := append(currentBatch, betMessage)
		batchSize := c.calculateMessageSize(testBatch)

		if len(currentBatch) >= c.config.BatchMaxAmount || batchSize > MAX_BATCH_SIZE_BYTES {
			// Send current batch first
			if len(currentBatch) > 0 {
				c.sendBatch(currentBatch)
			}
			// Start new batch with current bet
			currentBatch = []BetMessage{betMessage}
		} else {
			// Add to current batch
			currentBatch = testBatch
		}
	}

	// Send remaining bets in the last batch
	if len(currentBatch) > 0 && !c.shutdownRequested {
		c.sendBatch(currentBatch)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

// sendBatch sends a batch of bets to the server
func (c *Client) sendBatch(bets []BetMessage) {
	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close()

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
