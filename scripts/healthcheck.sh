#!/bin/sh
# Health check script for HomeLog backend

set -e

# Check if the API is responding
if ! wget --quiet --tries=1 --spider http://localhost:8080/health; then
    echo "ERROR: API health check failed"
    exit 1
fi

# Check that the database file exists
if [ ! -f /app/data/homelog.db ]; then
    echo "ERROR: Database file not found at /app/data/homelog.db"
    exit 1
fi

echo "Health check passed"
exit 0
