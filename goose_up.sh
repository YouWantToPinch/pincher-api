#!/usr/bin/bash

set -e

DB_URL="postgresql://postgres:postgres@127.0.0.1:5432/pincher?sslmode=disable"
MAX_RETRIES=10
SLEEP_INTERVAL=1

echo "Waiting for Postgres..."

cd ./sql/schema
# Retry Goose quietly until success, only show output on final failure
for i in $(seq 1 $MAX_RETRIES); do
  if goose postgres "$DB_URL" up >/tmp/goose.log 2>&1; then
    echo "Migrations applied successfully!"
    rm -f /tmp/goose.log
    break
  else
    if [ "$i" -lt "$MAX_RETRIES" ]; then
      echo "Connecting... ($i/$MAX_RETRIES)"
      sleep $SLEEP_INTERVAL
    else
      echo "Goose failed to migrate after $MAX_RETRIES attempts. Logs:"
      cat /tmp/goose.log
      exit 1
    fi
  fi
done
