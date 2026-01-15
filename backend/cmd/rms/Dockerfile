# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o rms ./cmd/rms

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/rms .

RUN adduser -D -u 1000 rms && chown -R rms:rms /app
USER rms

ENTRYPOINT ["./rms"]
