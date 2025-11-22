# OnyxIRC Server Dockerfile
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

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy configuration
COPY --from=builder /app/configs ./configs

# Create directories for keys and logs
RUN mkdir -p keys logs

# Expose port
EXPOSE 6667

# Run the server
CMD ["./server", "-config", "configs/server.yaml"]
