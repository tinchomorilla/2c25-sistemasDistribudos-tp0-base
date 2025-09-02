package client

import (
	"net"
	"os"
	"os/signal"
	"strconv"
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
	writer            *Writer
	listener          *Listener
	fileManager       *FileManager
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

	// Initialize client components
	if err := client.initClient(); err != nil {
		log.Errorf("action: init_client | result: fail | client_id: %v | error: %v", client.config.ID, err)
		return nil
	}

	return client
}

// initClient initializes all client components
func (c *Client) initClient() error {
	// Create client socket
	if err := c.createClientSocket(); err != nil {
		return err
	}

	// Initialize Writer and Listener
	c.writer = NewWriter(c.conn, c.config.Agency, c.config.ID)
	c.listener = NewListener(c.conn, c.config.Agency, c.config.ID)

	// Initialize FileManager
	fileManager, err := NewFileManager(strconv.Itoa(c.config.Agency), c.config.CSVFile)
	if err != nil {
		return err
	}
	c.fileManager = fileManager

	return nil
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

// StartClientWithCSV processes CSV file in streaming mode without loading all records in memory
func (c *Client) StartClientWithCSV() {
	defer c.conn.Close()
	defer c.fileManager.Close()
	var currentBatch []protocol.BetMessage
	validCount := 0

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

		// Get next valid record
		bet, isEOF, err := c.fileManager.NextRecord()
		if err != nil {
			log.Errorf("action: read_csv_streaming | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
		if isEOF {
			break
		}

		// Check if adding this bet would exceed size or count limits
		testBatch := append(currentBatch, bet)
		batchSize := c.writer.CalculateMessageSize(testBatch)

		if len(testBatch) > c.config.BatchMaxAmount || batchSize > common.MAX_BATCH_SIZE_BYTES {
			c.writer.SendBatch(currentBatch, false)
			// Start new batch with current bet
			currentBatch = []protocol.BetMessage{bet}
		} else {
			// Add to current batch
			currentBatch = testBatch
		}

		validCount++
	}

	if len(currentBatch) > 0 && !c.shutdownRequested {
		c.writer.SendBatch(currentBatch, true) // EOF = true for the last batch
	}

	// After sending all bets with EOF, request winners
	if !c.shutdownRequested {
		c.listener.RequestWinners(&c.shutdownRequested)
	}

	log.Infof("action: csv_streaming_finished | result: success | client_id: %v | valid_records: %d",
		c.config.ID, validCount)
}
