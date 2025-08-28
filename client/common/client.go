package common

import (
	"encoding/csv"
	"encoding/json"
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

// calculateMessageSize estimates the JSON size of a batch message
func (c *Client) calculateMessageSize(bets []BetMessage) int {
	batch := NewBatchMessage(c.config.Agency, bets, false) // Use false for size calculation
	data, err := json.Marshal(batch)
	if err != nil {
		return 0
	}
	return len(data)
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
	log.Infof("action: start_csv_client | csv_file: %s | client_id: %v", c.config.CSVFile, c.config.ID)

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
		if eof {
			log.Infof("action: apuesta_enviada | result: success | batch_size: %v | eof: true",
				len(bets))
		} else {
			log.Infof("action: apuesta_enviada | result: success | batch_size: %v",
				len(bets))
		}
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | batch_size: %v | error: %v",
			len(bets), response.Error)
	}
}
