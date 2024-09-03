# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o transaction-processor ./cmd/api

# Final stage
FROM alpine:latest

# Add non-root user
RUN adduser -D appuser

# Install CA certificates
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/transaction-processor .

# Copy the migrations folder
COPY --from=builder /app/migrations ./migrations

# Use non-root user
USER appuser

# Expose port 4000
EXPOSE 4000

# Use environment variables for configuration
ENV PORT=4000 \
    ENV=development

# Run the binary
CMD ["./transaction-processor"]