package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gsbingo17/es-to-mongodb/pkg/common"
)

// Config represents the main configuration structure
type Config struct {
	// Source configuration
	Source SourceConfig `json:"source"`

	// Target MongoDB configuration
	Target TargetConfig `json:"target"`

	// Index to collection mapping
	IndexMappings []IndexMapping `json:"indexMappings,omitempty"`

	// Default mapping for indices not explicitly mapped
	DefaultMapping DefaultMapping `json:"defaultMapping"`

	// Document mapping configuration
	DocumentMapping common.MappingConfig `json:"documentMapping"`

	// Parameters for migration
	ReadBatchSize     int `json:"readBatchSize"`     // Number of documents to read in a batch during migration
	WriteBatchSize    int `json:"writeBatchSize"`    // Number of documents to write in a batch during migration
	ChannelBufferSize int `json:"channelBufferSize"` // Size of channel buffer for batches during migration
	MigrationWorkers  int `json:"migrationWorkers"`  // Number of worker goroutines for batch processing
	ConcurrentIndices int `json:"concurrentIndices"` // Number of indices to process concurrently
	SlicedScrollCount int `json:"slicedScrollCount"` // Number of slices for parallel scrolling within a single index

	// Retry configuration
	RetryConfig RetryConfig `json:"retryConfig"` // Configuration for retry mechanisms
}

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries           int  `json:"maxRetries"`           // Maximum number of retries
	BaseDelayMs          int  `json:"baseDelayMs"`          // Base delay in milliseconds
	MaxDelayMs           int  `json:"maxDelayMs"`           // Maximum delay in milliseconds
	EnableBatchSplitting bool `json:"enableBatchSplitting"` // Enable batch splitting for contention errors
	MinBatchSize         int  `json:"minBatchSize"`         // Minimum batch size for splitting
	ConvertInvalidIds    bool `json:"convertInvalidIds"`    // Convert invalid _id types to string
}

// SourceConfig represents the source Elasticsearch configuration
type SourceConfig struct {
	Addresses []string `json:"addresses"` // Elasticsearch addresses
	Username  string   `json:"username"`  // Elasticsearch username
	Password  string   `json:"password"`  // Elasticsearch password
	APIKey    string   `json:"apiKey"`    // Elasticsearch API key (alternative to username/password)
	Indices   []string `json:"indices"`   // Indices to migrate (can use patterns like "my-index-*")

	// HTTPS configuration
	TLS                    bool   `json:"tls"`                    // Enable TLS/HTTPS (default: false)
	CACertPath             string `json:"caCertPath"`             // Path to CA certificate file for server verification
	SkipVerify             bool   `json:"skipVerify"`             // Skip server certificate verification (not recommended for production)
	CertificateFingerprint string `json:"certificateFingerprint"` // Certificate fingerprint for verification
}

// TargetConfig represents the target MongoDB configuration
type TargetConfig struct {
	ConnectionString string `json:"connectionString"` // MongoDB connection string
	Database         string `json:"database"`         // MongoDB database name
}

// IndexMapping represents an index to collection mapping
type IndexMapping struct {
	SourceIndex      string `json:"sourceIndex"`      // Elasticsearch index name or pattern
	TargetCollection string `json:"targetCollection"` // MongoDB collection name
	IndexPatternType string `json:"indexPatternType"` // "exact", "wildcard", or "regex"
}

// DefaultMapping represents the default mapping for indices not explicitly mapped
type DefaultMapping struct {
	Type      string `json:"type"`      // "direct", "prefix", "suffix", "transform"
	Prefix    string `json:"prefix"`    // Prefix for collection names
	Suffix    string `json:"suffix"`    // Suffix for collection names
	Transform string `json:"transform"` // "lowercase", "uppercase", "remove-hyphens"
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// Set default config path if not provided
	if configPath == "" {
		configPath = "elasticsearch_to_mongodb_config.json"
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the config
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate the config
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	// Set default values for migration parameters
	if config.ReadBatchSize <= 0 {
		config.ReadBatchSize = 1000 // Default to 1000 documents
	}

	if config.WriteBatchSize <= 0 {
		config.WriteBatchSize = 100 // Default to 100 documents
	}

	if config.ChannelBufferSize <= 0 {
		config.ChannelBufferSize = 10 // Default to buffer for 10 batches
	}

	if config.MigrationWorkers <= 0 {
		config.MigrationWorkers = 5 // Default to 5 worker goroutines
	}

	if config.ConcurrentIndices <= 0 {
		config.ConcurrentIndices = 4 // Default to 4 concurrent indices
	}

	// Set default value for sliced scroll
	if config.SlicedScrollCount <= 0 {
		config.SlicedScrollCount = 4 // Default to 4 slices per index
	}

	// Set default values for retry configuration
	if config.RetryConfig.MaxRetries <= 0 {
		config.RetryConfig.MaxRetries = 5 // Default to 5 retries
	}

	if config.RetryConfig.BaseDelayMs <= 0 {
		config.RetryConfig.BaseDelayMs = 100 // Default to 100ms base delay
	}

	if config.RetryConfig.MaxDelayMs <= 0 {
		config.RetryConfig.MaxDelayMs = 5000 // Default to 5s max delay
	}

	if config.RetryConfig.MinBatchSize <= 0 {
		config.RetryConfig.MinBatchSize = 10 // Default to 10 docs per batch
	}

	// Set default value for ConvertInvalidIds
	// Default to true to automatically convert invalid _id types
	if !config.RetryConfig.ConvertInvalidIds {
		config.RetryConfig.ConvertInvalidIds = true
	}

	// Set default values for default mapping
	if config.DefaultMapping.Type == "" {
		config.DefaultMapping.Type = "direct" // Default to direct mapping
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate source config
	if len(config.Source.Addresses) == 0 {
		return fmt.Errorf("at least one Elasticsearch address is required")
	}

	if len(config.Source.Indices) == 0 {
		return fmt.Errorf("at least one Elasticsearch index is required")
	}

	// Validate TLS configuration
	if config.Source.TLS {
		// Check if certificate files exist if paths are provided
		if config.Source.CACertPath != "" {
			if _, err := os.Stat(config.Source.CACertPath); os.IsNotExist(err) {
				return fmt.Errorf("CA certificate file not found at path: %s", config.Source.CACertPath)
			}
		}
	}

	// Validate target config
	if config.Target.ConnectionString == "" {
		return fmt.Errorf("MongoDB connection string is required")
	}

	if config.Target.Database == "" {
		return fmt.Errorf("MongoDB database name is required")
	}

	// Validate index mappings if provided
	for i, mapping := range config.IndexMappings {
		if mapping.SourceIndex == "" {
			return fmt.Errorf("source index is required for index mapping at index %d", i)
		}
		if mapping.TargetCollection == "" {
			return fmt.Errorf("target collection is required for index mapping at index %d", i)
		}
		if mapping.IndexPatternType != "" && mapping.IndexPatternType != "exact" && mapping.IndexPatternType != "wildcard" && mapping.IndexPatternType != "regex" {
			return fmt.Errorf("invalid index pattern type for index mapping at index %d: must be 'exact', 'wildcard', or 'regex'", i)
		}
	}

	// Validate default mapping
	if config.DefaultMapping.Type != "direct" && config.DefaultMapping.Type != "prefix" && config.DefaultMapping.Type != "suffix" && config.DefaultMapping.Type != "transform" {
		return fmt.Errorf("invalid default mapping type: must be 'direct', 'prefix', 'suffix', or 'transform'")
	}

	if config.DefaultMapping.Type == "transform" && config.DefaultMapping.Transform != "lowercase" && config.DefaultMapping.Transform != "uppercase" && config.DefaultMapping.Transform != "remove-hyphens" {
		return fmt.Errorf("invalid default mapping transform: must be 'lowercase', 'uppercase', or 'remove-hyphens'")
	}

	return nil
}

// GetTargetCollection returns the target collection name for an index
func (c *Config) GetTargetCollection(indexName string) string {
	// Check explicit mappings first
	for _, mapping := range c.IndexMappings {
		if mapping.MatchesIndex(indexName) {
			return mapping.TargetCollection
		}
	}

	// Apply default mapping
	return c.ApplyDefaultMapping(indexName)
}

// MatchesIndex checks if an index matches a mapping
func (m *IndexMapping) MatchesIndex(indexName string) bool {
	switch m.IndexPatternType {
	case "wildcard":
		return wildcardMatch(m.SourceIndex, indexName)
	case "regex":
		matched, _ := regexMatch(m.SourceIndex, indexName)
		return matched
	default: // "exact"
		return m.SourceIndex == indexName
	}
}

// ApplyDefaultMapping applies the default mapping to an index name
func (c *Config) ApplyDefaultMapping(indexName string) string {
	switch c.DefaultMapping.Type {
	case "prefix":
		return c.DefaultMapping.Prefix + indexName
	case "suffix":
		return indexName + c.DefaultMapping.Suffix
	case "transform":
		return transformIndexName(indexName, c.DefaultMapping.Transform)
	default: // "direct"
		return indexName
	}
}

// wildcardMatch checks if a string matches a wildcard pattern
func wildcardMatch(pattern, str string) bool {
	// Simple wildcard matching implementation
	// This is a basic implementation and could be improved
	if pattern == "*" {
		return true
	}

	if pattern == str {
		return true
	}

	// Check if pattern ends with *
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}

	// Check if pattern starts with *
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
	}

	// Check if pattern has * in the middle
	parts := splitWildcard(pattern)
	if len(parts) > 1 {
		// Check if string starts with first part
		if !startsWith(str, parts[0]) {
			return false
		}

		// Check if string ends with last part
		if !endsWith(str, parts[len(parts)-1]) {
			return false
		}

		// Check if string contains all middle parts in order
		current := parts[0]
		for i := 1; i < len(parts)-1; i++ {
			idx := indexOf(str, parts[i], len(current))
			if idx == -1 {
				return false
			}
			current = str[:idx+len(parts[i])]
		}

		return true
	}

	return false
}

// regexMatch checks if a string matches a regex pattern
func regexMatch(pattern, str string) (bool, error) {
	// In a real implementation, this would use the regexp package
	// For simplicity, we'll just return false for now
	return false, fmt.Errorf("regex matching not implemented")
}

// transformIndexName transforms an index name based on the transform type
func transformIndexName(indexName, transform string) string {
	switch transform {
	case "lowercase":
		return toLower(indexName)
	case "uppercase":
		return toUpper(indexName)
	case "remove-hyphens":
		return removeHyphens(indexName)
	default:
		return indexName
	}
}

// Helper functions for string manipulation
func toLower(s string) string {
	// Simple lowercase implementation
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func toUpper(s string) string {
	// Simple uppercase implementation
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			result[i] = s[i] - 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func removeHyphens(s string) string {
	// Simple hyphen removal implementation
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '-' {
			result = append(result, s[i])
		}
	}
	return string(result)
}

func splitWildcard(pattern string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			if i > start {
				parts = append(parts, pattern[start:i])
			}
			start = i + 1
		}
	}
	if start < len(pattern) {
		parts = append(parts, pattern[start:])
	}
	return parts
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func indexOf(s, substr string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
