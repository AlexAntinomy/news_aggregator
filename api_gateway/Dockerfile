FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api_gateway .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/api_gateway .
EXPOSE 8080
CMD ["./api_gateway"] 