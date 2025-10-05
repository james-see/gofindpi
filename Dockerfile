# Build stage
FROM golang:1.24-alpine AS builder

# Install required tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    iputils \
    net-tools

WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o gofindpi \
    .

# Final stage
FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    iputils \
    net-tools \
    arp-scan

WORKDIR /app

COPY --from=builder /build/gofindpi .

# Run as non-root user
RUN addgroup -g 1000 scanner && \
    adduser -D -u 1000 -G scanner scanner && \
    chown -R scanner:scanner /app

USER scanner

ENTRYPOINT ["./gofindpi"]