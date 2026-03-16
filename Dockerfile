FROM golang:1.23.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

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

EXPOSE 8085 9595

ENTRYPOINT ["/app/mx_news_bot"]
