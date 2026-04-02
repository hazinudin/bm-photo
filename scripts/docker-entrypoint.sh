#!/bin/sh
set -e

echo "Running database migrations..."
/app/migrate -path /app/migrations -database "$DATABASE_URL" up || {
    echo "Migration failed. If this is the first run, the database might not exist yet."
    echo "Continuing anyway..."
}

echo "Starting application..."
exec /app/server "$@"
