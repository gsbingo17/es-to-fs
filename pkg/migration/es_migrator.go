package migration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gsbingo17/es-to-mongodb/pkg/common"
	"github.com/gsbingo17/es-to-mongodb/pkg/config"
	"github.com/gsbingo17/es-to-mongodb/pkg/db"
	"github.com/gsbingo17/es-to-mongodb/pkg/es"
	"github.com/gsbingo17/es-to-mongodb/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ESMigrator handles the migration from Elasticsearch to MongoDB
type ESMigrator struct {
	config *config.Config
	log    *logger.Logger
}

// NewESMigrator creates a new ESMigrator
func NewESMigrator(config *config.Config, log *logger.Logger) *ESMigrator {
	return &ESMigrator{
		config: config,
		log:    log,
	}
}

// Start starts the migration process
func (m *ESMigrator) Start(ctx context.Context) error {
	m.log.Info("Starting Elasticsearch to MongoDB migration process")

	// Connect to Elasticsearch
	m.log.Infof("Connecting to Elasticsearch at %v", m.config.Source.Addresses)

	// Create TLS configuration with timeout settings
	var tlsConfig *es.TLSConfig
	if m.config.Source.TLS {
		tlsConfig = &es.TLSConfig{
			Enabled:                true,
			CACertPath:             m.config.Source.CACertPath,
			SkipVerify:             m.config.Source.SkipVerify,
			CertificateFingerprint: m.config.Source.CertificateFingerprint,
			ConnectionTimeout:      m.config.Source.ConnectionTimeout,
			ResponseTimeout:        m.config.Source.ResponseTimeout,
		}
		m.log.Info("TLS is enabled for Elasticsearch connection")
	} else {
		// Even without TLS, we still need to pass timeout settings
		tlsConfig = &es.TLSConfig{
			Enabled:           false,
			ConnectionTimeout: m.config.Source.ConnectionTimeout,
			ResponseTimeout:   m.config.Source.ResponseTimeout,
		}
		m.log.Infof("Using non-TLS connection with timeouts: connection=%ds, response=%ds",
			m.config.Source.ConnectionTimeout, m.config.Source.ResponseTimeout)
	}

	esClient, err := es.NewElasticsearchClient(
		m.config.Source.Addresses,
		m.config.Source.Username,
		m.config.Source.Password,
		m.config.Source.APIKey,
		tlsConfig,
		m.log,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	// Add defer for Elasticsearch client cleanup
	defer func() {
		if err := esClient.Close(); err != nil {
			m.log.Errorf("Error closing Elasticsearch client: %v", err)
		}
	}()

	// Connect to MongoDB
	m.log.Infof("Connecting to MongoDB at %s", m.config.Target.ConnectionString)
	mongoClient, err := db.NewMongoDB(m.config.Target.ConnectionString, m.config.Target.Database, m.log)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Create document mapper
	mapper := common.NewDocumentMapper(m.config.DocumentMapping, m.log)

	// Process index patterns concurrently
	var patternWg sync.WaitGroup
	// Use a semaphore to limit concurrent pattern processing
	// Default to 2 concurrent patterns, but this could be configurable
	concurrentPatterns := 2
	patternSemaphore := make(chan struct{}, concurrentPatterns)

	// Channel to collect errors from goroutines
	errorChan := make(chan error, len(m.config.Source.Indices))

	m.log.Infof("Processing %d index patterns concurrently with limit of %d", len(m.config.Source.Indices), concurrentPatterns)

	// Process each index pattern in parallel
	for _, indexPattern := range m.config.Source.Indices {
		patternWg.Add(1)
		// Acquire semaphore
		patternSemaphore <- struct{}{}

		// Start processing pattern in a goroutine
		go func(pattern string) {
			defer patternWg.Done()
			defer func() { <-patternSemaphore }() // Release semaphore when done

			// Create a context with timeout for getting indices
			// 30 second timeout for getting indices
			indexCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Get matching indices
			m.log.Infof("Finding indices matching pattern: %s", pattern)
			indices, err := esClient.GetIndices(indexCtx, pattern)
			if err != nil {
				if err == context.DeadlineExceeded {
					errorChan <- fmt.Errorf("timeout getting indices for pattern %s: %w", pattern, err)
				} else if err == context.Canceled {
					errorChan <- fmt.Errorf("getting indices for pattern %s was canceled: %w", pattern, err)
				} else {
					errorChan <- fmt.Errorf("failed to get indices for pattern %s: %w", pattern, err)
				}
				return
			}

			m.log.Infof("Found %d indices matching pattern %s: %v", len(indices), pattern, indices)

			// Process indices concurrently with a semaphore
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, m.config.ConcurrentIndices)

			// Channel for index-level errors
			indexErrorChan := make(chan error, len(indices))

			for _, index := range indices {
				// Check if the parent context is canceled
				select {
				case <-ctx.Done():
					indexErrorChan <- ctx.Err()
					return
				default:
					// Continue processing
				}

				wg.Add(1)
				// Acquire semaphore
				semaphore <- struct{}{}

				// Start migration in a goroutine
				go func(indexName string) {
					defer wg.Done()
					defer func() { <-semaphore }() // Release semaphore when done

					// Get target collection name
					targetCollection := m.config.GetTargetCollection(indexName)
					m.log.Infof("Migrating index %s to collection %s", indexName, targetCollection)

					// Migrate the index
					if err := m.migrateIndex(ctx, esClient, mongoClient, indexName, targetCollection, mapper); err != nil {
						if err == context.Canceled {
							m.log.Infof("Migration of index %s interrupted due to user interrupt (Ctrl+C)", indexName)
							indexErrorChan <- err
						} else {
							m.log.Errorf("Error migrating index %s: %v", indexName, err)
							// Continue with other indices even if one fails
						}
					}
				}(index)
			}

			// Wait for all migrations to complete
			wg.Wait()

			// Check for index-level errors
			select {
			case err := <-indexErrorChan:
				errorChan <- fmt.Errorf("error processing pattern %s: %w", pattern, err)
			default:
				// No errors
			}

		}(indexPattern)
	}

	// Wait for all pattern processing to complete
	patternWg.Wait()
	close(errorChan)

	// Check for errors
	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}

	// Close MongoDB connection
	if err := mongoClient.Close(ctx); err != nil {
		m.log.Errorf("Error closing MongoDB connection: %v", err)
		errs = append(errs, fmt.Errorf("error closing MongoDB connection: %w", err))
	}

	// Return the first error if any
	if len(errs) > 0 {
		m.log.Errorf("Migration completed with %d errors", len(errs))
		return errs[0]
	}

	m.log.Info("Migration completed successfully")
	return nil
}

// migrateIndex migrates a single Elasticsearch index to a MongoDB collection
func (m *ESMigrator) migrateIndex(ctx context.Context, esClient *es.ElasticsearchClient, mongoClient *db.MongoDB, indexName, collectionName string, mapper *common.DocumentMapper) error {
	// Use sync.Once to ensure batchChan is closed exactly once
	var closeOnce sync.Once
	// Get MongoDB collection
	collection := mongoClient.GetCollection(collectionName)

	// Count documents in the index
	totalCount, err := esClient.CountDocuments(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to count documents in index %s: %w", indexName, err)
	}

	m.log.Infof("Found %d documents to migrate from index %s", totalCount, indexName)

	// If no documents, we're done
	if totalCount == 0 {
		m.log.Infof("No documents to migrate for index %s", indexName)
		return nil
	}

	// Create retry manager for batch processing
	retryManager := NewRetryManager(
		m.config.RetryConfig.MaxRetries,
		time.Duration(m.config.RetryConfig.BaseDelayMs)*time.Millisecond,
		time.Duration(m.config.RetryConfig.MaxDelayMs)*time.Millisecond,
		m.config.RetryConfig.EnableBatchSplitting,
		m.config.RetryConfig.MinBatchSize,
		m.config.RetryConfig.ConvertInvalidIds,
		m.log,
	)

	// Set up parallel batch processing
	var wg sync.WaitGroup
	batchChan := make(chan []interface{}, m.config.ChannelBufferSize) // Buffer for batches
	errorChan := make(chan error, 1)                                  // Channel for errors
	doneChan := make(chan struct{})                                   // Channel to signal completion

	// Track progress
	var migratedCount int64
	var lastLoggedPercentage int = -1 // Start at -1 to ensure 0% is logged
	var mu sync.Mutex                 // Mutex for thread-safe updates to migratedCount and lastLoggedPercentage

	// Start worker pool for parallel batch processing
	workerCount := m.config.MigrationWorkers
	m.log.Infof("Starting %d workers for parallel document batch processing", workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for batch := range batchChan {
				// Use RetryManager to handle retries with batch splitting
				err := retryManager.RetryWithSplit(ctx, batch, indexName, func(b []interface{}) error {
					return processESBatch(ctx, collection, b, false) // Use insert by default
				})
				if err != nil {
					select {
					case errorChan <- fmt.Errorf("worker %d failed to process batch: %w", workerID, err):
					default:
						// Error channel already has an error
					}
					return
				}

				// Update progress
				mu.Lock()
				migratedCount += int64(len(batch))
				currentCount := migratedCount // Copy for logging outside the lock

				// Calculate current percentage (0-10 for 0%-100%)
				currentPercentage := int(float64(currentCount) / float64(totalCount) * 10)

				// Only log when crossing a 10% threshold at the index level
				// and update lastLoggedPercentage atomically to prevent multiple logs
				shouldLog := false
				if currentPercentage > lastLoggedPercentage {
					lastLoggedPercentage = currentPercentage
					shouldLog = true
				}
				mu.Unlock()

				// Log outside the mutex lock to reduce lock contention
				if shouldLog {
					m.log.Infof("Index %s progress: %d/%d documents (%.0f%%)",
						indexName, currentCount, totalCount, float64(currentPercentage)*10)
				}
			}
		}(i)
	}

	// Start a goroutine to close channels when all batches are processed
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	// Use sliced scroll for parallel reading from a single index
	scrollTime := "5m" // 5 minutes scroll time
	numSlices := m.config.SlicedScrollCount
	m.log.Infof("Using sliced scroll with %d slices for parallel reading from index %s", numSlices, indexName)

	// Create a wait group for scroll iterators
	var scrollWg sync.WaitGroup
	scrollErrorChan := make(chan error, numSlices) // Channel for scroll errors

	// Start a goroutine for each slice
	for slice := 0; slice < numSlices; slice++ {
		scrollWg.Add(1)
		go func(sliceID int) {
			defer scrollWg.Done()

			// Create a sliced scroll iterator
			iterator, err := esClient.ScrollDocumentsSliced(ctx, indexName, m.config.ReadBatchSize, scrollTime, sliceID, numSlices)
			if err != nil {
				scrollErrorChan <- fmt.Errorf("failed to create scroll iterator for slice %d/%d: %w", sliceID, numSlices, err)
				return
			}
			defer iterator.Close()

			// Process documents from this slice
			var batch []interface{}
			var batchCount int

			for {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					scrollErrorChan <- context.Canceled
					return
				default:
					// Continue processing
				}

				// Get next document
				doc, err := iterator.Next()
				if err != nil {
					scrollErrorChan <- fmt.Errorf("error reading document from slice %d/%d: %w", sliceID, numSlices, err)
					return
				}

				// If no more documents in this slice, we're done
				if doc == nil {
					break
				}

				// Map document
				mappedDoc, err := mapper.MapDocument(doc)
				if err != nil {
					scrollErrorChan <- fmt.Errorf("error mapping document from slice %d/%d: %w", sliceID, numSlices, err)
					return
				}

				// Add to batch
				batch = append(batch, mappedDoc)
				batchCount++

				// Send batch if it reaches the write batch size
				if batchCount >= m.config.WriteBatchSize {
					select {
					case batchChan <- batch:
						// Batch sent to worker
					case <-ctx.Done():
						// Context cancelled
						scrollErrorChan <- context.Canceled
						return
					}

					// Reset batch
					batch = nil
					batchCount = 0

					// Add a small delay between batches to reduce contention
					time.Sleep(5 * time.Millisecond)
				}
			}

			// Process any remaining documents in this slice
			if len(batch) > 0 {
				select {
				case batchChan <- batch:
					// Final batch sent to worker
				case <-ctx.Done():
					// Context cancelled
					scrollErrorChan <- context.Canceled
					return
				}
			}

			m.log.Infof("Slice %d/%d completed reading", sliceID, numSlices)
		}(slice)
	}

	// Wait for all scroll iterators to complete or for an error
	go func() {
		scrollWg.Wait()
		closeOnce.Do(func() {
			close(batchChan) // Close batch channel when all readers are done
			m.log.Info("All slices completed reading")
		})
	}()

	// Monitor for errors from scroll iterators
	var scrollError error
	go func() {
		select {
		case err := <-scrollErrorChan:
			scrollError = err
			closeOnce.Do(func() {
				close(batchChan) // Close batch channel on error
				m.log.Info("Closing batch channel due to scroll error")
			})
		case <-doneChan:
			// All processing completed successfully
		case <-ctx.Done():
			// Context cancelled
			scrollError = context.Canceled
			closeOnce.Do(func() {
				close(batchChan) // Close batch channel on context cancellation
				m.log.Info("Closing batch channel due to context cancellation")
			})
		}
	}()

	// Wait for all workers to finish or for an error
	select {
	case <-doneChan:
		// All workers finished successfully
		if scrollError != nil {
			return scrollError
		}
	case err := <-errorChan:
		// Error from a worker
		return err
	case <-ctx.Done():
		// Context cancelled
		m.log.Info("Migration interrupted due to context cancellation")
		return context.Canceled // Return context.Canceled for consistent error handling
	}

	m.log.Infof("Migration for index %s completed successfully! Total documents: %d", indexName, migratedCount)
	return nil
}

// processESBatch processes a batch of documents from Elasticsearch
func processESBatch(ctx context.Context, collection *mongo.Collection, batch []interface{}, useUpsert bool) error {
	if len(batch) == 0 {
		return nil
	}

	// If upsert mode is enabled, use upsert operations directly
	if useUpsert {
		var models []mongo.WriteModel
		for _, doc := range batch {
			// Extract the _id from the document
			var id interface{}
			switch d := doc.(type) {
			case bson.D:
				for _, elem := range d {
					if elem.Key == "_id" {
						id = elem.Value
						break
					}
				}
			case bson.M:
				id = d["_id"]
			}

			if id != nil {
				// Create a replace model with upsert
				model := mongo.NewReplaceOneModel().
					SetFilter(bson.M{"_id": id}).
					SetReplacement(doc).
					SetUpsert(true)
				models = append(models, model)
			}
		}

		// Execute the bulk write with the upsert models
		if len(models) > 0 {
			_, err := collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
			return err
		}
		return nil
	}

	// Try to insert the batch
	_, err := collection.InsertMany(ctx, batch, options.InsertMany().SetOrdered(false))
	if err == nil {
		return nil
	}

	// If there's an error, check if it's a bulk write error with duplicate key errors
	var bulkWriteErr mongo.BulkWriteException
	if errors.As(err, &bulkWriteErr) {
		// Check if all errors are duplicate key errors
		allDuplicateKeyErrors := true
		for _, writeErr := range bulkWriteErr.WriteErrors {
			if writeErr.Code != 11000 { // 11000 is the code for duplicate key error
				allDuplicateKeyErrors = false
				break
			}
		}

		if allDuplicateKeyErrors {
			// Use upsert for the failed documents
			var models []mongo.WriteModel
			failedIndices := make(map[int]bool)

			// Mark the failed indices
			for _, writeErr := range bulkWriteErr.WriteErrors {
				failedIndices[writeErr.Index] = true
			}

			// Create upsert models for the failed documents
			for i, doc := range batch {
				if failedIndices[i] {
					// Extract the _id from the document
					var id interface{}
					switch d := doc.(type) {
					case bson.D:
						for _, elem := range d {
							if elem.Key == "_id" {
								id = elem.Value
								break
							}
						}
					case bson.M:
						id = d["_id"]
					}

					if id != nil {
						// Create a replace model with upsert
						model := mongo.NewReplaceOneModel().
							SetFilter(bson.M{"_id": id}).
							SetReplacement(doc).
							SetUpsert(true)
						models = append(models, model)
					}
				}
			}

			// Execute the bulk write with the upsert models
			if len(models) > 0 {
				_, err := collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
				return err
			}

			// If no models were created, return nil
			return nil
		}
	}

	// For other errors, return the original error
	return err
}
