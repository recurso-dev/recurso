# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o recurso-api cmd/api/main.go

# Run Stage
FROM alpine:3.18

WORKDIR /app

# Install certificates for external APIs (e.g., Payment Gateway)
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/recurso-api .
COPY --from=builder /app/internal/adapter/templates ./internal/adapter/templates
COPY --from=builder /app/internal/adapter/db/migrations ./internal/adapter/db/migrations

EXPOSE 8080

CMD ["./recurso-api"]
