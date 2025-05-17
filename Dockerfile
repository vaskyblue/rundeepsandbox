FROM golang:1.20-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Use a smaller image for the final application
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from builder
COPY --from=builder /app/main ./
COPY --from=builder /app/prometheus.yml ./

# Create datasets directory
RUN mkdir -p /app/datasets

# Set environment variables
ENV GIN_MODE=release

# Expose port
EXPOSE 8000

# Run the application
CMD ["./main"] 