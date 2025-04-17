#!/bin/bash

# Script to run Elasticsearch to Firestore migration
# This script handles authentication and runs the migration

# Configuration file path (default or specified by user)
CONFIG_FILE=${1:-"firestore_migration_config.json"}
LOG_LEVEL=${2:-"info"}

# Check if configuration file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file '$CONFIG_FILE' not found."
    echo "Usage: $0 [config_file] [log_level]"
    echo "Example: $0 firestore_migration_config.json debug"
    exit 1
fi

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "Error: gcloud CLI not found. Please install Google Cloud SDK."
    echo "Visit: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check if migrate executable exists
if [ ! -f "./migrate" ]; then
    echo "Error: 'migrate' executable not found in current directory."
    echo "Please build the migration tool first."
    exit 1
fi

echo "=== Elasticsearch to Firestore Migration ==="
echo "Configuration file: $CONFIG_FILE"
echo "Log level: $LOG_LEVEL"
echo

# Authenticate with Google Cloud (for Firestore OIDC)
echo "Authenticating with Google Cloud..."
gcloud auth application-default login

# Check authentication status
if [ $? -ne 0 ]; then
    echo "Error: Failed to authenticate with Google Cloud."
    exit 1
fi

echo "Authentication successful."
echo

# Run the migration
echo "Starting migration..."
echo "Press Ctrl+C to gracefully stop the migration."
echo

# Execute the migration tool
./migrate -config="$CONFIG_FILE" -log-level="$LOG_LEVEL"

# Check exit status
if [ $? -eq 0 ]; then
    echo "Migration completed successfully."
else
    echo "Migration completed with errors. Check the logs for details."
    exit 1
fi
