## Source
```
$ curl -XGET -u elastic:password 'https://localhost:9200/_cat/indices/my-index?v'  -k
health status index    uuid                   pri rep docs.count docs.deleted store.size pri.store.size dataset.size
yellow open   my-index wu71KyjLQM-y0PjdEs4U4Q   1   1   51256604            0     11.7gb         11.7gb       11.7gb

$ curl -XGET -u elastic:password 'https://localhost:9200/my-index/_count' -k
{"count":5000000,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0}}
```
## Configuration
```
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
    "certificateFingerprint": ""
  },
  "target": {
    "connectionString": "mongodb://uid.us-central1.firestore.goog:443/my-db?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE",
    "database": "my-db"
  },
  "indexMappings": [
    {
      "sourceIndex": "my-index",
      "targetCollection": "my_index",
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
## Start the migration
$./migrate
```
INFO[2025-04-12T09:06:58Z] Loading configuration...
INFO[2025-04-12T09:06:58Z] Starting Elasticsearch to MongoDB migration process
INFO[2025-04-12T09:06:58Z] Starting Elasticsearch to MongoDB migration process
INFO[2025-04-12T09:06:58Z] Connecting to Elasticsearch at [https://localhost:9200]
INFO[2025-04-12T09:06:58Z] TLS is enabled for Elasticsearch connection
INFO[2025-04-12T09:06:58Z] Using username/password authentication for Elasticsearch
INFO[2025-04-12T09:06:58Z] Configuring TLS for Elasticsearch connection
INFO[2025-04-12T09:06:58Z] TLS configuration completed
INFO[2025-04-12T09:06:58Z] Connected to Elasticsearch 8.17.4
INFO[2025-04-12T09:06:58Z] Connecting to MongoDB at mongodb://uid.us-central1.firestore.goog:443/my-db?loadBalanced=true&tls=true&retryWrites=false&authMechanism=MONGODB-OIDC&authMechanismProperties=ENVIRONMENT:gcp,TOKEN_RESOURCE:FIRESTORE
INFO[2025-04-12T09:06:59Z] Processing 1 index patterns concurrently with limit of 2
INFO[2025-04-12T09:06:59Z] Finding indices matching pattern: my-index
INFO[2025-04-12T09:06:59Z] Found 1 indices matching pattern my-index: [my-index]
INFO[2025-04-12T09:06:59Z] Migrating index my-index to collection my_index
INFO[2025-04-12T09:06:59Z] Found 5000000 documents to migrate from index my-index
INFO[2025-04-12T09:06:59Z] Starting 64 workers for parallel document batch processing
INFO[2025-04-12T09:06:59Z] Using sliced scroll with 4 slices for parallel reading from index my-index
INFO[2025-04-12T09:06:59Z] Slice 0/4 has 1249717 documents
INFO[2025-04-12T09:06:59Z] Slice 1/4 has 1249381 documents
INFO[2025-04-12T09:06:59Z] Slice 3/4 has 1250583 documents
INFO[2025-04-12T09:06:59Z] Slice 2/4 has 1250319 documents
INFO[2025-04-12T09:07:00Z] Index my-index progress: 128/5000000 documents (0%)
INFO[2025-04-12T09:07:06Z] Processed 10000/1250319 documents (0.80%)
INFO[2025-04-12T09:07:06Z] Processed 10000/1249381 documents (0.80%)
...
INFO[2025-04-12T09:13:19Z] Processed 1230000/1250319 documents (98.37%)
INFO[2025-04-12T09:13:19Z] Processed 1230000/1249717 documents (98.42%)
INFO[2025-04-12T09:13:19Z] Processed 1230000/1249381 documents (98.45%)
INFO[2025-04-12T09:13:21Z] Processed 1240000/1250583 documents (99.15%)
INFO[2025-04-12T09:13:21Z] Processed 1240000/1250319 documents (99.17%)
INFO[2025-04-12T09:13:21Z] Processed 1240000/1249717 documents (99.22%)
INFO[2025-04-12T09:13:21Z] Processed 1240000/1249381 documents (99.25%)
INFO[2025-04-12T09:13:23Z] Processed 1250000/1250583 documents (99.95%)
INFO[2025-04-12T09:13:23Z] Slice 0/4 completed reading
INFO[2025-04-12T09:13:23Z] Processed 1250000/1250319 documents (99.97%)
INFO[2025-04-12T09:13:23Z] Slice 3/4 completed reading
INFO[2025-04-12T09:13:24Z] Slice 1/4 completed reading
INFO[2025-04-12T09:13:24Z] Slice 2/4 completed reading
INFO[2025-04-12T09:13:24Z] All slices completed reading
INFO[2025-04-12T09:13:24Z] Index my-index progress: 5000000/5000000 documents (100%)
INFO[2025-04-12T09:13:24Z] Migration for index my-index completed successfully! Total documents: 5000000
INFO[2025-04-12T09:13:24Z] Migration completed successfully
INFO[2025-04-12T09:13:24Z] Migration completed in 385.39 seconds
```