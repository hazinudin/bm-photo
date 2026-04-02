# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags " \
        -X main.version=${VERSION:-dev} \
        -X main.buildTime=${BUILD_TIME:-unknown} \
    " -o /app/bin/server ./cmd/server

# Install migrate tool
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create scripts directory and copy entrypoint
RUN mkdir -p /app/scripts

# Runtime stage
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata postgresql-client wget

RUN adduser -D -g '' appuser

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /go/bin/migrate /app/migrate
COPY scripts/docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

ENV PORT=8080
ENV LOG_LEVEL=info

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
