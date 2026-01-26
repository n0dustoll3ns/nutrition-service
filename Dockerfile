# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/auth-service .
COPY --from=builder /app/config.yaml .

# Copy USDA food data JSON file
COPY --from=builder /app/usda-importer/FoodData_Central_foundation_food_json_2025-12-18.json /app/data/foods.json

# Create data directory
RUN mkdir -p /app/data

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./auth-service"]
