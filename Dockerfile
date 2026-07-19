# Build Stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
# Gate the image — and therefore the Cloud Run deploy — on the test suite: a
# failing test fails the build, so a broken commit never reaches production.
# DB-backed tests skip without TEST_DATABASE_URL; unit/logic tests run here.
RUN CGO_ENABLED=1 go test ./...
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o recurso-api ./cmd/api
RUN CGO_ENABLED=1 GOOS=linux go build -o demo-seed ./cmd/demo_seed

# Run Stage
FROM alpine:3.21

WORKDIR /app

# Install certificates for external APIs (e.g., Payment Gateway)
RUN apk --no-cache add ca-certificates \
    && addgroup -S recurso && adduser -S recurso -G recurso

COPY --from=builder --chown=recurso:recurso /app/recurso-api .
COPY --from=builder --chown=recurso:recurso /app/demo-seed .
COPY --from=builder --chown=recurso:recurso /app/internal/adapter/templates ./internal/adapter/templates
COPY --from=builder --chown=recurso:recurso /app/internal/adapter/db/migrations ./internal/adapter/db/migrations

USER recurso

EXPOSE 8080

CMD ["./recurso-api"]
