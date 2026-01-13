#!/bin/bash
set -e

cd "$(dirname "$0")/.."

# run all go integration tests, pretty-printing any JSON from slog output determined by the SLOG_LEVEL environment variable
go test -tags=integration -v ./... 2>&1 | while IFS= read -r line; do
  echo "$line" | jq . 2>/dev/null || echo "$line"
done
