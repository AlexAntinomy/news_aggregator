FROM golang:1.23.4 AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aggregator ./cmd/aggregator

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/aggregator /app/
COPY config.json /app/config.json
COPY web /app/web

EXPOSE 8080
CMD ["./aggregator"]
