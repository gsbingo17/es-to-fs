package common

// FieldMapping represents a field mapping configuration
type FieldMapping struct {
	SourceField string `json:"sourceField"`
	TargetField string `json:"targetField"`
	Type        string `json:"type,omitempty"`
}

// MappingConfig represents a mapping configuration
type MappingConfig struct {
	IDField         string         `json:"idField"`
	IDType          string         `json:"idType"`
	IncludeMetadata bool           `json:"includeMetadata"`
	MetadataPrefix  string         `json:"metadataPrefix"`
	DateFormat      string         `json:"dateFormat"`
	FieldMappings   []FieldMapping `json:"fieldMappings"`
	ExcludeFields   []string       `json:"excludeFields"`
}
