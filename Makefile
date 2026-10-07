.PHONY: all build run test test-coverage clean help

all: test build

build:
	go build -o server.exe ./cmd/server

run:
	go run ./cmd/server -port 8080 -db traffic.db -web web

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	@if exist server.exe del server.exe
	@if exist traffic.db del traffic.db
	@if exist coverage.out del coverage.out

help:
	@echo "Available commands:"
	@echo "  make build         - Compile production server binary"
	@echo "  make run           - Run server locally on port 8080"
	@echo "  make test          - Run all unit and integration tests"
	@echo "  make test-coverage - Output test coverage metrics"
	@echo "  make clean         - Remove build artifacts and databases"
