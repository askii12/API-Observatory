.PHONY: fmt test vet run-api infra-up infra-down

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

vet:
	go vet ./...

run-api:
	go run ./cmd/api

infra-up:
	docker compose up -d postgres

infra-down:
	docker compose down
