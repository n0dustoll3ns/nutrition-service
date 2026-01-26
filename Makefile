.PHONY: build run test clean docker-up docker-down migrate

# Build the application
build:
	go build -o auth-service ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f auth-service

# Start Docker containers
docker-up:
	docker-compose up -d

# Stop Docker containers
docker-down:
	docker-compose down

# Run database migrations
migrate:
	@echo "Running migrations..."
	# TODO: Add migration command

# Run with hot reload (using air)
dev:
	air

# Generate swagger docs
swagger:
	@echo "Generating swagger documentation..."
	# TODO: Add swagger generation

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
