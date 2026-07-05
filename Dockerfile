# Build Stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o recurso-api ./cmd/api

# Run Stage
FROM alpine:3.21

WORKDIR /app

# Install certificates for external APIs (e.g., Payment Gateway)
RUN apk --no-cache add ca-certificates \
    && addgroup -S recurso && adduser -S recurso -G recurso

COPY --from=builder --chown=recurso:recurso /app/recurso-api .
COPY --from=builder --chown=recurso:recurso /app/internal/adapter/templates ./internal/adapter/templates
COPY --from=builder --chown=recurso:recurso /app/internal/adapter/db/migrations ./internal/adapter/db/migrations

USER recurso

EXPOSE 8080

CMD ["./recurso-api"]
