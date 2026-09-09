FROM golang:1.23.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
# proxy.golang.org drops connections often enough that a single attempt makes the whole deploy
# flaky: the failure is a transport error mid-stream, not a missing module, and the next attempt
# usually succeeds.
RUN for attempt in 1 2 3; do \
        go mod download && break; \
        echo "go mod download failed (attempt ${attempt}), retrying"; \
        sleep $((attempt * 5)); \
    done; \
    go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mx_news_bot ./cmd/mx

FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    chromium \
    nss \
    poppler-utils \
    tzdata

ENV CHROME_BIN=/usr/bin/chromium-browser
WORKDIR /app

COPY --from=builder /out/mx_news_bot /app/mx_news_bot
# Migrations ship alongside the binary that applies them, so `-migrate-up` can never run a set
# of migrations from a different build than the code that expects them.
COPY --from=builder /src/infra/migrations /app/migrations

EXPOSE 8085 9595

ENTRYPOINT ["/app/mx_news_bot"]
