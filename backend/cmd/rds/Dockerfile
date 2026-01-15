# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build RDS binary
RUN CGO_ENABLED=0 go build -o rds ./cmd/rds

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binary and rules
COPY --from=builder /build/rds .
COPY --from=builder /build/rules ./rules

# Run as non-root
RUN adduser -D -u 1000 rds && chown -R rds:rds /app
USER rds

ENTRYPOINT ["./rds"]
