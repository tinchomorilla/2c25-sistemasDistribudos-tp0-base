package common

// Field validation limits (in bytes)
const (
	// String field byte limits
	MAX_NOMBRE_BYTES     = 50
	MAX_APELLIDO_BYTES   = 50
	MAX_DOCUMENTO_BYTES  = 20
	MAX_NACIMIENTO_BYTES = 10

	// Numeric field limits
	MIN_NUMERO = 0
	MAX_NUMERO = 999999

	// CSV validation
	EXPECTED_CSV_FIELDS = 5

	// Batch processing limits
	MAX_BATCH_SIZE_BYTES     = 8 * 1024 // 8KB limit
	DEFAULT_BATCH_MAX_AMOUNT = 10       // Default batch size when not configured
	MAX_SEND_RETRIES         = 3        // Maximum retries for sending messages

	// CSV processing limits
	MAX_CSV_RECORDS = 100000 // Maximum number of valid records to process from CSV

	// Protocol constants
	LENGTH_PREFIX_BYTES = 4 // Length prefix size in bytes
)

// Field names for better error messages
const (
	FIELD_NOMBRE     = "nombre"
	FIELD_APELLIDO   = "apellido"
	FIELD_DOCUMENTO  = "documento"
	FIELD_NACIMIENTO = "nacimiento"
	FIELD_NUMERO     = "numero"
)

// CSV field indices
const (
	CSV_INDEX_NOMBRE     = 0
	CSV_INDEX_APELLIDO   = 1
	CSV_INDEX_DOCUMENTO  = 2
	CSV_INDEX_NACIMIENTO = 3
	CSV_INDEX_NUMERO     = 4
)
