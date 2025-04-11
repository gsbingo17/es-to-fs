package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gsbingo17/es-to-mongodb/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents a MongoDB connection
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	log      *logger.Logger
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(connectionString, databaseName string, log *logger.Logger) (*MongoDB, error) {
	// Set client options
	clientOptions := options.Client().
		ApplyURI(connectionString).
		SetMaxPoolSize(256).
		SetMinPoolSize(128).
		SetConnectTimeout(30 * time.Second).
		SetSocketTimeout(120 * time.Second)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database
	database := client.Database(databaseName)

	return &MongoDB{
		client:   client,
		database: database,
		log:      log,
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// GetCollection returns a MongoDB collection
func (m *MongoDB) GetCollection(collectionName string) *mongo.Collection {
	return m.database.Collection(collectionName)
}

// ListCollections returns a list of all collection names in the database
func (m *MongoDB) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := m.database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, nil
}

// GetDatabaseName returns the database name
func (m *MongoDB) GetDatabaseName() string {
	return m.database.Name()
}

// GetClient returns the MongoDB client
func (m *MongoDB) GetClient() *mongo.Client {
	return m.client
}
