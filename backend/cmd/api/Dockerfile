# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o build/api ./cmd/api

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/build/api .

RUN adduser -D -u 1000 api && chown -R api:api /app
USER api

ENTRYPOINT ["./api"]
