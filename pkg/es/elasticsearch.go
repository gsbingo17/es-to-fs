package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/gsbingo17/es-to-mongodb/pkg/logger"
)

// ElasticsearchClient represents an Elasticsearch client
type ElasticsearchClient struct {
	client *elasticsearch.Client
	log    *logger.Logger
}

// NewElasticsearchClient creates a new Elasticsearch client
func NewElasticsearchClient(addresses []string, username, password, apiKey string, log *logger.Logger) (*ElasticsearchClient, error) {
	// Create Elasticsearch configuration
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	// Set authentication - prefer API key if provided, otherwise use username/password
	if apiKey != "" {
		cfg.APIKey = apiKey
		log.Info("Using API key authentication for Elasticsearch")
	} else if username != "" && password != "" {
		cfg.Username = username
		cfg.Password = password
		log.Info("Using username/password authentication for Elasticsearch")
	} else {
		log.Info("No authentication provided for Elasticsearch")
	}

	// Create Elasticsearch client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Ping the Elasticsearch server to verify connection
	res, err := client.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %s", res.String())
	}

	// Get Elasticsearch version
	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get Elasticsearch info: %w", err)
	}
	defer info.Body.Close()

	var infoResponse map[string]interface{}
	if err := json.NewDecoder(info.Body).Decode(&infoResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Elasticsearch info: %w", err)
	}

	version := infoResponse["version"].(map[string]interface{})["number"].(string)
	log.Infof("Connected to Elasticsearch %s", version)

	return &ElasticsearchClient{
		client: client,
		log:    log,
	}, nil
}

// GetIndices returns a list of indices matching the pattern
func (e *ElasticsearchClient) GetIndices(ctx context.Context, pattern string) ([]string, error) {
	// Create a request to get indices
	req := esapi.CatIndicesRequest{
		Index:  []string{pattern},
		Format: "json",
	}

	// Execute the request
	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get indices: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to get indices: %s", res.String())
	}

	// Parse the response
	var indices []map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("failed to decode indices response: %w", err)
	}

	// Extract index names
	var indexNames []string
	for _, index := range indices {
		indexNames = append(indexNames, index["index"].(string))
	}

	return indexNames, nil
}

// GetMappings returns the mappings for an index
func (e *ElasticsearchClient) GetMappings(ctx context.Context, index string) (map[string]interface{}, error) {
	// Create a request to get mappings
	req := esapi.IndicesGetMappingRequest{
		Index: []string{index},
	}

	// Execute the request
	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("failed to get mappings: %s", res.String())
	}

	// Parse the response
	var mappings map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&mappings); err != nil {
		return nil, fmt.Errorf("failed to decode mappings response: %w", err)
	}

	return mappings, nil
}

// CountDocuments returns the number of documents in an index
func (e *ElasticsearchClient) CountDocuments(ctx context.Context, index string) (int64, error) {
	// Create a request to count documents
	req := esapi.CountRequest{
		Index: []string{index},
	}

	// Execute the request
	res, err := req.Do(ctx, e.client)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("failed to count documents: %s", res.String())
	}

	// Parse the response
	var countResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&countResponse); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}

	count := int64(countResponse["count"].(float64))
	return count, nil
}

// ScrollDocuments scrolls through documents in an index
func (e *ElasticsearchClient) ScrollDocuments(ctx context.Context, index string, batchSize int, scrollTime string) (*ScrollIterator, error) {
	// Use the low-level API to perform a search with scroll
	// We'll use the client.Perform method to execute a custom request
	req := esapi.SearchRequest{
		Index: []string{index},
		Size:  &batchSize,
		Body:  strings.NewReader(`{"query": {"match_all": {}}}`),
	}

	// Convert scrollTime string to time.Duration
	scrollDuration, err := time.ParseDuration(scrollTime)
	if err != nil {
		// If parsing fails, use a default duration of 5 minutes
		scrollDuration = 5 * time.Minute
		e.log.Warnf("Failed to parse scroll time '%s', using default of 5m: %v", scrollTime, err)
	}
	req.Scroll = scrollDuration

	// Execute the request
	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	if res.IsError() {
		defer res.Body.Close()
		return nil, fmt.Errorf("failed to search documents: %s", res.String())
	}

	// Parse the response
	var searchResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		defer res.Body.Close()
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}
	res.Body.Close()

	// Get scroll ID
	scrollID, ok := searchResponse["_scroll_id"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to get scroll ID from response")
	}

	// Create scroll iterator
	iterator := &ScrollIterator{
		client:         e.client,
		scrollID:       scrollID,
		scrollTime:     scrollTime,
		scrollDuration: scrollDuration,
		currentBatch:   searchResponse,
		currentIndex:   0,
		batchSize:      batchSize,
		log:            e.log,
		ctx:            ctx,
		totalHits:      int64(searchResponse["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		processedHits:  0,
	}

	return iterator, nil
}

// ScrollIterator iterates through documents in a scroll
type ScrollIterator struct {
	client         *elasticsearch.Client
	scrollID       string
	scrollTime     string
	scrollDuration time.Duration
	currentBatch   map[string]interface{}
	currentIndex   int
	batchSize      int
	log            *logger.Logger
	ctx            context.Context
	totalHits      int64
	processedHits  int64
}

// Next returns the next document in the scroll
func (s *ScrollIterator) Next() (map[string]interface{}, error) {
	// Check if we need to fetch the next batch
	hits := s.currentBatch["hits"].(map[string]interface{})["hits"].([]interface{})
	if s.currentIndex >= len(hits) {
		// Fetch next batch
		if err := s.fetchNextBatch(); err != nil {
			return nil, err
		}

		// Check if we have more documents
		hits = s.currentBatch["hits"].(map[string]interface{})["hits"].([]interface{})
		if len(hits) == 0 {
			return nil, nil // No more documents
		}

		// Reset index
		s.currentIndex = 0
	}

	// Get current document
	hit := hits[s.currentIndex].(map[string]interface{})
	s.currentIndex++
	s.processedHits++

	// Log progress every 10000 documents
	if s.processedHits%10000 == 0 {
		s.log.Infof("Processed %d/%d documents (%.2f%%)", s.processedHits, s.totalHits, float64(s.processedHits)/float64(s.totalHits)*100)
	}

	// Extract document
	doc := make(map[string]interface{})
	doc["_id"] = hit["_id"]
	doc["_index"] = hit["_index"]
	doc["_source"] = hit["_source"]

	return doc, nil
}

// fetchNextBatch fetches the next batch of documents
func (s *ScrollIterator) fetchNextBatch() error {
	// Create a request to scroll
	req := esapi.ScrollRequest{
		ScrollID: s.scrollID,
		Scroll:   s.scrollDuration,
	}

	// Execute the request
	res, err := req.Do(s.ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to scroll: %w", err)
	}

	if res.IsError() {
		defer res.Body.Close()
		return fmt.Errorf("failed to scroll: %s", res.String())
	}

	// Parse the response
	var scrollResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&scrollResponse); err != nil {
		defer res.Body.Close()
		return fmt.Errorf("failed to decode scroll response: %w", err)
	}
	res.Body.Close()

	// Update scroll ID and current batch
	scrollID, ok := scrollResponse["_scroll_id"].(string)
	if !ok {
		return fmt.Errorf("failed to get scroll ID from response")
	}
	s.scrollID = scrollID
	s.currentBatch = scrollResponse

	return nil
}

// Close closes the scroll
func (s *ScrollIterator) Close() error {
	// Create a request to clear scroll
	req := esapi.ClearScrollRequest{
		ScrollID: []string{s.scrollID},
	}

	// Execute the request
	res, err := req.Do(s.ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to clear scroll: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to clear scroll: %s", res.String())
	}

	return nil
}
