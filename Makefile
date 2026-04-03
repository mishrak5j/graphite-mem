.PHONY: build run test docker-up docker-down clean lint

BINARY := graphite-mem
BUILD_DIR := ./cmd/graphite-mem

build:
	go build -o bin/$(BINARY) $(BUILD_DIR)

run: build
	./bin/$(BINARY)

run-sse: build
	GRAPHITE_TRANSPORT=sse ./bin/$(BINARY)

test:
	go test ./... -v -race

lint:
	go vet ./...

docker-up:
	docker compose -f scripts/docker-compose.yml up -d

docker-down:
	docker compose -f scripts/docker-compose.yml down

docker-reset: docker-down
	docker compose -f scripts/docker-compose.yml down -v
	docker compose -f scripts/docker-compose.yml up -d

clean:
	rm -rf bin/

tidy:
	go mod tidy
