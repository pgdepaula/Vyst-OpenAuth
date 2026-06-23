#!/bin/bash
echo "Building Vyst Identity..."
go build -o bin/identity-api cmd/identity-api/main.go
go build -o bin/identity-worker cmd/identity-worker/main.go
