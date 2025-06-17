# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make build-base

# Create non-root user
RUN adduser -D -g '' appuser
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -extldflags '-static'" -o /app/main ./cmd/api


FROM alpine:3.19
WORKDIR /app
# Install necessary runtime dependencies and security updates
RUN apk update && \
    apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary and migrations from builder
COPY --from=builder /app/main .
COPY --from=builder /app/db/migrations ./db/migrations

# Set proper permissions
RUN chown -R appuser:appuser /app
USER appuser
EXPOSE 8080
CMD ["./main"]
COPY --from=builder /app/main .
COPY db/migrations ./db/migrations
