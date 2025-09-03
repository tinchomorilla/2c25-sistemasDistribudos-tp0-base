package client

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/tinchomorilla/2c25-sistemasDistribudos-tp0-base/client/common"
	"github.com/tinchomorilla/2c25-sistemasDistribudos-tp0-base/client/protocol"
)

type FileManager struct {
	agency string
	file   *os.File
	reader *csv.Reader
}

func NewFileManager(agency string, csvFilePath string) (*FileManager, error) {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s | %v", csvFilePath, err)
	}

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 0

	return &FileManager{
		agency: agency,
		file:   file,
		reader: reader,
	}, nil
}

func (f *FileManager) NextRecord() (protocol.BetMessage, bool, error) {
	for {
		line, err := f.reader.Read()
		if err != nil {
			if err == io.EOF {
				return protocol.BetMessage{}, true, nil // EOF reached
			}
			return protocol.BetMessage{}, false, fmt.Errorf("failed to read from file | %v", err)
		}

		// Validate and create bet message
		bet, err := f.createValidBetMessage(line)
		if err != nil {
			// Skip invalid records, continue to next
			continue
		}

		return bet, false, nil
	}
}

func (f *FileManager) Close() {
	err := f.file.Close()
	if err != nil {
		_= fmt.Errorf("failed to close file | %v", err)
	}
}

// createValidBetMessage validates a CSV record and creates a BetMessage
func (f *FileManager) createValidBetMessage(record []string) (protocol.BetMessage, error) {
	// Validate record using validation logic
	var numeroApostado int
	if err := f.validateBetRecord(record, &numeroApostado); err != nil {
		return protocol.BetMessage{}, err
	}

	// Create bet message 
	bet := protocol.BetMessage{
		Type:       protocol.MessageTypeBet,
		Nombre:     record[common.CSV_INDEX_NOMBRE],
		Apellido:   record[common.CSV_INDEX_APELLIDO],
		Documento:  record[common.CSV_INDEX_DOCUMENTO],
		Nacimiento: record[common.CSV_INDEX_NACIMIENTO],
		Numero:     numeroApostado,
	}

	return bet, nil
}

// validateBetRecord validates a CSV record (adapted from client.go)
func (f *FileManager) validateBetRecord(record []string, numeroAux *int) error {
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

	// Validate numero_apostado field
	numero, err := strconv.Atoi(strings.TrimSpace(record[common.CSV_INDEX_NUMERO]))
	if err != nil {
		return fmt.Errorf("invalid %s: %s", common.FIELD_NUMERO, record[common.CSV_INDEX_NUMERO])
	}
	if numero < common.MIN_NUMERO || numero > common.MAX_NUMERO {
		return fmt.Errorf("%s out of range [%d-%d]: %d", common.FIELD_NUMERO, common.MIN_NUMERO, common.MAX_NUMERO, numero)
	}

	*numeroAux = numero
	return nil
}
