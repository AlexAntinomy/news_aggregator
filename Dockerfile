FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aggregator ./cmd/aggregator

FROM alpine:latest
COPY --from=builder /app/aggregator /app/
COPY config.prod.json /app/config.json
COPY web /app/web
EXPOSE 8080
CMD ["/app/aggregator"]