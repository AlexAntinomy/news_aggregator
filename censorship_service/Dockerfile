FROM golang:1.23.4-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o censorship_service

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/censorship_service .

EXPOSE 8083
CMD ["./censorship_service"] 