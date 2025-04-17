# Elasticsearch to MongoDB Migration Tool

This Go application migrates data from Elasticsearch indices to MongoDB collections. It performs a one-time migration of data from Elasticsearch to MongoDB with advanced features for mapping and transformation. It especially enables migration from Elasticsearch to Firestore with MongoDB compatibility.

## Prerequisites

- Go 1.21 or later
- Elasticsearch server running and accessible
- MongoDB server running and accessible

## Installation

1. Clone this repository:

   ```bash
   git clone https://github.com/gsbingo17/es-to-fs.git
   cd es-to-fs
   ```

2. Build the application:

   ```bash
   go mod tidy
   go build -o migrate ./cmd/migrate
   ```

## Configuration

Create an `elasticsearch_to_mongodb_config.json` file to define the migration settings. This file specifies the source Elasticsearch connection details, target MongoDB connection details, and mapping configurations.

Here's a basic example:

```json
{
  "source": {
    "addresses": ["http://localhost:9200"],
    "username": "",
    "password": "",
    "indices": ["my-index", "products-*"]
  },
  "target": {
    "connectionString": "mongodb://localhost:27017",
    "database": "es_migration"
  },
  "indexMappings": [
    {
      "sourceIndex": "my-index",
      "targetCollection": "my_collection",
      "indexPatternType": "exact"
    },
    {
      "sourceIndex": "products-*",
      "targetCollection": "products",
      "indexPatternType": "wildcard"
    }
  ],
  "defaultMapping": {
    "type": "direct"
  },
  "documentMapping": {
    "idField": "_id",
    "idType": "string",
    "includeMetadata": true,
    "metadataPrefix": "es_",
    "dateFormat": "strict_date_optional_time",
    "fieldMappings": [
      {
        "sourceField": "created_at",
        "targetField": "created_at",
        "type": "date"
      },
      {
        "sourceField": "location",
        "targetField": "geo_location",
        "type": "geo_point"
      }
    ],
    "excludeFields": ["_score", "_type"]
  },
  "readBatchSize": 1000,
  "writeBatchSize": 100,
  "channelBufferSize": 10,
  "migrationWorkers": 5,
  "concurrentIndices": 2,
  "slicedScrollCount": 4,
  "retryConfig": {
    "maxRetries": 5,
    "baseDelayMs": 100,
    "maxDelayMs": 5000,
    "enableBatchSplitting": true,
    "minBatchSize": 10,
    "convertInvalidIds": true
  }
}
```

## Configuration Options

### Source Configuration
- **addresses**: Array of Elasticsearch server addresses.
- **username**: Elasticsearch username (if authentication is required).
- **password**: Elasticsearch password (if authentication is required).
- **apiKey**: Elasticsearch API key (alternative to username/password).
- **indices**: Array of index names or patterns to migrate.
- **tls**: Enable TLS/HTTPS (default: false).
- **caCertPath**: Path to CA certificate file for server verification.
- **skipVerify**: Skip server certificate verification (not recommended for production).
- **certificateFingerprint**: Certificate fingerprint for verification (SHA256 hex fingerprint).

### Target Configuration
- **connectionString**: MongoDB connection string.
- **database**: Target MongoDB database name.

### Index Mapping Configuration
- **indexMappings**: Array of mappings from Elasticsearch indices to MongoDB collections.
  - **sourceIndex**: Elasticsearch index name or pattern.
  - **targetCollection**: Target MongoDB collection name.
  - **indexPatternType**: Type of pattern matching ("exact", "wildcard", or "regex").
- **defaultMapping**: Default mapping for indices not explicitly mapped.
  - **type**: Mapping type ("direct", "prefix", "suffix", or "transform").
  - **prefix**: Prefix to add to collection names (for "prefix" type).
  - **suffix**: Suffix to add to collection names (for "suffix" type).
  - **transform**: Transform to apply to collection names (for "transform" type).

#### How to Provide sourceIndex in Different Formats

The migration tool supports three different formats for specifying source indices in Elasticsearch:

##### 1. Exact Matching

The simplest format is exact matching, where you specify the exact name of an index.

**Configuration Example:**
```json
"indexMappings": [
  {
    "sourceIndex": "my-index",
    "targetCollection": "my_collection",
    "indexPatternType": "exact"
  }
]
```

**When to use:** When you want to migrate a specific index with an exact name.

##### 2. Wildcard Patterns

Wildcard patterns allow you to match multiple indices using asterisk (*) characters.

**Configuration Example:**
```json
"indexMappings": [
  {
    "sourceIndex": "logs-*",
    "targetCollection": "all_logs",
    "indexPatternType": "wildcard"
  }
]
```

**Supported wildcard patterns:**
- `*` - Matches any number of characters
- `prefix*` - Matches indices starting with "prefix"
- `*suffix` - Matches indices ending with "suffix"
- `prefix*suffix` - Matches indices starting with "prefix" and ending with "suffix"

**When to use:** When you want to match multiple indices following a naming pattern, such as time-based indices (e.g., `logs-2023-*`).

##### 3. Regular Expressions

For more complex matching patterns, you can use regular expressions.

**Configuration Example:**
```json
"indexMappings": [
  {
    "sourceIndex": "logs-\\d{4}-\\d{2}-\\d{2}",
    "targetCollection": "daily_logs",
    "indexPatternType": "regex"
  }
]
```

**When to use:** When you need complex pattern matching that can't be achieved with wildcards.

##### Using Default Mapping

When indices don't have explicit mappings in the `indexMappings` array, the `defaultMapping` configuration is applied:

```json
"defaultMapping": {
  "type": "direct",
  "prefix": "",
  "suffix": "",
  "transform": ""
}
```

**Default mapping types:**
- `"direct"` - Use the index name as the collection name
- `"prefix"` - Add a prefix to the index name (e.g., "mongodb_my-index")
- `"suffix"` - Add a suffix to the index name (e.g., "my-index_collection")
- `"transform"` - Transform the index name using one of these methods:
  - `"lowercase"` - Convert to lowercase
  - `"uppercase"` - Convert to uppercase
  - `"remove-hyphens"` - Remove hyphens from the name

### Document Mapping Configuration
- **documentMapping**: Configuration for mapping Elasticsearch documents to MongoDB documents.
  - **idField**: Field to use as the MongoDB document ID.
  - **idType**: Type of the ID field ("string", "objectid", or "int").
  - **includeMetadata**: Whether to include Elasticsearch metadata fields.
  - **metadataPrefix**: Prefix to add to metadata fields.
  - **dateFormat**: Format for date fields.
  - **fieldMappings**: Array of field mappings.
    - **sourceField**: Field name in Elasticsearch.
    - **targetField**: Field name in MongoDB.
    - **type**: Field type ("date", "geo_point", or "nested").
  - **excludeFields**: Array of fields to exclude from the mapping.

#### Understanding strict_date_optional_time

`strict_date_optional_time` is an Elasticsearch built-in date format that follows the ISO8601 standard. It:

1. Accepts ISO8601-compliant date strings
2. Requires a strict syntax (no deviations from the format)
3. Makes the time component optional

The format accepts these patterns:
- `yyyy-MM-dd'T'HH:mm:ss.SSSZ` - Date with time and timezone
- `yyyy-MM-dd'T'HH:mm:ss.SSS` - Date with time (no timezone, assumes UTC)
- `yyyy-MM-dd'T'HH:mm:ss` - Date with time (no milliseconds)
- `yyyy-MM-dd` - Date only (no time component)

Examples of valid dates in this format:
- `2023-04-15T14:30:45.123Z`
- `2023-04-15T14:30:45.123`
- `2023-04-15T14:30:45`
- `2023-04-15`

During migration, date fields are converted to proper MongoDB date objects rather than being stored as strings, preserving their semantic meaning and allowing for proper date operations in MongoDB.

#### Why You Need to Specify geo_point

Specifying `"type": "geo_point"` in your field mappings is necessary because Elasticsearch and MongoDB handle geographical data differently:

**Elasticsearch geo_point formats:**
1. Object format: `{ "lat": 40.73, "lon": -74.1 }`
2. String format: `"40.73,-74.1"`
3. Array format: `[-74.1, 40.73]` (note: in [lon, lat] order)

**MongoDB GeoJSON format:**
```json
{
  "type": "Point",
  "coordinates": [-74.1, 40.73]  // [longitude, latitude] order
}
```

Without specifying `"type": "geo_point"`, geographical data would be preserved in its original format but not converted to MongoDB's GeoJSON format, making MongoDB's geospatial queries and indexes unusable on this data.

Benefits of proper conversion:
1. **Geospatial Functionality**: MongoDB's geospatial queries will work on the migrated data
2. **Indexing**: You can create geospatial indexes on the converted fields
3. **Consistency**: All geo_point data is converted to a consistent format regardless of the original format in Elasticsearch

### Performance Configuration
- **readBatchSize**: Number of documents to read in a batch.
- **writeBatchSize**: Number of documents to write in a batch.
- **channelBufferSize**: Size of the channel buffer for batches.
- **migrationWorkers**: Number of worker goroutines for batch processing.
- **concurrentIndices**: Number of indices to process concurrently.
- **slicedScrollCount**: Number of slices for parallel reading within a single index (default: 4).

### Retry Configuration
- **retryConfig**: Configuration for retry mechanisms.
  - **maxRetries**: Maximum number of retries.
  - **baseDelayMs**: Base delay in milliseconds.
  - **maxDelayMs**: Maximum delay in milliseconds.
  - **enableBatchSplitting**: Enable batch splitting for contention errors.
  - **minBatchSize**: Minimum batch size for splitting.
  - **convertInvalidIds**: Automatically convert invalid _id types to string.

## Usage

To perform a migration:

```bash
./migrate
```

Additional options:

```bash
./migrate -help
```

This will display all available command-line options:

```
Options:
  -config string
        Path to configuration file (default "elasticsearch_to_mongodb_config.json")
  -log-level string
        Log level: debug, info, warn, error (default "info")
  -help
        Display this help information
```

## Key Features

### Document Mapping

The tool provides a flexible document mapping system that can:

1. **Transform Field Types**:
   - Convert Elasticsearch date strings to MongoDB Date objects
   - Transform geo_point fields to MongoDB GeoJSON format
   - Handle nested objects and arrays

2. **ID Field Handling**:
   - Use Elasticsearch IDs directly
   - Generate new MongoDB ObjectIDs
   - Convert string IDs to integers

3. **Metadata Handling**:
   - Include or exclude Elasticsearch metadata fields
   - Add prefixes to metadata fields

### Index Pattern Matching

The tool supports different types of index pattern matching:

1. **Exact Matching**: Match indices with exact names
2. **Wildcard Matching**: Match indices using wildcard patterns (e.g., "logs-*")
3. **Regex Matching**: Match indices using regular expressions

### Parallel Processing

The application implements parallelism at multiple levels:

1. **Pattern-Level Parallelism**:
   - Multiple index patterns are processed concurrently
   - Default limit of 2 concurrent patterns
   - Improves overall migration speed when using multiple index patterns

2. **Index-Level Parallelism**:
   - Multiple indices are processed concurrently
   - Controlled by the `concurrentIndices` parameter

3. **Intra-Index Parallelism (Sliced Scroll)**:
   - Each index is divided into multiple slices that are read in parallel
   - Uses Elasticsearch's built-in sliced scroll feature
   - Controlled by the `slicedScrollCount` parameter
   - Significantly improves read performance for large indices

4. **Batch-Level Parallelism**:
   - Documents are processed in batches by multiple workers
   - Controlled by the `migrationWorkers` parameter

### Robust Error Handling

The application includes a sophisticated retry mechanism:

1. **Exponential Backoff**: Retries use exponential backoff with jitter
2. **Batch Splitting**: For contention errors, batches are progressively split
3. **Automatic ID Conversion**: Invalid ID types can be automatically converted to strings

## Project Structure

- `cmd/migrate/`: Contains the main application entry point
- `pkg/common/`: Common utilities and shared functionality
- `pkg/config/`: Configuration handling
- `pkg/db/`: MongoDB connection and operations
- `pkg/es/`: Elasticsearch client
- `pkg/logger/`: Logging utilities
- `pkg/migration/`: Migration logic and implementation
