# Build Stage
FROM golang:1.25.11-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o identity-api ./cmd/identity-api

# Final Stage
FROM alpine:3.22

WORKDIR /app

# Install certificates
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/identity-api .

# Copy migrations (if needed at runtime, though usually embedded)
# COPY --from=builder /app/migrations ./migrations

# Expose ports
EXPOSE 8080 50051

# Command to run
CMD ["./identity-api"]
