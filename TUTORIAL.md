# Tutorial: Migrating from Elasticsearch to Firestore with MongoDB Compatibility

This tutorial guides you through the process of migrating data from Elasticsearch to Firestore with MongoDB compatibility using this tool. 

## Overview

The migration tool performs the following tasks:
- Connects to your Elasticsearch instance
- Reads documents from specified indices
- Maps the documents to the MongoDB/Firestore format
- Writes the documents to Firestore collections
- Handles errors and retries automatically

## Prerequisites

1. **Elasticsearch Source**
   - A running Elasticsearch instance
   - Access credentials (username/password or API key)
   - Knowledge of the indices you want to migrate

2. **Firestore Target**
   - A Google Cloud Firestore instance with MongoDB API enabled
   - Proper authentication setup (OIDC/SCRAM authentication)
   - Sufficient permissions to write to the target database

3. **Migration Tool**
   - The `migrate` executable (built from this repository)
   - A properly configured JSON configuration file

## Step 1: Set Up Firestore with MongoDB Compatibility

1. Create a Firestore instance of Enterprise Edition in Google Cloud Console
2. Enable Firestore with MongoDB compatibility
3. Note your Firestore connection string, which will look similar to:
   ```
   mongodb://[UID].[LOCATION].firestore.goog:443/[DATABASE]?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE
   ```

## Step 2: Configure the Migration

Create a configuration file named `elasticsearch_to_firestore_config.json` with the following structure:

```json
{
  "source": {
    "addresses": ["https://your-elasticsearch-host:9200"],
    "username": "your-username",
    "password": "your-password",
    "apiKey": "",
    "indices": ["index-to-migrate"],
    "tls": true,
    "caCertPath": "",
    "skipVerify": true,
    "certificateFingerprint": "",
    "connectionTimeout": 30,
    "responseTimeout": 60
  },
  "target": {
    "connectionString": "mongodb://[DATABASE_UID].[LOCATION].firestore.goog:443/[DATABASE]?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE",
    "database": "your-database-name"
  },
  "indexMappings": [
    {
      "sourceIndex": "index-to-migrate",
      "targetCollection": "target_collection_name",
      "indexPatternType": "exact"
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
    "fieldMappings": [],
    "excludeFields": ["_score", "_type"]
  },
  "readBatchSize": 1000,
  "writeBatchSize": 128,
  "channelBufferSize": 10,
  "migrationWorkers": 64,
  "concurrentIndices": 4,
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

### Configuration Sections Explained

#### Source Configuration

```json
"source": {
  "addresses": ["https://your-elasticsearch-host:9200"],
  "username": "your-username",
  "password": "your-password",
  "apiKey": "",
  "indices": ["index-to-migrate"],
  "tls": true,
  "caCertPath": "",
  "skipVerify": true,
  "certificateFingerprint": "",
  "connectionTimeout": 30,
  "responseTimeout": 60
}
```

- `addresses`: Array of Elasticsearch server URLs
- `username` and `password`: Your Elasticsearch credentials
- `apiKey`: Alternative to username/password (leave empty if using username/password)
- `indices`: Array of index names or patterns to migrate
- `tls`: Set to `true` if your Elasticsearch uses HTTPS
- `skipVerify`: Set to `true` to skip certificate verification (not recommended for production)
- `connectionTimeout`: Connection timeout in seconds
- `responseTimeout`: Response timeout in seconds

#### Target Configuration (Firestore)

```json
"target": {
  "connectionString": "mongodb://[DATABASE_UID].[LOCATION].firestore.goog:443/[DATABASE]?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE",
  "database": "your-database-name"
}
```

- `connectionString`: Your Firestore MongoDB connection string
- `database`: The database name in Firestore

#### Index Mappings

```json
"indexMappings": [
  {
    "sourceIndex": "index-to-migrate",
    "targetCollection": "target_collection_name",
    "indexPatternType": "exact"
  }
]
```

- `sourceIndex`: The Elasticsearch index name
- `targetCollection`: The Firestore collection name
- `indexPatternType`: How to interpret the index name (options: "exact", "wildcard", "regex")

#### Document Mapping

```json
"documentMapping": {
  "idField": "_id",
  "idType": "string",
  "includeMetadata": true,
  "metadataPrefix": "es_",
  "dateFormat": "strict_date_optional_time",
  "excludeFields": ["_score", "_type"]
}
```

- `idField`: The field to use as document ID
- `idType`: The type of ID field (options: "string", "objectid", "int")
- `includeMetadata`: Whether to include Elasticsearch metadata
- `metadataPrefix`: Prefix for metadata fields
- `fieldMappings`: Array of field mappings for special field types
- `excludeFields`: Fields to exclude from the migration

#### Performance Settings

```json
"readBatchSize": 1000,
"writeBatchSize": 128,
"channelBufferSize": 10,
"migrationWorkers": 64,
"concurrentIndices": 4,
"slicedScrollCount": 4
```

- `readBatchSize`: Number of documents to read in each batch
- `writeBatchSize`: Number of documents to write in each batch
- `channelBufferSize`: Size of the internal buffer for batches
- `migrationWorkers`: Number of parallel workers for writing documents
- `concurrentIndices`: Number of indices to process concurrently
- `slicedScrollCount`: Number of parallel slices for reading from a single index

#### Retry Configuration

```json
"retryConfig": {
  "maxRetries": 5,
  "baseDelayMs": 100,
  "maxDelayMs": 5000,
  "enableBatchSplitting": true,
  "minBatchSize": 10,
  "convertInvalidIds": true
}
```

- `maxRetries`: Maximum number of retry attempts
- `baseDelayMs`: Initial delay between retries (in milliseconds)
- `maxDelayMs`: Maximum delay between retries (in milliseconds)
- `enableBatchSplitting`: Whether to split batches on error
- `minBatchSize`: Minimum batch size when splitting
- `convertInvalidIds`: Whether to convert invalid IDs automatically

## Step 3: Authenticate with Google Cloud

Before running the migration, ensure you're authenticated with Google Cloud:

Option 1:Connect from a Compute Engine VM where the migration tool is running:

```bash
gcloud projects add-iam-policy-binding PROJECT_NAME --member="COMPUTE_ENGINE_DEFAULT_SERVICE_ACCOUNT" --role=roles/datastore.user
```

Use the following format to construct the connection string:
```
mongodb://DATABASE_UID.LOCATION.firestore.goog:443/DATABASE_ID?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE
```

This sets up the necessary credentials for the OIDC authentication used by Firestore.

Option 2: Connect with Username and password(SCRAM)

In the Google Cloud Console, go to the Database page; Select a database from the list of databases. In the navigation menu, click Auth. Click Add User to create user.

Use the following connection string to connect to your database with SCRAM:
```
mongodb://USERNAME:PASSWORD@UID.LOCATION.firestore.goog:443/DATABASE_ID?loadBalanced=true&authMechanism=SCRAM-SHA-256&tls=true&retryWrites=false
```

## Step 4: Run the Migration

Execute the migration tool:

```bash
./migrate -config=elasticsearch_to_firestore_config.json
```

You can also specify a log level:

```bash
./migrate -config=elasticsearch_to_firestore_config.json -log-level=debug
```

Available log levels: debug, info, warn, error

## Step 5: Monitor the Migration

The migration tool provides detailed progress information:

```
INFO[2025-04-12T09:06:58Z] Starting Elasticsearch to MongoDB migration process
INFO[2025-04-12T09:06:58Z] Connecting to Elasticsearch at [https://localhost:9200]
...
INFO[2025-04-12T09:06:59Z] Found 5000000 documents to migrate from index my-index
...
INFO[2025-04-12T09:07:00Z] Index my-index progress: 128/5000000 documents (0%)
...
INFO[2025-04-12T09:13:24Z] Index my-index progress: 5000000/5000000 documents (100%)
INFO[2025-04-12T09:13:24Z] Migration for index my-index completed successfully! Total documents: 5000000
INFO[2025-04-12T09:13:24Z] Migration completed successfully
INFO[2025-04-12T09:13:24Z] Migration completed in 385.39 seconds
```

## Performance Tuning

For large migrations, consider adjusting these parameters:

1. **Increase `migrationWorkers`** (default: 5)
   - For Firestore, values between 32-128 often work well
   - Example: `"migrationWorkers": 64`

2. **Adjust `slicedScrollCount`** (default: 4)
   - Increase for faster reading from Elasticsearch
   - Example: `"slicedScrollCount": 8`

3. **Optimize batch sizes**
   - `readBatchSize`: How many documents to read at once (default: 1000)
   - `writeBatchSize`: How many documents to write at once (default: 128)

## Troubleshooting

### Connection Issues

1. **Elasticsearch Connection Failures**
   - Verify your Elasticsearch credentials
   - Check if TLS settings are correct
   - Try increasing `connectionTimeout` and `responseTimeout`

2. **Firestore Connection Failures**
   - Verify you're authenticated with Google Cloud
   - Check if the Firestore connection string is correct
   - Ensure you have the necessary permissions

### Migration Errors

1. **Document Mapping Errors**
   - Check your `fieldMappings` configuration
   - Consider adding specific mappings for complex field types

2. **Rate Limiting**
   - If you encounter rate limiting, reduce `migrationWorkers`
   - Adjust retry settings: increase `baseDelayMs` and `maxDelayMs`

3. **Memory Issues**
   - Reduce `readBatchSize` and `writeBatchSize`
   - Decrease `channelBufferSize` to lower memory usage

## Example: Complete Migration Configuration

Here's a complete example for migrating a 5 million document index to Firestore:

```json
{
  "source": {
    "addresses": ["https://localhost:9200"],
    "username": "elastic",
    "password": "password",
    "apiKey": "",
    "indices": ["my-index"],
    "tls": true,
    "caCertPath": "",
    "skipVerify": true,
    "certificateFingerprint": "",
    "connectionTimeout": 30,
    "responseTimeout": 60
  },
  "target": {
    "connectionString": "mongodb://2424a090-4d55-41a4-96d1-4a1e52c02775.us-central1.firestore.goog:443/my-database?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE",
    "database": "my-database"
  },
  "indexMappings": [
    {
      "sourceIndex": "my-index",
      "targetCollection": "my_collection",
      "indexPatternType": "exact"
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
      }
    ],
    "excludeFields": ["_score", "_type"]
  },
  "readBatchSize": 1000,
  "writeBatchSize": 128,
  "channelBufferSize": 10,
  "migrationWorkers": 64,
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

## Conclusion

This tutorial has guided you through the process of migrating data from Elasticsearch to Firestore using the MongoDB compatibility API. By following these steps, you can efficiently transfer your data while maintaining the document structure and relationships.

For large migrations, consider running the process in stages, monitoring performance, and adjusting configuration parameters as needed.
