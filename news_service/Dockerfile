FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o news_service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/news_service .
EXPOSE 8082
CMD ["./news_service"] 