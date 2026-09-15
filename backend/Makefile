.PHONY: build run-api run-worker migrate seed test vet tidy

build:
	go build ./...

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

migrate:
	go run ./cmd/migrate up

seed:
	go run ./cmd/migrate seed

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy
