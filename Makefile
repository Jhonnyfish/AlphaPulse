APP_NAME=alphapulse

.PHONY: build run migrate tidy

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server

migrate:
	go run ./cmd/server -migrate-only

tidy:
	go mod tidy
