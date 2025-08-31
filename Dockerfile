# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pulsedb cmd/pulsedb/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -g '' pulsedb

# Set working directory
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/pulsedb .

# Change ownership
RUN chown pulsedb:pulsedb pulsedb

# Switch to non-root user
USER pulsedb

# Expose ports
EXPOSE 6380 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD nc -z localhost 6380 || exit 1

# Run the binary
CMD ["./pulsedb"]
