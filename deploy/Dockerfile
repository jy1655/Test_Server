# Multi-stage build for optimal image size
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o oculo-pilot-server .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/oculo-pilot-server .

# Copy static files
COPY static/ ./static/

# Create data directory for database
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Set environment variables
ENV DB_PATH=/data/users.db

# Run the application
CMD ["./oculo-pilot-server"]
