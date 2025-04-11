package es

import (
	"fmt"
	"time"

	"github.com/gsbingo17/es-to-mongodb/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

// DocumentMapper maps Elasticsearch documents to MongoDB documents
type DocumentMapper struct {
	config        MappingConfig
	fieldMappings map[string]*FieldMapping
	log           *logger.Logger
}

// NewDocumentMapper creates a new document mapper
func NewDocumentMapper(config MappingConfig, log *logger.Logger) *DocumentMapper {
	// Create field mappings map for quick lookup
	fieldMappings := make(map[string]*FieldMapping)
	for i := range config.FieldMappings {
		mapping := &config.FieldMappings[i]
		fieldMappings[mapping.SourceField] = mapping
	}

	// Set default values
	if config.IDField == "" {
		config.IDField = "_id"
	}
	if config.IDType == "" {
		config.IDType = "string"
	}
	if config.MetadataPrefix == "" {
		config.MetadataPrefix = "es_"
	}
	if config.DateFormat == "" {
		config.DateFormat = "strict_date_optional_time"
	}

	return &DocumentMapper{
		config:        config,
		fieldMappings: fieldMappings,
		log:           log,
	}
}

// MapDocument maps an Elasticsearch document to a MongoDB document
func (m *DocumentMapper) MapDocument(esDoc map[string]interface{}) (bson.D, error) {
	mongoDoc := bson.D{}

	// Handle ID field
	if id, ok := esDoc["_id"]; ok {
		convertedID, err := m.convertID(id.(string))
		if err != nil {
			return nil, err
		}
		mongoDoc = append(mongoDoc, bson.E{Key: "_id", Value: convertedID})
	}

	// Handle metadata fields
	if m.config.IncludeMetadata {
		for key, value := range esDoc {
			if key == "_source" {
				continue
			}
			if key == "_id" {
				continue
			}
			mongoDoc = append(mongoDoc, bson.E{Key: m.config.MetadataPrefix + key[1:], Value: value})
		}
	}

	// Handle source fields
	source, ok := esDoc["_source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("document has no _source field")
	}

	// Process each field in the source
	for key, value := range source {
		// Skip excluded fields
		if m.shouldSkipField(key) {
			continue
		}

		// Apply field mapping if exists
		if mapping, ok := m.fieldMappings[key]; ok {
			mappedValue, err := m.applyMapping(mapping, value)
			if err != nil {
				return nil, err
			}
			mongoDoc = append(mongoDoc, bson.E{Key: mapping.TargetField, Value: mappedValue})
		} else {
			// Apply dynamic mapping
			convertedValue, err := m.dynamicMap(key, value)
			if err != nil {
				return nil, err
			}
			mongoDoc = append(mongoDoc, bson.E{Key: key, Value: convertedValue})
		}
	}

	return mongoDoc, nil
}

// shouldSkipField checks if a field should be skipped
func (m *DocumentMapper) shouldSkipField(field string) bool {
	for _, excludeField := range m.config.ExcludeFields {
		if field == excludeField {
			return true
		}
	}
	return false
}

// convertID converts an Elasticsearch ID to a MongoDB ID
func (m *DocumentMapper) convertID(esID string) (interface{}, error) {
	switch m.config.IDType {
	case "objectid":
		// Try to convert to ObjectID if it's in the right format
		if len(esID) == 24 {
			objectID, err := primitive.ObjectIDFromHex(esID)
			if err == nil {
				return objectID, nil
			}
		}
		// Otherwise create a new ObjectID
		return primitive.NewObjectID(), nil
	case "string":
		return esID, nil
	case "int":
		// Try to parse as integer
		var intID int
		_, err := fmt.Sscanf(esID, "%d", &intID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ID to int: %w", err)
		}
		return intID, nil
	default:
		return esID, nil
	}
}

// applyMapping applies a field mapping to a value
func (m *DocumentMapper) applyMapping(mapping *FieldMapping, value interface{}) (interface{}, error) {
	switch mapping.Type {
	case "date":
		return m.convertDate(value)
	case "geo_point":
		return m.convertGeoPoint(value)
	case "nested":
		return m.convertNested(value)
	default:
		return value, nil
	}
}

// dynamicMap dynamically maps a field value based on its type
func (m *DocumentMapper) dynamicMap(field string, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		// Handle nested object
		return m.convertObject(v)
	case []interface{}:
		// Handle array
		return m.convertArray(field, v)
	case string:
		// Try to detect if it's a date
		if date, err := m.tryParseDate(v); err == nil {
			return date, nil
		}
		// Otherwise keep as string
		return v, nil
	default:
		// Keep as is for other types
		return v, nil
	}
}

// convertObject converts a nested object
func (m *DocumentMapper) convertObject(obj map[string]interface{}) (interface{}, error) {
	result := bson.D{}
	for key, value := range obj {
		// Skip excluded fields
		if m.shouldSkipField(key) {
			continue
		}

		// Apply field mapping if exists
		if mapping, ok := m.fieldMappings[key]; ok {
			mappedValue, err := m.applyMapping(mapping, value)
			if err != nil {
				return nil, err
			}
			result = append(result, bson.E{Key: mapping.TargetField, Value: mappedValue})
		} else {
			// Apply dynamic mapping
			convertedValue, err := m.dynamicMap(key, value)
			if err != nil {
				return nil, err
			}
			result = append(result, bson.E{Key: key, Value: convertedValue})
		}
	}
	return result, nil
}

// convertArray converts an array
func (m *DocumentMapper) convertArray(field string, arr []interface{}) (interface{}, error) {
	result := make([]interface{}, len(arr))
	for i, item := range arr {
		var err error
		result[i], err = m.dynamicMap(field, item)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// convertDate converts a date string to a MongoDB date
func (m *DocumentMapper) convertDate(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return m.tryParseDate(v)
	default:
		return value, nil
	}
}

// tryParseDate tries to parse a string as a date
func (m *DocumentMapper) tryParseDate(dateStr string) (interface{}, error) {
	// Try common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Not a date
	return dateStr, nil
}

// convertGeoPoint converts an Elasticsearch geo_point to a MongoDB GeoJSON point
func (m *DocumentMapper) convertGeoPoint(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		// Check if it has lat/lon fields
		lat, hasLat := v["lat"]
		lon, hasLon := v["lon"]
		if hasLat && hasLon {
			// Convert to GeoJSON
			return bson.D{
				{Key: "type", Value: "Point"},
				{Key: "coordinates", Value: []interface{}{lon, lat}},
			}, nil
		}
		return v, nil
	case string:
		// Parse "lat,lon" format
		var lat, lon float64
		_, err := fmt.Sscanf(v, "%f,%f", &lat, &lon)
		if err != nil {
			return v, nil
		}
		return bson.D{
			{Key: "type", Value: "Point"},
			{Key: "coordinates", Value: []interface{}{lon, lat}},
		}, nil
	case []interface{}:
		// Parse [lon, lat] format
		if len(v) == 2 {
			return bson.D{
				{Key: "type", Value: "Point"},
				{Key: "coordinates", Value: []interface{}{v[0], v[1]}},
			}, nil
		}
		return v, nil
	default:
		return v, nil
	}
}

// convertNested converts a nested array of objects
func (m *DocumentMapper) convertNested(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			switch obj := item.(type) {
			case map[string]interface{}:
				converted, err := m.convertObject(obj)
				if err != nil {
					return nil, err
				}
				result[i] = converted
			default:
				result[i] = item
			}
		}
		return result, nil
	default:
		return value, nil
	}
}
