#!/usr/bin/env bash
set -euo pipefail

echo "=== build ==="
go build ./...

echo "=== vet ==="
go vet ./...

echo "=== test ==="
go test ./...

echo "=== docs ==="
make docs

echo "=== all checks passed ==="
