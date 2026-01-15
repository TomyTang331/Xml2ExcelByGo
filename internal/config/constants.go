package config

// Performance configuration constants (all values use powers of 2)
const (
	// XML parser buffer size: 64KB
	DefaultXMLBufferSize = 64 * 1024

	// Excel writer batch size: 2048 rows
	DefaultBatchSize = 2048

	// Header sample size: 128 rows
	HeaderSampleSize = 128

	// Excel formatting
	DefaultColWidth = 15
	HeaderStyleBg   = "#E0E0E0"
)
